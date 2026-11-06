package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/copywriter"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/logs"
	"github.com/Pastir/footballdata/renderer"
	"github.com/Pastir/footballdata/viewmodel"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

var ensureSchemaOnce sync.Once

func getStorage(ctx context.Context) *PGStorage {
	if db.DBx.PGxPool == nil {
		return nil
	}
	s := NewPGStorage(db.DBx.PGxPool)
	ensureSchemaOnce.Do(func() {
		_ = s.EnsureSchema(ctx)
	})
	return s
}

// Dispatch координирует запуск публикации по переданному типу контента.
func Dispatch(ctx context.Context, taskData config.TaskData) error {
	log := logs.Logs["access"]
	if log == nil {
		l, _ := zap.NewDevelopment()
		log = l
	}

	pubType := taskData.GetType()
	if pubType == "" {
		return fmt.Errorf("empty -type parameter. Use: table, roundFixtures, roundResults, previewMatch, roundDigest")
	}

	log.Info("dispatcher start",
		zap.String("task", taskData.Name()),
		zap.String("type", pubType),
		zap.Int("leagueId", taskData.GetLeagueId()),
		zap.Bool("publish", taskData.GetPublish()),
	)

	switch pubType {
	case "table", "standings":
		return dispatchTable(ctx, taskData, log)
	case "roundFixtures", "fixtures":
		return dispatchRoundFixtures(ctx, taskData, log)
	case "weekendHub", "hub", "schedule5Leagues", "scheduleAll", "scheduleTop5", "weekendSchedule", "scheduleOnly", "postVariants", "variants", "nativeTable", "richTable", "tableNative", "albumOnly", "weekendAlbum", "topMatches", "top5Matches", "centralMatches", "topMatchesAlbum", "topMatchesTable", "topMatchesNative", "topMatchesHub":
		return dispatchWeekendHub(ctx, taskData, log)
	case "roundResults", "results", "tuesdaySummary", "roundLeaders", "postRound":
		return dispatchRoundResults(ctx, taskData, log)
	case "previewMatch", "preview":
		return dispatchPreviewMatch(ctx, taskData, log)
	case "roundDigest", "digest", "interactive":
		return dispatchRoundDigest(ctx, taskData, log)
	case "leaders", "topscorers":
		return dispatchLeaders(ctx, taskData, log)
	case "leadersEuro", "euroLeaders":
		return dispatchLeadersEuro(ctx, taskData, log)
	case "facts", "streaks", "factCard":
		return dispatchFacts(ctx, taskData, log)
	case "recalculateStreaks", "computeStreaks":
		return dispatchRecalculateStreaks(ctx, taskData, log)
	case "catalog", "leagues", "activeLeagues":
		return dispatchCatalog(ctx, taskData, log)
	case "syncAll", "syncData", "generateAll":
		return dispatchSyncAll(ctx, taskData, log)
	case "express", "accumulator", "smartBet", "smartExpress", "expressBet":
		return dispatchExpress(ctx, taskData, log)
	case "banter", "anomaly", "anomalyCard", "banterCard":
		return dispatchBanter(ctx, taskData, log)
	default:
		return fmt.Errorf("unsupported -type '%s'. Valid types: table, roundFixtures, schedule5Leagues, roundResults, express, previewMatch, roundDigest, leaders, facts, recalculateStreaks, catalog, syncAll, banter", pubType)
	}
}



// roundExportPayload описывает структуру выгрузки данных тура для Telegram Mini App.
type roundExportPayload = analytics.RoundExportPayload




// dispatchPreviewMatch подготавливает и публикует карточку превью матча.
func dispatchPreviewMatch(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	fixtureIDStr := taskData.GetFixtureId()
	if fixtureIDStr == "" {
		return fmt.Errorf("-fixturesId is required for previewMatch (e.g. -fixturesId 18535518)")
	}

	fixtureID, err := strconv.Atoi(fixtureIDStr)
	if err != nil {
		return fmt.Errorf("invalid fixture ID '%s': %w", fixtureIDStr, err)
	}

	leagueID := taskData.GetLeagueId()
	if leagueID == 0 {
		leagueID = 8
	}

	outputPath := taskData.GetOutput()
	if outputPath == "" {
		outputPath = fmt.Sprintf("match_preview_%d.png", fixtureID)
	}

	log.Info("previewMatch initiated", zap.Int("fixtureId", fixtureID), zap.Int("leagueId", leagueID))

	caption := fmt.Sprintf("🔥 <b>Матч-центр: Превью матча #%d</b>\nЛига ID: %d", fixtureID, leagueID)
	fmt.Printf("Карточка матча #%d подготовлена для рендера\n", fixtureID)

	if !taskData.GetPublish() {
		return nil
	}

	return publishToTelegram(ctx, taskData, PubTypePrediction, leagueID, 0, 0, fixtureID, fmt.Sprintf("Превью матча #%d", fixtureID), caption, outputPath, log)
}

// publishToTelegram отправляет контент в Telegram и сохраняет запись в БД.
func publishToTelegram(
	ctx context.Context,
	taskData config.TaskData,
	pubType PublicationType,
	leagueID, seasonID, roundID, fixtureID int,
	title, caption, filePath string,
	log *zap.Logger,
) error {
	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return fmt.Errorf("telegram configuration missing (BotToken or ChannelID is empty)")
	}

	channelID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
	storage := getStorage(ctx)

	// Проверка идемпотентности: не публиковать повторно без флага -force
	if storage != nil && !taskData.GetForce() {
		already, err := storage.IsAlreadyPublished(ctx, channelID, pubType, leagueID, roundID)
		if err == nil && already {
			log.Info("already published, skipping duplicate publication (use -force to override)",
				zap.String("type", string(pubType)),
				zap.Int("league_id", leagueID),
				zap.Int("round_id", roundID),
			)
			fmt.Printf("ℹ️ Материал (%s, лига %d, тур %d) уже был опубликован в канале %s. Для повторной отправки укажите флаг -force\n", pubType, leagueID, roundID, tg.ChannelID)
			return nil
		}
	}

	pubCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	log.Info("publishing to telegram",
		zap.String("channel", tg.ChannelID),
		zap.String("file", filePath),
		zap.String("type", string(pubType)),
	)

	var (
		msgID  int64
		pubErr error
	)

	if taskData.GetAsDocument() {
		msgID, pubErr = SendDocumentWithResult(pubCtx, tg.BotToken, tg.ChannelID, caption, filePath)
	} else {
		msgID, pubErr = SendPhotoWithResult(pubCtx, tg.BotToken, tg.ChannelID, caption, filePath)
	}

	if pubErr != nil {
		return fmt.Errorf("telegram send failed: %w", pubErr)
	}

	log.Info("published to telegram successfully",
		zap.String("channel", tg.ChannelID),
		zap.Int64("message_id", msgID),
	)
	fmt.Printf("✅ Успешно опубликовано в Telegram (%s)! Message ID: %d\n", tg.ChannelID, msgID)

	// Запись в хранилище истории telegram_publications
	if storage != nil {
		pub := &Publication{
			ChannelID:       channelID,
			MessageID:       &msgID,
			PublicationType: pubType,
			LeagueID:        &leagueID,
			SeasonID:        &seasonID,
			RoundID:         &roundID,
			Title:           title,
			TextContent:     caption,
			MediaURL:        filePath,
			MediaType:       MediaTypePhoto,
			Status:          StatusPublished,
		}
		if fixtureID > 0 {
			pub.FixtureID = &fixtureID
		}
		if taskData.GetAsDocument() {
			pub.MediaType = MediaTypeDocument
		}

		if _, err := storage.Create(ctx, pub); err != nil {
			log.Warn("failed to record publication to db", zap.Error(err))
		}
	}

	return nil
}

// dispatchRoundDigest формирует и публикует интерактивный дайджест тура:
// 1. Рендерит турнирную таблицу и расписание в PNG.
// 2. Сохраняет их в постоянное медиа-хранилище SSD: /extssd/.../{league}/{season}/{round}/*.png.
// 3. Формирует текст превью с акцентом на Матч дня (Match of the Day).
// 4. Добавляет Telegram WebApp кнопки для нативного просмотра в мессенджере.
func dispatchRoundDigest(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	leagueID := taskData.GetLeagueId()
	if leagueID == 0 {
		leagueID = 8 // Default: Premier League
	}

	info, err := analytics.GetLeagueSeasonInfo(ctx, pool, leagueID)
	if err != nil {
		return fmt.Errorf("GetLeagueSeasonInfo: %w", err)
	}
	info.Locale = taskData.GetLocale()

	if taskData.GetRoundId() > 0 {
		info.CurrRoundID = taskData.GetRoundId()
		info.NextRoundID = taskData.GetRoundId()
	}

	matches, err := analytics.GetNextRoundFixtures(ctx, pool, &info)
	if err != nil {
		return fmt.Errorf("GetNextRoundFixtures: %w", err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no upcoming round fixtures found for league %d", leagueID)
	}

	// Определение ID и номера тура
	roundID := info.NextRoundID
	if roundID == 0 && len(matches) > 0 {
		roundID = matches[0].RoundID
	}
	roundNum := info.NextRoundName
	if roundNum == "" && len(matches) > 0 {
		roundNum = strconv.Itoa(roundID)
	}

	leagueName := "АПЛ"
	if info.LeagueID != 8 && info.LeagueName != "" {
		leagueName = info.LeagueName
	}

	// Настройка путей хранения медиа
	mediaDir := "/extssd/footballstats/public/media"
	if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
		mediaDir = config.Config.WebApp.MediaDir
	}

	roundDir := filepath.Join(mediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID), strconv.Itoa(roundID))
	if err := os.MkdirAll(roundDir, 0755); err != nil {
		log.Warn("cannot create round media dir on external storage, fallback to local", zap.Error(err))
	}

	standingsPath := filepath.Join(roundDir, "standings.png")
	fixturesPath := filepath.Join(roundDir, "round_fixtures.png")

	// 1. Рендер Турнирной Таблицы
	rows, err := analytics.GetCurrentStandings(ctx, pool, &info)
	if err != nil {
		return fmt.Errorf("GetCurrentStandings: %w", err)
	}
	stats := analytics.GetKeyStats(rows)
	insight := analytics.BuildInsight(rows)

	tableVM := viewmodel.BuildTableViewModel(info, rows, nil, nil, nil, insight, stats)
	tableTmpl := filepath.Join("template", "football_table_template.html")
	log.Info("rendering standings PNG for digest", zap.String("output", standingsPath))
	if err = renderer.RenderStandingsPNG(tableVM, tableTmpl, standingsPath); err != nil {
		return fmt.Errorf("RenderStandingsPNG: %w", err)
	}
	_ = copyFile(standingsPath, "standings.png")

	// 2. Рендер Расписания Тура
	candidates := analytics.BuildMatchCandidates(matches)
	_, _ = analytics.GetOrComputeMatchOfRound(ctx, pool, &info, candidates)
	fixturesVM := viewmodel.BuildRoundFixturesViewModel(info, matches, taskData.GetTimezone())
	fixturesTmpl := filepath.Join("template", "round_fixtures_template.html")
	log.Info("rendering round fixtures PNG for digest", zap.String("output", fixturesPath))
	if err = renderer.RenderRoundFixturesPNG(fixturesVM, fixturesTmpl, fixturesPath); err != nil {
		return fmt.Errorf("RenderRoundFixturesPNG: %w", err)
	}
	_ = copyFile(fixturesPath, "round_fixtures.png")

	// 3. Рендер Шапки Матча Тура (Master Hybrid)
	headerPath := filepath.Join(roundDir, "match_header.png")
	var hasHeaderPNG bool
	var featuredMatch *viewmodel.FixtureMatchItem = fixturesVM.FeaturedMatch

	if featuredMatch != nil {
		fm := *featuredMatch
		headerVM := viewmodel.MatchHeaderViewModel{
			LeagueName:   leagueName,
			RoundTitle:   fmt.Sprintf("ТУР %s", roundNum),
			BadgeTitle:   "МАТЧ ТУРА",
			HomeTeamName: strings.ToUpper(fm.HomeTeamName),
			HomeTeamCode: getTeamShortCode(fm.HomeTeamName),
			HomeTeamLogo: fm.HomeTeamLogo,
			AwayTeamName: strings.ToUpper(fm.AwayTeamName),
			AwayTeamCode: getTeamShortCode(fm.AwayTeamName),
			AwayTeamLogo: fm.AwayTeamLogo,
			KickoffTime:  fm.KickoffTime,
			Venue:        fm.Venue,
			Timezone:     taskData.GetTimezone(),
			HomeRank:     fm.HomeRank,
			AwayRank:     fm.AwayRank,
			OddsHome:     fm.OddsHome,
			OddsDraw:     fm.OddsDraw,
			OddsAway:     fm.OddsAway,
			ProbHome:     fm.ProbHome,
			ProbDraw:     fm.ProbDraw,
			ProbAway:     fm.ProbAway,
		}
		headerTmpl := filepath.Join("template", "match_header_template.html")
		log.Info("rendering match header PNG for digest", zap.String("output", headerPath))
		if err := renderer.RenderMatchHeaderPNG(headerVM, headerTmpl, headerPath); err != nil {
			log.Warn("RenderMatchHeaderPNG failed, falling back to text", zap.Error(err))
		} else {
			_ = copyFile(headerPath, "match_header.png")
			hasHeaderPNG = true
			log.Info("match header PNG rendered successfully", zap.String("file", headerPath))
		}
	}



	// 4. Формирование нативной ссылки на Mini App (t.me/<bot>/<short_name>)
	botUser := "football_data_lab_bot"
	shortName := "data_lab"
	if config.Config.WebApp != nil {
		if config.Config.WebApp.BotUsername != "" {
			botUser = config.Config.WebApp.BotUsername
		}
		if config.Config.WebApp.BotShortName != "" {
			shortName = config.Config.WebApp.BotShortName
		}
	}

	tableURL := fmt.Sprintf("https://t.me/%s/%s?startapp=%d_table", botUser, shortName, info.LeagueID)
	fixturesURL := fmt.Sprintf("https://t.me/%s/%s?startapp=%d_matches", botUser, shortName, info.LeagueID)

	// В новом нативном стиле Telegram ссылки на Mini App размещаются внутри текста,
	// а инлайн-клавиатура не передается (nil) для отображения нативной плашки комментариев Telegram.
	var keyboard *InlineKeyboardMarkup = nil

	// 5. Формирование текста поста с динамическим сравнением
	var postText string
	if featuredMatch != nil {
		fm := *featuredMatch

		// Извлечение голов из standings
		var homeGF, homeGA, awayGF, awayGA float64
		var hasHomeGoals, hasAwayGoals bool
		for _, r := range rows {
			if r.TeamID == fm.HomeTeamID {
				if r.Played > 0 {
					homeGF = float64(r.GF) / float64(r.Played)
					homeGA = float64(r.GA) / float64(r.Played)
					hasHomeGoals = true
				}
			}
			if r.TeamID == fm.AwayTeamID {
				if r.Played > 0 {
					awayGF = float64(r.GF) / float64(r.Played)
					awayGA = float64(r.GA) / float64(r.Played)
					hasAwayGoals = true
				}
			}
		}

		metrics := buildMatchComparisonMetrics(fm, homeGF, homeGA, hasHomeGoals, awayGF, awayGA, hasAwayGoals)
		hCode := getTeamShortCode(fm.HomeTeamName)
		aCode := getTeamShortCode(fm.AwayTeamName)
		tableStr := copywriter.FormatDynamicComparisonTable(hCode, aCode, fm.OddsHome, fm.OddsDraw, fm.OddsAway, metrics)
		intrigueStr := copywriter.GenerateMatchIntrigue(fm, nil)

		postText = copywriter.BuildInteractiveDigestCaption(tableStr, intrigueStr)
	} else {
		postText = copywriter.BuildInteractivePostText(info, fixturesVM, matches, leagueName, roundNum)
	}
	postText = fmt.Sprintf("%s\n\n<a href=\"%s\">Все матчи</a> · <a href=\"%s\">Турнирная таблица</a>", postText, fixturesURL, tableURL)

	if !taskData.GetPublish() {
		fmt.Printf("Интерактивный дайджест готов (-publish=false).\nТаблица: %s\nРасписание: %s\nШапка: %s\nТекст:\n%s\n", standingsPath, fixturesPath, headerPath, postText)
		return nil
	}

	// 6. Публикация в Telegram
	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return fmt.Errorf("telegram configuration missing (BotToken or ChannelID is empty)")
	}

	channelID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
	storage := getStorage(ctx)

	// Проверка идемпотентности: защита от случайных дубликатов публикаций
	if storage != nil && !taskData.GetForce() {
		already, err := storage.IsAlreadyPublished(ctx, channelID, PubTypeRoundDigest, info.LeagueID, roundID)
		if err == nil && already {
			log.Info("round digest already published, skipping duplicate publication (use -force to override)",
				zap.String("type", string(PubTypeRoundDigest)),
				zap.Int("league_id", info.LeagueID),
				zap.Int("round_id", roundID),
			)
			fmt.Printf("ℹ️ Интерактивный дайджест (лига %d, тур %d) уже был опубликован в канале %s. Для повторной отправки укажите флаг -force\n", info.LeagueID, roundID, tg.ChannelID)
			return nil
		}
	}

	pubCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	log.Info("publishing interactive digest to telegram",
		zap.String("channel", tg.ChannelID),
		zap.String("league", leagueName),
		zap.String("round", roundNum),
		zap.Bool("hasPhotoHeader", hasHeaderPNG),
	)

	var (
		msgID  int64
		pubErr error
	)
	if hasHeaderPNG {
		msgID, pubErr = SendPhotoWithKeyboard(pubCtx, tg.BotToken, tg.ChannelID, postText, headerPath, keyboard)
	} else {
		msgID, pubErr = SendTextMessageWithKeyboard(pubCtx, tg.BotToken, tg.ChannelID, postText, keyboard)
	}
	if pubErr != nil {
		return fmt.Errorf("SendToTelegram failed: %w", pubErr)
	}

	log.Info("interactive digest published successfully",
		zap.String("channel", tg.ChannelID),
		zap.Int64("message_id", msgID),
	)
	fmt.Printf("✅ Интерактивный дайджест опубликован в Telegram (%s)! Message ID: %d\n", tg.ChannelID, msgID)

	// Запись в БД
	if storage != nil {
		pub := &Publication{
			ChannelID:       channelID,
			MessageID:       &msgID,
			PublicationType: PubTypeRoundDigest,
			LeagueID:        &info.LeagueID,
			SeasonID:        &info.SeasonID,
			RoundID:         &roundID,
			Title:           fmt.Sprintf("%s — %s: Интерактивный дайджест", leagueName, roundNum),
			TextContent:     postText,
			MediaURL:        standingsPath,
			MediaType:       MediaTypePhoto,
			Status:          StatusPublished,
		}
		if hasHeaderPNG {
			pub.MediaURL = headerPath
		}
		if _, saveErr := storage.Create(ctx, pub); saveErr != nil {
			log.Warn("failed to record publication to database", zap.Error(saveErr))
		}
	}

	return nil
}

func buildMatchComparisonMetrics(
	fm viewmodel.FixtureMatchItem,
	homeGF, homeGA float64, hasHomeGoals bool,
	awayGF, awayGA float64, hasAwayGoals bool,
) []copywriter.MatchMetricRow {
	var metrics []copywriter.MatchMetricRow

	// 1. Форма (до 5 матчей)
	if len(fm.HomeForm) > 0 || len(fm.AwayForm) > 0 {
		metrics = append(metrics, copywriter.MatchMetricRow{
			Label:   "форма",
			HomeVal: copywriter.FormatForm(fm.HomeForm, 5, ""),
			AwayVal: copywriter.FormatForm(fm.AwayForm, 5, ""),
		})
	}

	// 2. Место
	if fm.HomeRank > 0 || fm.AwayRank > 0 {
		hRank := "-"
		if fm.HomeRank > 0 {
			hRank = fmt.Sprintf("%d", fm.HomeRank)
		}
		aRank := "-"
		if fm.AwayRank > 0 {
			aRank = fmt.Sprintf("%d", fm.AwayRank)
		}
		metrics = append(metrics, copywriter.MatchMetricRow{
			Label:   "место",
			HomeVal: hRank,
			AwayVal: aRank,
		})
	}

	// 3. Очки
	if fm.HomePoints > 0 || fm.AwayPoints > 0 {
		metrics = append(metrics, copywriter.MatchMetricRow{
			Label:   "очки",
			HomeVal: fmt.Sprintf("%d", fm.HomePoints),
			AwayVal: fmt.Sprintf("%d", fm.AwayPoints),
		})
	}

	// 4. Голы/м
	if hasHomeGoals && hasAwayGoals {
		metrics = append(metrics, copywriter.MatchMetricRow{
			Label:   "голы/м",
			HomeVal: fmt.Sprintf("%.1f", homeGF),
			AwayVal: fmt.Sprintf("%.1f", awayGF),
		})
	}

	// 5. Пропущ/м
	if hasHomeGoals && hasAwayGoals {
		metrics = append(metrics, copywriter.MatchMetricRow{
			Label:   "пропущ/м",
			HomeVal: fmt.Sprintf("%.1f", homeGA),
			AwayVal: fmt.Sprintf("%.1f", awayGA),
		})
	}

	return metrics
}

func getTeamShortCode(name string) string {
	name = strings.TrimSpace(name)
	switch strings.ToLower(name) {
	case "liverpool":
		return "LIV"
	case "manchester city":
		return "MCI"
	case "manchester united":
		return "MUN"
	case "arsenal":
		return "ARS"
	case "chelsea":
		return "CHE"
	case "tottenham hotspur", "tottenham":
		return "TOT"
	case "brentford":
		return "BRE"
	case "aston villa":
		return "AVL"
	case "newcastle united", "newcastle":
		return "NEW"
	case "brighton & hove albion", "brighton":
		return "BHA"
	case "west ham united", "west ham":
		return "WHU"
	case "everton":
		return "EVE"
	case "fulham":
		return "FUL"
	case "crystal palace":
		return "CRY"
	case "wolverhampton wanderers", "wolves":
		return "WOL"
	case "afc bournemouth", "bournemouth":
		return "BOU"
	case "nottingham forest":
		return "NFO"
	case "ipswich town", "ipswich":
		return "IPS"
	case "leicester city", "leicester":
		return "LEI"
	case "southampton":
		return "SOU"
	}
	runes := []rune(name)
	if len(runes) <= 3 {
		return strings.ToUpper(name)
	}
	return strings.ToUpper(string(runes[:3]))
}

func copyFile(src, dst string) error {
	srcClean := filepath.Clean(src)
	dstClean := filepath.Clean(dst)
	if srcClean == dstClean {
		return nil
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if dstInfo, err := os.Stat(dst); err == nil {
		if os.SameFile(srcInfo, dstInfo) {
			return nil
		}
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Атомарная запись через .tmp, чтобы исключить затирание файла в 0 байт
	tmpDst := dst + ".tmp"
	out, err := os.Create(tmpDst)
	if err != nil {
		return err
	}

	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmpDst)
		return err
	}
	out.Close()

	return os.Rename(tmpDst, dst)
}

func syncViewerFiles(publicDir string) {
	if publicDir == "" {
		return
	}

	// Если publicDir совпадает с web/viewer или является симлинком на него (Zero-Copy Deployment),
	// копирование не требуется — веб-сервер уже отдает файлы напрямую из web/viewer.
	if absPublic, err := filepath.Abs(publicDir); err == nil {
		if absViewer, vErr := filepath.Abs(filepath.Join("web", "viewer")); vErr == nil {
			realPublic, rErr := filepath.EvalSymlinks(absPublic)
			realViewer, rvErr := filepath.EvalSymlinks(absViewer)
			if rErr == nil && rvErr == nil && realPublic == realViewer {
				return
			}
		}
	}

	viewerDir := filepath.Join(publicDir, "viewer")
	_ = os.MkdirAll(viewerDir, 0755)

	files := []string{"index.html", "app.js", "style.css", "round_data.json", "active_leagues.json", "leaders.json"}
	for _, f := range files {
		src := filepath.Join("web", "viewer", f)
		if _, statErr := os.Stat(src); statErr == nil {
			dst := filepath.Join(viewerDir, f)
			_ = copyFile(src, dst)
			_ = os.Chmod(dst, 0644)
			_ = copyFile(src, filepath.Join(publicDir, f))
			_ = os.Chmod(filepath.Join(publicDir, f), 0644)
		}
	}
	rootIndex := filepath.Join(publicDir, "index.html")
	_ = copyFile(filepath.Join("web", "viewer", "index.html"), rootIndex)
	_ = os.Chmod(rootIndex, 0644)
	_ = copyFile(filepath.Join("web", "viewer", "app.js"), filepath.Join(publicDir, "app.js"))
	_ = os.Chmod(filepath.Join(publicDir, "app.js"), 0644)
	_ = copyFile(filepath.Join("web", "viewer", "style.css"), filepath.Join(publicDir, "style.css"))
	_ = os.Chmod(filepath.Join(publicDir, "style.css"), 0644)

	// Синхронизируем все сгенерированные json-данные туров и лидеров
	dataEntries, err := os.ReadDir(filepath.Join("web", "viewer", "data"))
	if err == nil {
		publicDataDir := filepath.Join(publicDir, "data")
		viewerDataDir := filepath.Join(viewerDir, "data")
		_ = os.MkdirAll(publicDataDir, 0755)
		_ = os.MkdirAll(viewerDataDir, 0755)
		for _, entry := range dataEntries {
			if !entry.IsDir() {
				srcData := filepath.Join("web", "viewer", "data", entry.Name())
				_ = copyFile(srcData, filepath.Join(publicDataDir, entry.Name()))
				_ = copyFile(srcData, filepath.Join(viewerDataDir, entry.Name()))
			}
		}
	}
}

// dispatchCatalog генерирует и выгружает active_leagues.json для Telegram Mini App.
func dispatchCatalog(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	catalog, err := analytics.GetActiveLeagues(ctx, pool)
	if err != nil {
		return fmt.Errorf("GetActiveLeagues: %w", err)
	}
	catalog.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	catalogJSON, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog JSON: %w", err)
	}

	// 1. Сохранение локально
	_ = os.MkdirAll("web/viewer", 0755)
	_ = os.WriteFile("web/viewer/active_leagues.json", catalogJSON, 0644)

	// 2. Публичные директории Nginx
	if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
		publicDir := filepath.Dir(config.Config.WebApp.MediaDir)
		viewerDir := filepath.Join(publicDir, "viewer")
		_ = os.MkdirAll(viewerDir, 0755)
		_ = os.MkdirAll(config.Config.WebApp.MediaDir, 0755)

		_ = os.WriteFile(filepath.Join(publicDir, "active_leagues.json"), catalogJSON, 0644)
		_ = os.WriteFile(filepath.Join(viewerDir, "active_leagues.json"), catalogJSON, 0644)
		_ = os.WriteFile(filepath.Join(config.Config.WebApp.MediaDir, "active_leagues.json"), catalogJSON, 0644)

		syncViewerFiles(publicDir)
	}

	log.Info("active leagues catalog exported successfully",
		zap.Int("total_leagues", catalog.Total),
		zap.Int("categories", len(catalog.Categories)),
	)
	fmt.Printf("✅ Каталог активных турниров обновлен (%d лиг, %d категорий): web/viewer/active_leagues.json\n",
		catalog.Total, len(catalog.Categories))

	return nil
}

// dispatchSyncAll выполняет пакетную генерацию данных (каталог, таблицы, туры, лидеры)
// для ВСЕХ активных лиг из базы данных и синхронизирует их с Telegram Mini App.
func dispatchSyncAll(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	locale := taskData.GetLocale()
	if locale == "" {
		locale = "ru"
	}

	// 1. Каталог активных лиг
	catalog, err := analytics.GetActiveLeagues(ctx, pool)
	if err != nil {
		return fmt.Errorf("GetActiveLeagues: %w", err)
	}
	catalog.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	catalogJSON, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog JSON: %w", err)
	}

	_ = os.MkdirAll("web/viewer", 0755)
	_ = os.WriteFile("web/viewer/active_leagues.json", catalogJSON, 0644)

	publicDir := ""
	if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
		publicDir = filepath.Dir(config.Config.WebApp.MediaDir)
		_ = os.MkdirAll(config.Config.WebApp.MediaDir, 0755)
		_ = os.WriteFile(filepath.Join(publicDir, "active_leagues.json"), catalogJSON, 0644)
		_ = os.WriteFile(filepath.Join(publicDir, "viewer", "active_leagues.json"), catalogJSON, 0644)
		_ = os.WriteFile(filepath.Join(config.Config.WebApp.MediaDir, "active_leagues.json"), catalogJSON, 0644)
	}

	fmt.Printf("🔄 Запуск пакетной генерации актуальных данных для %d активных лиг...\n", len(catalog.Leagues))

	// 2. Синхронизация данных для каждого турнира
	for _, l := range catalog.Leagues {
		fmt.Printf("   ⏳ [%s] (ID: %d, Сезон: %d)...\n", l.DisplayName, l.LeagueID, l.SeasonID)
		if err := syncSingleLeagueData(ctx, pool, l.LeagueID, locale, log); err != nil {
			log.Warn("syncSingleLeagueData failed", zap.Int("league_id", l.LeagueID), zap.Error(err))
			fmt.Printf("   ⚠️  [%s] Ошибка: %v\n", l.DisplayName, err)
		} else {
			fmt.Printf("   ✅ [%s] Данные успешно сгенерированы (тур %s, лидеры)\n", l.DisplayName, l.CurrRound)
		}
	}

	// 3. Синхронизация статики Mini App
	if publicDir != "" {
		syncViewerFiles(publicDir)
	}

	fmt.Printf("\n✨ Пакетная синхронизация завершена! Все %d лиг готовы для работы в Mini App.\n", len(catalog.Leagues))
	return nil
}

func syncSingleLeagueData(ctx context.Context, pool *pgxpool.Pool, leagueID int, locale string, log *zap.Logger) error {
	info, err := analytics.GetLeagueSeasonInfo(ctx, pool, leagueID)
	if err != nil {
		return fmt.Errorf("GetLeagueSeasonInfo: %w", err)
	}
	info.Locale = locale

	publicDir := ""
	mediaDir := ""
	if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
		mediaDir = config.Config.WebApp.MediaDir
		publicDir = filepath.Dir(mediaDir)
	}

	_ = os.MkdirAll("web/viewer/data", 0755)

	// 1. Лидеры сезона (топ-10)
	leadersData, err := analytics.GetSeasonPlayerLeaders(ctx, pool, &info, 10)
	if err == nil && leadersData != nil {
		if leadersJSON, jErr := json.MarshalIndent(leadersData, "", "  "); jErr == nil {
			_ = os.WriteFile(fmt.Sprintf("web/viewer/data/%d_%d_leaders.json", info.LeagueID, info.SeasonID), leadersJSON, 0644)
			if mediaDir != "" {
				seasonMediaDir := filepath.Join(mediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID))
				_ = os.MkdirAll(seasonMediaDir, 0755)
				_ = os.WriteFile(filepath.Join(seasonMediaDir, "leaders.json"), leadersJSON, 0644)
				if publicDir != "" {
					viewerDataDir := filepath.Join(publicDir, "viewer", "data")
					_ = os.MkdirAll(viewerDataDir, 0755)
					_ = os.WriteFile(filepath.Join(viewerDataDir, fmt.Sprintf("%d_%d_leaders.json", info.LeagueID, info.SeasonID)), leadersJSON, 0644)
				}
			}
		}
	} else if err != nil {
		log.Warn("GetSeasonPlayerLeaders warning", zap.Int("league_id", leagueID), zap.Error(err))
	}

	// 2. Актуальная турнирная таблица
	standings, _ := analytics.GetCurrentStandings(ctx, pool, &info)

	// 3. Результаты тура (CurrRoundID)
	if info.CurrRoundID > 0 {
		matches, mErr := analytics.GetRoundResults(ctx, pool, &info)
		if mErr == nil && len(matches) > 0 {
			roundID := matches[0].RoundID
			roundName := matches[0].RoundName
			summary := analytics.ComputeRoundSummary(matches)

			payload := roundExportPayload{
				LeagueID:      info.LeagueID,
				LeagueName:    info.LeagueName,
				SeasonID:      info.SeasonID,
				SeasonName:    info.SeasonName,
				RoundID:       roundID,
				RoundName:     roundName,
				PrevRoundID:   info.PrevRoundID,
				NextRoundID:   info.NextRoundID,
				NextRoundName: info.NextRoundName,
				Summary:       summary,
				Matches:       matches,
				Standings:     standings,
			}
			if pJSON, pErr := json.MarshalIndent(payload, "", "  "); pErr == nil {
				_ = os.WriteFile(fmt.Sprintf("web/viewer/data/%d_%d_%d.json", info.LeagueID, info.SeasonID, roundID), pJSON, 0644)
				if mediaDir != "" {
					mediaRoundDir := filepath.Join(mediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID), strconv.Itoa(roundID))
					_ = os.MkdirAll(mediaRoundDir, 0755)
					_ = os.WriteFile(filepath.Join(mediaRoundDir, "round_data.json"), pJSON, 0644)
					if publicDir != "" {
						viewerDataDir := filepath.Join(publicDir, "viewer", "data")
						_ = os.MkdirAll(viewerDataDir, 0755)
						_ = os.WriteFile(filepath.Join(viewerDataDir, fmt.Sprintf("%d_%d_%d.json", info.LeagueID, info.SeasonID, roundID)), pJSON, 0644)
					}
				}
			}
		}
	}

	// 4. Актуальные данные текущего / предстоящего тура (NextRoundID)
	if info.NextRoundID > 0 && info.NextRoundID != info.CurrRoundID {
		nextInfo := info
		nextInfo.CurrRoundID = info.NextRoundID
		nextStandings, sErr := analytics.GetCurrentStandings(ctx, pool, &nextInfo)
		if sErr != nil || len(nextStandings) == 0 {
			nextStandings = standings
		}

		nextMatches, nmErr := analytics.GetRoundResults(ctx, pool, &nextInfo)
		if nmErr == nil && len(nextMatches) > 0 {
			for i := range nextMatches {
				m := &nextMatches[i]
				h2h, _ := analytics.GetH2HMatches(ctx, pool, m.TeamHomeID, m.TeamAwayID, info.Locale, 10)
				homeProfile, _ := analytics.GetTeamPreMatchProfile(ctx, pool, m.TeamHomeID, info.SeasonID)
				awayProfile, _ := analytics.GetTeamPreMatchProfile(ctx, pool, m.TeamAwayID, info.SeasonID)
				m.H2H = &h2h
				m.PreMatchStats = &analytics.PreMatchStatsComparison{
					Home: homeProfile,
					Away: awayProfile,
				}
			}

			summary := analytics.ComputeRoundSummary(nextMatches)
			roundName := info.NextRoundName
			if nextMatches[0].RoundName != "" {
				roundName = nextMatches[0].RoundName
			}

			payload := roundExportPayload{
				LeagueID:      info.LeagueID,
				LeagueName:    info.LeagueName,
				SeasonID:      info.SeasonID,
				SeasonName:    info.SeasonName,
				RoundID:       info.NextRoundID,
				RoundName:     roundName,
				PrevRoundID:   info.CurrRoundID,
				NextRoundID:   info.NextRoundID,
				NextRoundName: info.NextRoundName,
				Summary:       summary,
				Matches:       nextMatches,
				Standings:     nextStandings,
			}
			if pJSON, pErr := json.MarshalIndent(payload, "", "  "); pErr == nil {
				_ = os.WriteFile(fmt.Sprintf("web/viewer/data/%d_%d_%d.json", info.LeagueID, info.SeasonID, info.NextRoundID), pJSON, 0644)
				if mediaDir != "" {
					mediaRoundDir := filepath.Join(mediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID), strconv.Itoa(info.NextRoundID))
					_ = os.MkdirAll(mediaRoundDir, 0755)
					_ = os.WriteFile(filepath.Join(mediaRoundDir, "round_data.json"), pJSON, 0644)
					if publicDir != "" {
						viewerDataDir := filepath.Join(publicDir, "viewer", "data")
						_ = os.MkdirAll(viewerDataDir, 0755)
						_ = os.WriteFile(filepath.Join(viewerDataDir, fmt.Sprintf("%d_%d_%d.json", info.LeagueID, info.SeasonID, info.NextRoundID)), pJSON, 0644)
					}
				}
			}
		}
	}

	return nil
}

// dispatchLeaders формирует аналитику лидеров сезона, экспортирует leaders.json для Mini App,
// рендерит инфографику 1920px и публикует пост в Telegram.
func dispatchLeaders(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	style := strings.ToLower(strings.TrimSpace(taskData.GetStyle()))
	if style == "euro" {
		return dispatchLeadersEuro(ctx, taskData, log)
	}

	leagueID := taskData.GetLeagueId()
	if leagueID == 0 {
		// Вариант А: Медиа-альбом лидеров Топ-5 лиг Европы
		return dispatchLeadersAlbum(ctx, taskData, log)
	}

	info, err := analytics.GetLeagueSeasonInfo(ctx, pool, leagueID)
	if err != nil {
		return fmt.Errorf("GetLeagueSeasonInfo: %w", err)
	}
	info.Locale = taskData.GetLocale()

	// 1. Расчёт всех категорий лидеров сезона (топ-10)
	leadersData, err := analytics.GetSeasonPlayerLeaders(ctx, pool, &info, 10)
	if err != nil {
		return fmt.Errorf("GetSeasonPlayerLeaders: %w", err)
	}

	// 2. Экспорт leaders.json для Mini App
	leadersJSON, err := json.MarshalIndent(leadersData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal leaders JSON: %w", err)
	}

	_ = os.MkdirAll("web/viewer/data", 0755)
	_ = os.WriteFile("web/viewer/leaders.json", leadersJSON, 0644)
	_ = os.WriteFile(fmt.Sprintf("web/viewer/data/%d_%d_leaders.json", info.LeagueID, info.SeasonID), leadersJSON, 0644)

	// Публичные директории WebApp
	if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
		publicDir := filepath.Dir(config.Config.WebApp.MediaDir)
		viewerDir := filepath.Join(publicDir, "viewer")
		viewerDataDir := filepath.Join(viewerDir, "data")
		_ = os.MkdirAll(viewerDataDir, 0755)

		_ = os.WriteFile(filepath.Join(publicDir, "leaders.json"), leadersJSON, 0644)
		_ = os.WriteFile(filepath.Join(viewerDir, "leaders.json"), leadersJSON, 0644)
		_ = os.WriteFile(filepath.Join(viewerDataDir, fmt.Sprintf("%d_%d_leaders.json", info.LeagueID, info.SeasonID)), leadersJSON, 0644)

		seasonMediaDir := filepath.Join(config.Config.WebApp.MediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID))
		_ = os.MkdirAll(seasonMediaDir, 0755)
		_ = os.WriteFile(filepath.Join(seasonMediaDir, "leaders.json"), leadersJSON, 0644)

		syncViewerFiles(publicDir)
	}

	// 3. Рендеринг инфографики лидеров в PNG
	vm := viewmodel.BuildLeadersViewModel(leadersData)
	templatePath := filepath.Join("template", "leaders_template.html")

	outputPath := taskData.GetOutput()
	if outputPath == "" {
		outputPath = "leaders.png"
	}

	log.Info("rendering leaders PNG", zap.String("output", outputPath))
	if err := renderer.RenderLeadersPNG(vm, templatePath, outputPath); err != nil {
		log.Warn("RenderLeadersPNG failed, proceeding with text", zap.Error(err))
	} else {
		log.Info("leaders PNG rendered successfully", zap.String("file", outputPath))
		if imgData, rErr := os.ReadFile(outputPath); rErr == nil {
			_ = os.WriteFile("web/viewer/leaders.png", imgData, 0644)
			if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
				publicDir := filepath.Dir(config.Config.WebApp.MediaDir)
				seasonMediaDir := filepath.Join(config.Config.WebApp.MediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID))
				_ = os.MkdirAll(seasonMediaDir, 0755)
				_ = os.WriteFile(filepath.Join(seasonMediaDir, "leaders.png"), imgData, 0644)
				_ = os.WriteFile(filepath.Join(publicDir, "viewer", "leaders.png"), imgData, 0644)
			}
		}
	}

	// 4. Формирование текста поста Telegram
	botUser := "football_data_lab_bot"
	shortName := "data_lab"
	baseURL := "https://t.me/football_data_lab_bot/data_lab"
	if config.Config.WebApp != nil {
		if config.Config.WebApp.BotUsername != "" {
			botUser = config.Config.WebApp.BotUsername
		}
		if config.Config.WebApp.BotShortName != "" {
			shortName = config.Config.WebApp.BotShortName
		}
		if config.Config.WebApp.BaseURL != "" {
			baseURL = config.Config.WebApp.BaseURL
		}
	}

	postCfg := copywriter.LeagueLeadersPostConfig{
		LeagueName:   info.LeagueName,
		SeasonName:   info.SeasonName,
		LeagueID:     info.LeagueID,
		SeasonID:     info.SeasonID,
		BotUsername:  botUser,
		BotShortName: shortName,
		BaseURL:      baseURL,
	}

	leadersURL := fmt.Sprintf("https://t.me/%s/%s?startapp=%d_leaders", botUser, shortName, info.LeagueID)
	tableURL := fmt.Sprintf("https://t.me/%s/%s?startapp=%d_table", botUser, shortName, info.LeagueID)

	postText := copywriter.GenerateLeadersHTMLPost(postCfg, leadersData)
	postText = fmt.Sprintf("%s\n\n<a href=\"%s\">Все лидеры</a> · <a href=\"%s\">Турнирная таблица</a>", postText, leadersURL, tableURL)

	if !taskData.GetPublish() {
		fmt.Printf("\n=== [Dry Run] Текст лидеров турнира (%s) ===\n\n%s\n\n", info.LeagueName, postText)
		return nil
	}

	// 5. Публикация в Telegram
	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return fmt.Errorf("telegram configuration missing (BotToken or ChannelID is empty)")
	}

	channelID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
	storage := getStorage(ctx)

	// Проверка идемпотентности: защита от дубликатов
	if storage != nil && !taskData.GetForce() {
		already, err := storage.IsAlreadyPublished(ctx, channelID, PubTypeLeaders, info.LeagueID, info.SeasonID)
		if err == nil && already {
			log.Info("leaders already published, skipping duplicate publication (use -force to override)",
				zap.String("type", string(PubTypeLeaders)),
				zap.Int("league_id", info.LeagueID),
				zap.Int("season_id", info.SeasonID),
			)
			fmt.Printf("ℹ️ Лидеры турнира (лига %d, сезон %d) уже были опубликованы в канале %s. Для повторной отправки укажите флаг -force\n", info.LeagueID, info.SeasonID, tg.ChannelID)
			return nil
		}
	}

	var keyboard *InlineKeyboardMarkup = nil

	pubCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	log.Info("publishing leaders infographic to telegram",
		zap.String("channel", tg.ChannelID),
		zap.String("league", info.LeagueName),
		zap.String("season", info.SeasonName),
	)

	msgID, pubErr := SendPhotoWithKeyboard(pubCtx, tg.BotToken, tg.ChannelID, postText, outputPath, keyboard)
	if pubErr != nil {
		return fmt.Errorf("SendPhotoWithKeyboard failed: %w", pubErr)
	}

	log.Info("leaders infographic published successfully",
		zap.String("channel", tg.ChannelID),
		zap.Int64("message_id", msgID),
	)
	fmt.Printf("✅ Лидеры сезона опубликованы в Telegram (%s)! Message ID: %d\n", tg.ChannelID, msgID)

	// Сохранение в БД
	if storage != nil {
		pub := &Publication{
			ChannelID:       channelID,
			MessageID:       &msgID,
			PublicationType: PubTypeLeaders,
			LeagueID:        &info.LeagueID,
			SeasonID:        &info.SeasonID,
			Title:           fmt.Sprintf("%s — Лидеры сезона %s", info.LeagueName, info.SeasonName),
			TextContent:     postText,
			MediaURL:        outputPath,
			MediaType:       MediaTypePhoto,
			Status:          StatusPublished,
		}
		if _, saveErr := storage.Create(ctx, pub); saveErr != nil {
			log.Warn("failed to record publication to database", zap.Error(saveErr))
		}
	}

	return nil
}

// dispatchLeadersAlbum формирует инфографику лидеров сезона для всех активных топ-лиг (Топ-5)
// и публикует их медиа-альбомом (MediaGroup) в Telegram (Вариант А).
func dispatchLeadersAlbum(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	catalog, err := analytics.GetActiveLeagues(ctx, pool)
	if err != nil {
		return fmt.Errorf("GetActiveLeagues: %w", err)
	}

	// Топ-5 европейских чемпионатов в порядке приоритета: АПЛ, Ла Лига, Бундеслига, Серия А, Лига 1
	targetLeagues := []int{8, 564, 82, 384, 301}
	leagueMap := make(map[int]analytics.ActiveLeagueItem)
	for _, l := range catalog.Leagues {
		leagueMap[l.LeagueID] = l
	}

	var imagePaths []string
	var summaryRows []string
	var miniAppLinks []string

	botUser := "football_data_lab_bot"
	shortName := "data_lab"
	if config.Config.WebApp != nil {
		if config.Config.WebApp.BotUsername != "" {
			botUser = strings.TrimPrefix(config.Config.WebApp.BotUsername, "@")
		}
		if config.Config.WebApp.BotShortName != "" {
			shortName = config.Config.WebApp.BotShortName
		}
	}

	for _, lid := range targetLeagues {
		lItem, exists := leagueMap[lid]
		if !exists {
			continue
		}

		info, lErr := analytics.GetLeagueSeasonInfo(ctx, pool, lid)
		if lErr != nil {
			log.Warn("GetLeagueSeasonInfo failed", zap.Int("league_id", lid), zap.Error(lErr))
			continue
		}
		info.Locale = taskData.GetLocale()

		leadersData, lDataErr := analytics.GetSeasonPlayerLeaders(ctx, pool, &info, 10)
		if lDataErr != nil || leadersData == nil {
			log.Warn("GetSeasonPlayerLeaders failed", zap.Int("league_id", lid), zap.Error(lDataErr))
			continue
		}

		// 1. Экспорт leaders.json для Mini App
		if leadersJSON, jErr := json.MarshalIndent(leadersData, "", "  "); jErr == nil {
			_ = os.MkdirAll("web/viewer/data", 0755)
			_ = os.WriteFile(fmt.Sprintf("web/viewer/data/%d_%d_leaders.json", info.LeagueID, info.SeasonID), leadersJSON, 0644)
			if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
				publicDir := filepath.Dir(config.Config.WebApp.MediaDir)
				seasonMediaDir := filepath.Join(config.Config.WebApp.MediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID))
				_ = os.MkdirAll(seasonMediaDir, 0755)
				_ = os.WriteFile(filepath.Join(seasonMediaDir, "leaders.json"), leadersJSON, 0644)
				viewerDataDir := filepath.Join(publicDir, "viewer", "data")
				_ = os.MkdirAll(viewerDataDir, 0755)
				_ = os.WriteFile(filepath.Join(viewerDataDir, fmt.Sprintf("%d_%d_leaders.json", info.LeagueID, info.SeasonID)), leadersJSON, 0644)
			}
		}

		// 2. Рендеринг PNG инфографики 1920x1080
		vm := viewmodel.BuildLeadersViewModel(leadersData)
		tmplPath := filepath.Join("template", "leaders_template.html")
		imgPath := fmt.Sprintf("leaders_%d.png", lid)

		if rErr := renderer.RenderLeadersPNG(vm, tmplPath, imgPath); rErr != nil {
			log.Warn("RenderLeadersPNG failed for league", zap.Int("league_id", lid), zap.Error(rErr))
		} else {
			imagePaths = append(imagePaths, imgPath)
			if imgData, readErr := os.ReadFile(imgPath); readErr == nil {
				_ = os.WriteFile(fmt.Sprintf("web/viewer/leaders_%d.png", lid), imgData, 0644)
				if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
					publicDir := filepath.Dir(config.Config.WebApp.MediaDir)
					seasonMediaDir := filepath.Join(config.Config.WebApp.MediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID))
					_ = os.MkdirAll(seasonMediaDir, 0755)
					_ = os.WriteFile(filepath.Join(seasonMediaDir, "leaders.png"), imgData, 0644)
					_ = os.WriteFile(filepath.Join(publicDir, "viewer", fmt.Sprintf("leaders_%d.png", lid)), imgData, 0644)
				}
			}
		}

		// Формируем строчку для подписи
		topScorer := "—"
		if len(leadersData.TopScorers) > 0 {
			ts := leadersData.TopScorers[0]
			topScorer = fmt.Sprintf("<b>%s</b> (%s) — <b>%d</b> ⚽", ts.PlayerName, ts.TeamCode, ts.PrimaryVal)
		}
		summaryRows = append(summaryRows, fmt.Sprintf("%s <b>%s:</b> %s", lItem.CountryFlag, lItem.DisplayName, topScorer))

		miniAppLinks = append(miniAppLinks, fmt.Sprintf("<a href=\"https://t.me/%s/%s?startapp=%d_leaders\">%s</a>", botUser, shortName, lid, lItem.DisplayName))
	}

	if len(imagePaths) == 0 {
		return fmt.Errorf("no leader infographics could be rendered")
	}

	// 3. Формирование единой подписи альбома
	var caption strings.Builder
	caption.WriteString("👑 <b>ЛИДЕРЫ СЕЗОНА • ТОП-5 ЛИГ ЕВРОПЫ</b>\n")
	caption.WriteString("<i>Главные бомбардиры и индивидуальная статистика</i>\n\n")
	for _, row := range summaryRows {
		caption.WriteString(row + "\n")
	}
	caption.WriteString("\n📊 <b>Все лидеры в Mini App:</b>\n")
	caption.WriteString(strings.Join(miniAppLinks, " · "))

	if !taskData.GetPublish() {
		fmt.Printf("\n=== [Dry Run] Альбом лидеров Топ-5 лиг (%d карточек) ===\n%s\n\n", len(imagePaths), caption.String())
		return nil
	}

	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return fmt.Errorf("telegram configuration missing (BotToken or ChannelID is empty)")
	}

	channelID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
	storage := getStorage(ctx)

	// Проверка идемпотентности альбома
	if storage != nil && !taskData.GetForce() {
		already, err := storage.IsAlreadyPublished(ctx, channelID, PubTypeLeaders, 0, 0)
		if err == nil && already {
			log.Info("leaders album already published, skipping duplicate (use -force to override)")
			fmt.Printf("ℹ️ Альбом лидеров сезона уже опубликован в %s. Для повторной публикации укажите -force\n", tg.ChannelID)
			return nil
		}
	}

	pubCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	fmt.Printf("📤 Отправка медиа-альбома лидеров (%d лиг) в Telegram %s...\n", len(imagePaths), tg.ChannelID)
	msgIDs, err := SendMediaGroup(pubCtx, tg.BotToken, tg.ChannelID, imagePaths, caption.String())
	if err != nil {
		return fmt.Errorf("SendMediaGroup failed: %w", err)
	}

	fmt.Printf("✅ Альбом лидеров сезона успешно опубликован (сообщений: %d, первое ID=%d)!\n", len(msgIDs), msgIDs[0])

	if storage != nil && len(msgIDs) > 0 {
		pub := &Publication{
			ChannelID:       channelID,
			MessageID:       &msgIDs[0],
			PublicationType: PubTypeLeaders,
			Title:           "Лидеры сезона • Топ-5 лиг Европы (Альбом)",
			TextContent:     caption.String(),
			MediaURL:        imagePaths[0],
			MediaType:       MediaTypePhoto,
			Status:          StatusPublished,
		}
		if _, saveErr := storage.Create(ctx, pub); saveErr != nil {
			log.Warn("failed to record album publication to database", zap.Error(saveErr))
		}
	}

	return nil
}

// dispatchLeadersEuro формирует сводный пан-европейский топ бомбардиров и ассистентов (Вариант Б).
func dispatchLeadersEuro(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	catalog, err := analytics.GetActiveLeagues(ctx, pool)
	if err != nil {
		return fmt.Errorf("GetActiveLeagues: %w", err)
	}

	targetLeagues := []int{8, 564, 82, 384, 301}
	leagueMap := make(map[int]analytics.ActiveLeagueItem)
	for _, l := range catalog.Leagues {
		leagueMap[l.LeagueID] = l
	}

	type euroScorerItem struct {
		PlayerName  string
		TeamCode    string
		LeagueName  string
		CountryFlag string
		Goals       int
		Assists     int
	}

	var allScorers []euroScorerItem

	for _, lid := range targetLeagues {
		lItem, exists := leagueMap[lid]
		if !exists {
			continue
		}

		info, lErr := analytics.GetLeagueSeasonInfo(ctx, pool, lid)
		if lErr != nil {
			continue
		}
		info.Locale = taskData.GetLocale()

		data, dErr := analytics.GetSeasonPlayerLeaders(ctx, pool, &info, 10)
		if dErr != nil || data == nil {
			continue
		}

		for _, p := range data.TopScorers {
			allScorers = append(allScorers, euroScorerItem{
				PlayerName:  p.PlayerName,
				TeamCode:    p.TeamCode,
				LeagueName:  lItem.DisplayName,
				CountryFlag: lItem.CountryFlag,
				Goals:       p.PrimaryVal,
				Assists:     p.SecondaryVal,
			})
		}
	}

	if len(allScorers) == 0 {
		return fmt.Errorf("no scorers data found across European leagues")
	}

	// Сортируем снайперов по убыванию голов
	sort.Slice(allScorers, func(i, j int) bool {
		if allScorers[i].Goals == allScorers[j].Goals {
			return allScorers[i].Assists > allScorers[j].Assists
		}
		return allScorers[i].Goals > allScorers[j].Goals
	})

	// Оставляем уникальных игроков топ-10
	var top10Scorers []euroScorerItem
	seen := make(map[string]bool)
	for _, s := range allScorers {
		if !seen[s.PlayerName] {
			top10Scorers = append(top10Scorers, s)
			seen[s.PlayerName] = true
			if len(top10Scorers) == 10 {
				break
			}
		}
	}

	botUser := "football_data_lab_bot"
	shortName := "data_lab"
	if config.Config.WebApp != nil {
		if config.Config.WebApp.BotUsername != "" {
			botUser = strings.TrimPrefix(config.Config.WebApp.BotUsername, "@")
		}
		if config.Config.WebApp.BotShortName != "" {
			shortName = config.Config.WebApp.BotShortName
		}
	}

	var b strings.Builder
	b.WriteString("🇪🇺 <b>ЗОЛОТАЯ БУТСА • ТОП БОМБАРДИРОВ ЕВРОПЫ</b>\n")
	b.WriteString("<i>Сводная таблица снайперов Топ-5 чемпионатов Европы</i>\n\n")

	for i, s := range top10Scorers {
		med := fmt.Sprintf("%d.", i+1)
		if i == 0 {
			med = "🥇"
		} else if i == 1 {
			med = "🥈"
		} else if i == 2 {
			med = "🥉"
		}
		b.WriteString(fmt.Sprintf("%s <b>%s</b> (%s %s) — <b>%d</b> ⚽ <i>(пас: %d)</i>\n", med, s.PlayerName, s.CountryFlag, s.TeamCode, s.Goals, s.Assists))
	}

	b.WriteString("\n📊 <b>Интерактивная статистика в Mini App:</b>\n")
	b.WriteString(fmt.Sprintf("<a href=\"https://t.me/%s/%s?startapp=all_leaders\">Все лидеры Европы</a>", botUser, shortName))

	postText := b.String()

	if !taskData.GetPublish() {
		fmt.Printf("\n=== [Dry Run] Сводный топ Европы (Вариант Б) ===\n%s\n\n", postText)
		return nil
	}

	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return fmt.Errorf("telegram configuration missing (BotToken or ChannelID is empty)")
	}

	pubCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var keyboard *InlineKeyboardMarkup = nil
	msgID, err := SendTextMessageWithKeyboardAndPreview(pubCtx, tg.BotToken, tg.ChannelID, postText, keyboard, &LinkPreviewOptions{IsDisabled: true})
	if err != nil {
		return fmt.Errorf("SendTextMessage failed: %w", err)
	}

	fmt.Printf("✅ Сводный топ бомбардиров Европы опубликован (Message ID: %d)!\n", msgID)
	return nil
}

// dispatchRecalculateStreaks выполняет ручной или автоматический перерасчёт серий в БД.
func dispatchRecalculateStreaks(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	filter := buildStreaksFilterFromTaskData(taskData)

	log.Info("recalculating streaks",
		zap.Int("league_id", filter.LeagueID),
		zap.Int("season_id", filter.SeasonID),
		zap.Int64("team_id", filter.TeamID),
		zap.Int("coach_id", filter.CoachID),
		zap.String("locale", filter.Locale),
	)

	streaks, err := analytics.ComputeAndSaveTeamStreaks(ctx, pool, filter)
	if err != nil {
		return fmt.Errorf("ComputeAndSaveTeamStreaks: %w", err)
	}

	fmt.Printf("✅ Перерасчёт завершён успешно! Найдено и сохранено серий: %d\n", len(streaks))
	for i, s := range streaks {
		if i >= 10 {
			fmt.Printf("... и ещё %d серий\n", len(streaks)-10)
			break
		}
		recordMarker := ""
		if s.IsRecordLeague {
			recordMarker = " [🏆 РЕКОРД ЛИГИ]"
		} else if s.IsRecordClub {
			recordMarker = " [⭐ РЕКОРД КЛУБА]"
		}
		fmt.Printf("  • %s (%s): %d матчей — %s%s\n", s.EntityName, s.ScopeName, s.Length, s.StreakTypeName, recordMarker)
	}

	return nil
}

// dispatchFacts формирует карточку топ-серии (1200x1200px) и готовит публикацию в Telegram.
func dispatchFacts(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	filter := buildStreaksFilterFromTaskData(taskData)
	if filter.LeagueID == 0 {
		filter.LeagueID = 8 // Default: Premier League
	}

	// 1. Загрузка лучших активных серий
	streaks, err := analytics.GetTopActiveStreaks(ctx, pool, filter, 5)
	if err != nil {
		return fmt.Errorf("GetTopActiveStreaks: %w", err)
	}

	// Если в БД ещё нет рассчитанных серий, автоматически рассчитываем их
	if len(streaks) == 0 {
		log.Info("no active streaks in database, automatically computing...", zap.Int("league_id", filter.LeagueID))
		if _, cErr := analytics.ComputeAndSaveTeamStreaks(ctx, pool, filter); cErr != nil {
			return fmt.Errorf("automatic ComputeAndSaveTeamStreaks failed: %w", cErr)
		}
		streaks, err = analytics.GetTopActiveStreaks(ctx, pool, filter, 5)
		if err != nil {
			return fmt.Errorf("GetTopActiveStreaks after recalculate: %w", err)
		}
	}

	if len(streaks) == 0 {
		fmt.Printf("ℹ️ Для выбранных параметров (лига: %d, команда: %d) активных серий длиной >= %d не найдено.\n",
			filter.LeagueID, filter.TeamID, filter.MinLength)
		return nil
	}

	topStreak := streaks[0]
	log.Info("selected top streak for publication",
		zap.Int64("streak_id", topStreak.ID),
		zap.String("entity", topStreak.EntityName),
		zap.String("type", topStreak.StreakType),
		zap.Int("length", topStreak.Length),
		zap.Float64("score", topStreak.InterestScore),
	)

	// 2. Рендеринг инфографики 1200x1200px в PNG
	vm := viewmodel.BuildFactCardViewModel(topStreak)
	templatePath := filepath.Join("template", "fact_card_template.html")

	outputPath := taskData.GetOutput()
	if outputPath == "" {
		outputPath = fmt.Sprintf("facts_%d_%d.png", topStreak.LeagueID, topStreak.EntityID)
	}

	log.Info("rendering fact card PNG", zap.String("output", outputPath))
	if err := renderer.RenderFactCardPNG(vm, templatePath, outputPath); err != nil {
		return fmt.Errorf("RenderFactCardPNG: %w", err)
	}
	log.Info("fact card PNG rendered successfully", zap.String("file", outputPath))

	// 3. Формирование текста поста Telegram
	botUser := "football_data_lab_bot"
	shortName := "data_lab"
	baseURL := "https://t.me/football_data_lab_bot/data_lab"
	if config.Config.WebApp != nil {
		if config.Config.WebApp.BotUsername != "" {
			botUser = config.Config.WebApp.BotUsername
		}
		if config.Config.WebApp.BotShortName != "" {
			shortName = config.Config.WebApp.BotShortName
		}
		if config.Config.WebApp.BaseURL != "" {
			baseURL = config.Config.WebApp.BaseURL
		}
	}

	postCfg := copywriter.FactPostConfig{
		LeagueName:   topStreak.LeagueName,
		LeagueID:     topStreak.LeagueID,
		BotUsername:  botUser,
		BotShortName: shortName,
		BaseURL:      baseURL,
	}

	streakURL := fmt.Sprintf("https://t.me/%s/%s?startapp=%d_matches", botUser, shortName, topStreak.LeagueID)
	tableURL := fmt.Sprintf("https://t.me/%s/%s?startapp=%d_table", botUser, shortName, topStreak.LeagueID)

	postText := copywriter.GenerateFactHTMLPost(topStreak, postCfg)
	postText = fmt.Sprintf("%s\n\n<a href=\"%s\">Все серии</a> · <a href=\"%s\">Турнирная таблица</a>", postText, streakURL, tableURL)

	// 4. Dry-run режим
	if !taskData.GetPublish() {
		fmt.Printf("\n=== [Dry Run] Текст карточки фактов & серий (%s) ===\n\n%s\n\n", topStreak.EntityName, postText)
		fmt.Printf("🖼 Инфографика сохранена: %s (1200x1200px)\n", outputPath)
		return nil
	}

	// 5. Публикация в Telegram
	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return fmt.Errorf("telegram configuration missing (BotToken or ChannelID is empty)")
	}

	channelID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
	storage := getStorage(ctx)

	if storage != nil && !taskData.GetForce() {
		already, err := storage.IsAlreadyPublished(ctx, channelID, PubTypeFacts, topStreak.LeagueID, int(topStreak.EntityID))
		if err == nil && already {
			log.Info("fact card already published, skipping duplicate publication (use -force to override)",
				zap.Int("league_id", topStreak.LeagueID),
				zap.Int64("entity_id", topStreak.EntityID),
			)
			fmt.Printf("ℹ️ Серия команды %s уже была опубликована. Для повторной отправки укажите флаг -force\n", topStreak.EntityName)
			return nil
		}
	}

	seasonID := 0
	if topStreak.EndSeasonID != nil {
		seasonID = *topStreak.EndSeasonID
	}

	var keyboard *InlineKeyboardMarkup = nil

	pubCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	log.Info("publishing fact card to telegram",
		zap.String("channel", tg.ChannelID),
		zap.String("entity", topStreak.EntityName),
		zap.Int("length", topStreak.Length),
	)

	msgID, err := SendPhotoWithKeyboard(pubCtx, tg.BotToken, tg.ChannelID, postText, outputPath, keyboard)
	if err != nil {
		return fmt.Errorf("SendPhotoWithKeyboard: %w", err)
	}

	log.Info("fact card published successfully",
		zap.String("channel", tg.ChannelID),
		zap.Int64("message_id", msgID),
	)
	fmt.Printf("✅ Карточка факта и серии опубликована в Telegram (%s)! Message ID: %d\n", tg.ChannelID, msgID)

	// Сохранение в БД
	if storage != nil {
		pub := &Publication{
			ChannelID:       channelID,
			MessageID:       &msgID,
			PublicationType: PubTypeFacts,
			LeagueID:        &topStreak.LeagueID,
			SeasonID:        &seasonID,
			Title:           fmt.Sprintf("%s — Серия %d матчей (%s)", topStreak.EntityName, topStreak.Length, topStreak.StreakTypeName),
			TextContent:     postText,
			MediaURL:        outputPath,
			MediaType:       MediaTypePhoto,
			Status:          StatusPublished,
		}
		if _, saveErr := storage.Create(ctx, pub); saveErr != nil {
			log.Warn("failed to record publication to database", zap.Error(saveErr))
		}
	}

	return nil
}

func buildStreaksFilterFromTaskData(taskData config.TaskData) analytics.StreaksFilter {
	filter := analytics.StreaksFilter{
		LeagueID: taskData.GetLeagueId(),
		Locale:   taskData.GetLocale(),
	}

	if filter.Locale == "" {
		filter.Locale = "ru"
	}

	if tIDStr := taskData.GetTeamId(); tIDStr != "" {
		if tID, err := strconv.ParseInt(tIDStr, 10, 64); err == nil {
			filter.TeamID = tID
		}
	}

	if cIDStr := taskData.GetCoachesId(); cIDStr != "" {
		if cID, err := strconv.Atoi(cIDStr); err == nil {
			filter.CoachID = cID
		}
	}

	if pIDStr := taskData.GetPlayerId(); pIDStr != "" {
		if pID, err := strconv.ParseInt(pIDStr, 10, 64); err == nil {
			filter.PlayerID = pID
		}
	}

	if sIDStr := taskData.GetSeasonsId(); sIDStr != "" {
		if sID, err := strconv.Atoi(sIDStr); err == nil {
			filter.SeasonID = sID
		}
	}

	if child := taskData.GetTaskChild(); child != "" {
		switch strings.ToLower(child) {
		case "win", "team_win":
			filter.StreakType = analytics.StreakTypeTeamWin
		case "unbeaten", "team_unbeaten":
			filter.StreakType = analytics.StreakTypeTeamUnbeaten
		case "scoring", "team_scoring":
			filter.StreakType = analytics.StreakTypeTeamScoring
		case "cleansheet", "clean_sheet", "team_clean_sheet":
			filter.StreakType = analytics.StreakTypeTeamCleanSheet
		case "nowin", "no_win", "team_no_win":
			filter.StreakType = analytics.StreakTypeTeamNoWin
		case "home":
			filter.Scope = analytics.StreakScopeHome
		case "away":
			filter.Scope = analytics.StreakScopeAway
		}
	}

	return filter
}

