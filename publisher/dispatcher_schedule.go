package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/copywriter"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/renderer"
	"github.com/Pastir/footballdata/viewmodel"
	"go.uber.org/zap"
)

// processRoundFixturesForLeague выполняет генерацию расписания, выбор центрального матча,
// рендеринг инфографики и публикацию в Telegram для указанной лиги.
func processRoundFixturesForLeague(
	ctx context.Context,
	taskData config.TaskData,
	leagueID int,
	outputPath string,
	shouldPin bool,
	log *zap.Logger,
) (int64, error) {
	timezone := taskData.GetTimezone()
	pool := db.DBx.PGxPool
	if pool == nil {
		return 0, fmt.Errorf("database pool is not initialized")
	}

	info, err := analytics.GetLeagueSeasonInfo(ctx, pool, leagueID)
	if err != nil {
		return 0, fmt.Errorf("GetLeagueSeasonInfo(league %d): %w", leagueID, err)
	}
	info.Locale = taskData.GetLocale()

	matches, err := analytics.GetNextRoundFixtures(ctx, pool, &info)
	if err != nil {
		return 0, fmt.Errorf("GetNextRoundFixtures(league %d): %w", leagueID, err)
	}

	if len(matches) == 0 {
		return 0, fmt.Errorf("no upcoming round fixtures found for league %d", leagueID)
	}

	roundID := matches[0].RoundID
	roundName := matches[0].RoundName

	// Проверка окна публикации: за 48 часов до starting_at первого матча тура
	firstMatchStart := matches[0].FixtureStart
	for _, m := range matches {
		if m.FixtureStart.Before(firstMatchStart) {
			firstMatchStart = m.FixtureStart
		}
	}

	const scheduleWindow = 48 * time.Hour
	now := time.Now()
	windowStart := firstMatchStart.Add(-scheduleWindow)

	// В live-режиме (-publish=true) без флага -force контролируем попадание в 48ч-окно контентной сетки
	if taskData.GetPublish() && !taskData.GetForce() {
		if now.Before(windowStart) {
			log.Info("too early to publish round fixtures (scheduled 48h before kickoff)",
				zap.Int("league_id", leagueID),
				zap.Time("first_match", firstMatchStart),
				zap.Time("window_start", windowStart),
			)
			fmt.Printf("⏳ Лига %s (%d): публикация расписания откроется %s (за 48ч до первого матча %s). Для принудительной отправки укажите -force\n",
				info.LeagueName, leagueID, windowStart.Format("02.01 15:04"), firstMatchStart.Format("02.01 15:04"))
			return 0, nil
		}
		if now.After(firstMatchStart) {
			log.Info("round already started, skipping fixtures publication",
				zap.Int("league_id", leagueID),
				zap.Time("first_match", firstMatchStart),
			)
			fmt.Printf("ℹ️ Лига %s (%d): первый матч тура уже стартовал (%s). Для отправки укажите -force\n",
				info.LeagueName, leagueID, firstMatchStart.Format("02.01 15:04"))
			return 0, nil
		}
	} else if !taskData.GetPublish() {
		if now.Before(windowStart) {
			fmt.Printf("ℹ️ [%s (%d): до 48ч-окна публикации расписания осталось %s (откроется %s)]\n",
				info.LeagueName, leagueID, time.Until(windowStart).Round(time.Minute), windowStart.Format("02.01 15:04"))
		}
	}

	// Выбираем центральный «Матч тура» с помощью математического движка (с кэшированием)
	candidates := analytics.BuildMatchCandidates(matches)
	matchOfRound, err := analytics.GetOrComputeMatchOfRound(ctx, pool, &info, candidates)
	if err != nil {
		log.Warn("GetOrComputeMatchOfRound failed", zap.Int("league_id", leagueID), zap.Error(err))
	}

	// Формируем ViewModel, передавая ID вычисленного матча тура для центральной карточки
	vm := viewmodel.BuildRoundFixturesViewModel(info, matches, timezone, matchOfRound.FixtureID)

	templatePath := filepath.Join("template", "round_fixtures_template.html")
	log.Info("rendering round fixtures PNG", zap.Int("league_id", leagueID), zap.String("output", outputPath))

	if err := renderer.RenderRoundFixturesPNG(vm, templatePath, outputPath); err != nil {
		log.Warn("RenderRoundFixturesPNG failed, proceeding with text", zap.Int("league_id", leagueID), zap.Error(err))
	} else {
		log.Info("round fixtures PNG rendered", zap.Int("league_id", leagueID), zap.String("file", outputPath))
	}

	// Сбор предматчевых данных для каждого матча (H2H до 10 матчей, профиль команд Hero-статистики)
	log.Info("collecting pre-match H2H and team statistics for Mini App", zap.Int("league_id", leagueID), zap.Int("matches", len(matches)))

	profileCache := make(map[int64]analytics.TeamPreMatchStats)
	getProfile := func(teamID int64) analytics.TeamPreMatchStats {
		if p, ok := profileCache[teamID]; ok {
			return p
		}
		p, _ := analytics.GetTeamPreMatchProfile(ctx, pool, teamID, info.SeasonID)
		profileCache[teamID] = p
		return p
	}

	var exportMatches []analytics.RoundResultMatch
	for _, m := range matches {
		h2h, h2hErr := analytics.GetH2HMatches(ctx, pool, m.TeamHomeID, m.TeamAwayID, info.Locale, 10)
		if h2hErr != nil {
			log.Warn("GetH2HMatches failed", zap.Int64("fixture_id", m.FixtureID), zap.Error(h2hErr))
		}

		homeProfile := getProfile(m.TeamHomeID)
		awayProfile := getProfile(m.TeamAwayID)

		matchEntry := analytics.RoundResultMatch{
			LeagueID:     info.LeagueID,
			FixtureID:    m.FixtureID,
			LeagueName:   info.LeagueName,
			SeasonID:     info.SeasonID,
			SeasonName:   info.SeasonName,
			RoundID:      m.RoundID,
			RoundName:    m.RoundName,
			FixtureStart: m.FixtureStart,
			VenueID:      m.VenueID,
			VenueName:    m.VenueName,
			CountryID:    m.CountryID,
			TeamHomeID:   m.TeamHomeID,
			TeamHomeName: m.TeamHomeName,
			TeamHomeLogo: m.TeamHomeLogo,
			TeamAwayID:   m.TeamAwayID,
			TeamAwayName: m.TeamAwayName,
			TeamAwayLogo: m.TeamAwayLogo,
			StateShort:   m.StateShort,
			Scores:       analytics.MatchScores{},
			Odds: analytics.MatchOdds{
				HomeOdds:      m.OddsHome,
				DrawOdds:      m.OddsDraw,
				AwayOdds:      m.OddsAway,
				BookmakerName: m.BookmakerName,
			},
			H2H: &h2h,
			PreMatchStats: &analytics.PreMatchStatsComparison{
				Home: homeProfile,
				Away: awayProfile,
			},
			IsRival: m.IsRival,
		}

		exportMatches = append(exportMatches, matchEntry)
	}

	// Загружаем актуальную турнирную таблицу
	standings, _ := analytics.GetCurrentStandings(ctx, pool, &info)

	// Формируем payload для Mini App
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
		Summary:       analytics.RoundSummary{TotalMatches: len(matches)},
		Matches:       exportMatches,
		Standings:     standings,
	}

	payloadJSON, err := json.MarshalIndent(payload, "", "  ")
	if err == nil {
		_ = os.MkdirAll("web/viewer/data", 0755)
		_ = os.WriteFile("web/viewer/round_data.json", payloadJSON, 0644)
		_ = os.WriteFile(fmt.Sprintf("web/viewer/data/%d_%d_%d.json", info.LeagueID, info.SeasonID, roundID), payloadJSON, 0644)

		if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
			publicDir := filepath.Dir(config.Config.WebApp.MediaDir)
			viewerDir := filepath.Join(publicDir, "viewer")
			_ = os.MkdirAll(viewerDir, 0755)

			mediaRoundDir := filepath.Join(config.Config.WebApp.MediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID), strconv.Itoa(roundID))
			_ = os.MkdirAll(mediaRoundDir, 0755)
			_ = os.WriteFile(filepath.Join(mediaRoundDir, "round_data.json"), payloadJSON, 0644)

			_ = os.WriteFile(filepath.Join(viewerDir, "round_data.json"), payloadJSON, 0644)
			_ = os.WriteFile(filepath.Join(publicDir, "round_data.json"), payloadJSON, 0644)

			viewerDataDir := filepath.Join(viewerDir, "data")
			_ = os.MkdirAll(viewerDataDir, 0755)
			_ = os.WriteFile(filepath.Join(viewerDataDir, fmt.Sprintf("%d_%d_%d.json", info.LeagueID, info.SeasonID, roundID)), payloadJSON, 0644)

			syncViewerFiles(publicDir)
		}
	}

	// Формируем лаконичный моноширинный текст с кликабельными ссылками на котировках
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

	postCfg := copywriter.RoundFixturesPostConfig{
		LeagueName:   info.LeagueName,
		RoundName:    vm.RoundTitle,
		LeagueID:     info.LeagueID,
		SeasonID:     info.SeasonID,
		RoundID:      roundID,
		BotUsername:  botUser,
		BotShortName: shortName,
		BaseURL:      baseURL,
	}

	postText := copywriter.GenerateRoundFixturesHTMLPost(postCfg, vm)

	if !taskData.GetPublish() {
		fmt.Printf("\n=== [Dry Run] Текст расписания тура (%s) ===\n\n%s\n\n", info.LeagueName, postText)
		return 0, nil
	}

	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return 0, fmt.Errorf("telegram configuration is missing (BotToken or ChannelID is empty)")
	}

	// В новом нативном стиле Telegram ссылки на Mini App размещены внутри текста поста,
	// а инлайн-клавиатура не передается (nil), чтобы Telegram отображал свою нативную плашку
	// комментариев («Прокомментировать» / «N комментариев») с отдельным изолированным тредом.
	var keyboard *InlineKeyboardMarkup = nil

	channelID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
	storage := getStorage(ctx)

	// Проверка идемпотентности: защита от случайных дубликатов публикаций
	if storage != nil && !taskData.GetForce() {
		already, err := storage.IsAlreadyPublished(ctx, channelID, PubTypeRoundFixtures, info.LeagueID, roundID)
		if err == nil && already {
			log.Info("round fixtures already published, skipping duplicate publication (use -force to override)",
				zap.String("type", string(PubTypeRoundFixtures)),
				zap.Int("league_id", info.LeagueID),
				zap.Int("round_id", roundID),
			)
			fmt.Printf("ℹ️ Расписание тура (лига %s [%d], тур %d) уже было опубликовано в канале %s. Для повторной отправки укажите флаг -force\n",
				info.LeagueName, info.LeagueID, roundID, tg.ChannelID)
			return 0, nil
		}
	}

	pubCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var (
		msgID  int64
		pubErr error
	)

	// Отключаем превью ссылок со счетов/котировок
	linkPreviewOpt := &LinkPreviewOptions{
		IsDisabled: true,
	}

	// Если есть сгенерированный PNG файл, отправляем с фото
	if _, statErr := os.Stat(outputPath); statErr == nil {
		msgID, pubErr = SendPhotoWithKeyboard(pubCtx, tg.BotToken, tg.ChannelID, postText, outputPath, keyboard)
	} else {
		msgID, pubErr = SendTextMessageWithKeyboardAndPreview(pubCtx, tg.BotToken, tg.ChannelID, postText, keyboard, linkPreviewOpt)
	}

	if pubErr != nil {
		return 0, fmt.Errorf("SendToTelegram (league %d) failed: %w", leagueID, pubErr)
	}

	log.Info("round fixtures published successfully",
		zap.Int("league_id", leagueID),
		zap.String("channel", tg.ChannelID),
		zap.Int64("message_id", msgID),
	)
	fmt.Printf("✅ Расписание тура (%s) опубликовано в Telegram (%s)! Message ID: %d\n", info.LeagueName, tg.ChannelID, msgID)

	// Автоматическое тихое закрепление (Silent Pin): только для флагманской лиги
	if shouldPin {
		if pinErr := PinChatMessage(pubCtx, tg.BotToken, tg.ChannelID, msgID, true); pinErr != nil {
			log.Warn("failed to pin round fixtures message", zap.Int("league_id", leagueID), zap.Error(pinErr))
		} else {
			log.Info("round fixtures message successfully pinned in channel", zap.Int("league_id", leagueID), zap.Int64("message_id", msgID))
			fmt.Printf("📌 Пост расписания %s тихо закреплен в канале (Silent Pin)\n", info.LeagueName)
		}
	}

	// Фиксация в БД telegram_publications
	if storage != nil {
		pub := &Publication{
			ChannelID:       channelID,
			MessageID:       &msgID,
			PublicationType: PubTypeRoundFixtures,
			LeagueID:        &info.LeagueID,
			SeasonID:        &info.SeasonID,
			RoundID:         &roundID,
			Title:           fmt.Sprintf("%s — %s: Расписание тура", info.LeagueName, vm.RoundTitle),
			TextContent:     postText,
			MediaURL:        outputPath,
			MediaType:       MediaTypePhoto,
			Status:          StatusPublished,
		}
		_, _ = storage.Create(ctx, pub)
	}

	return msgID, nil
}

// dispatchRoundFixtures формирует и публикует расписание тура одной лиги.
func dispatchRoundFixtures(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	leagueID := taskData.GetLeagueId()
	if leagueID == 0 {
		leagueID = 8
	}

	outputPath := taskData.GetOutput()
	if outputPath == "" {
		if leagueID == 8 {
			outputPath = "round_fixtures.png"
		} else {
			outputPath = fmt.Sprintf("round_fixtures_%d.png", leagueID)
		}
	}

	// Для одиночного вызова тихо закрепляем, если это АПЛ (8)
	shouldPin := (leagueID == 8)

	_, err := processRoundFixturesForLeague(ctx, taskData, leagueID, outputPath, shouldPin, log)
	return err
}

type centralMatchInfo struct {
	LeagueName   string
	RoundTitle   string
	Flag         string
	HomeTeam     string
	AwayTeam     string
	HomeRank     int
	AwayRank     int
	KickoffTime  string
	KickoffDate  string
	Venue        string
	OddsHome     string
	OddsDraw     string
	OddsAway     string
	CardPath     string
}

type scheduleMatchItem struct {
	LeagueOrder int
	LeagueID    int
	LeagueName  string
	LeagueFlag  string
	KickoffTime string
	KickoffDate string
	HomeTeam    string
	AwayTeam    string
	OddsHome    string
	OddsDraw    string
	OddsAway    string
	StartUTC    time.Time
}

// dispatchWeekendHub формирует двухэтапную публикацию расписания топ-5 лиг на уикенд:
// Этап 1: Альбом из 5 карточек центральных матчей тура (по одной на лигу) с компактной подписью (< 750 символов).
// Этап 2: Чистое текстовое расписание всех матчей уикенда (сортировка: День -> Лига -> Время в UTC),
// с кнопкой в Mini App и тихим закрепом (Silent Pin).
func dispatchWeekendHub(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tg := config.Config.TG
	if tg == nil {
		return fmt.Errorf("telegram configuration is missing")
	}

	top5Leagues := []struct {
		id   int
		name string
		flag string
	}{
		{id: 8, name: "Премьер-лига", flag: "🏴󠁧󠁢󠁥󠁮󠁧󠁿"},
		{id: 564, name: "Ла Лига", flag: "🇪🇸"},
		{id: 384, name: "Серия А", flag: "🇮🇹"},
		{id: 82, name: "Бундеслига", flag: "🇩🇪"},
		{id: 301, name: "Лига 1", flag: "🇫🇷"},
	}

	var centralMatches []centralMatchInfo
	var allMatches []scheduleMatchItem
	var cardPaths []string

	pubType := taskData.GetType()
	topMatchesHub := (pubType == "topMatches" || pubType == "top5Matches" || pubType == "centralMatches" || pubType == "topMatchesHub")
	topMatchesAlbumOnly := (pubType == "topMatchesAlbum" || pubType == "albumOnly" || pubType == "weekendAlbum")
	topMatchesTableOnly := (pubType == "topMatchesTable" || pubType == "topMatchesNative")

	scheduleOnly := (pubType == "weekendSchedule" || pubType == "scheduleOnly" || pubType == "nativeTable" || pubType == "richTable" || pubType == "tableNative" || topMatchesTableOnly)
	albumOnly := (topMatchesAlbumOnly)

	log.Info("starting weekend hub generation for top-5 leagues",
		zap.Bool("publish", taskData.GetPublish()),
		zap.Bool("force", taskData.GetForce()),
		zap.Bool("schedule_only", scheduleOnly),
		zap.Bool("album_only", albumOnly),
	)
	fmt.Printf("🚀 Запуск подготовки сводного уикенд-гида для 5 топ-лиг (Publish=%v, Force=%v, ScheduleOnly=%v, AlbumOnly=%v)...\n",
		taskData.GetPublish(), taskData.GetForce(), scheduleOnly, albumOnly)

	flagshipLeagueID := 8
	flagshipSeasonID := 0
	flagshipRoundID := taskData.GetRoundId()
	if fInfo, fErr := analytics.GetLeagueSeasonInfo(ctx, pool, flagshipLeagueID); fErr == nil {
		flagshipSeasonID = fInfo.SeasonID
		if flagshipRoundID == 0 {
			if fInfo.NextRoundID > 0 {
				flagshipRoundID = fInfo.NextRoundID
			} else {
				flagshipRoundID = fInfo.CurrRoundID
			}
		}
	}

	if taskData.GetPublish() && !taskData.GetForce() && flagshipRoundID > 0 {
		if storage := getStorage(ctx); storage != nil {
			cID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
			if already, err := storage.IsAlreadyPublished(ctx, cID, PubTypeRoundFixtures, flagshipLeagueID, flagshipRoundID); err == nil && already {
				log.Info("weekend hub already published, skipping duplicate",
					zap.Int("league_id", flagshipLeagueID),
					zap.Int("round_id", flagshipRoundID),
				)
				fmt.Printf("ℹ️ Уикенд-хаб (лига %d, тур %d) уже был опубликован. Для повторной отправки укажите -force\n", flagshipLeagueID, flagshipRoundID)
				return nil
			}
		}
	}

	for idx, l := range top5Leagues {
		info, err := analytics.GetLeagueSeasonInfo(ctx, pool, l.id)
		if err != nil {
			log.Error("failed to get league season info", zap.Int("league_id", l.id), zap.Error(err))
			continue
		}
		info.Locale = "ru"

		matches, err := analytics.GetNextRoundFixtures(ctx, pool, &info)
		if err != nil || len(matches) == 0 {
			log.Error("no upcoming fixtures found for league", zap.Int("league_id", l.id), zap.Error(err))
			continue
		}

		// Защита от пауз на матчи сборных: если первый матч тура лиги более чем через 96 часов (4 суток)
		// и не передан флаг -force, пропускаем эту лигу, так как на текущем уикенде она не играет.
		if !taskData.GetForce() {
			firstMatchStart := matches[0].FixtureStart
			for _, m := range matches {
				if m.FixtureStart.Before(firstMatchStart) {
					firstMatchStart = m.FixtureStart
				}
			}
			if time.Until(firstMatchStart) > 96*time.Hour {
				log.Info("skipping league in weekend hub: next round is on international/cup break (>96h away)",
					zap.Int("league_id", l.id),
					zap.String("league_name", l.name),
					zap.Time("first_match_utc", firstMatchStart.UTC()),
				)
				fmt.Printf("⏳ [%s] Пропуск: ближайший тур начнётся только %s UTC (пауза в чемпионате)\n",
					l.name, firstMatchStart.UTC().Format("02.01 15:04"))
				continue
			}
		}

		// Standings map for ranks
		standingsMap := make(map[int64]int)
		if standings, sErr := analytics.GetCurrentStandings(ctx, pool, &info); sErr == nil {
			for _, st := range standings {
				standingsMap[st.TeamID] = st.Position
			}
		}

		candidates := analytics.BuildMatchCandidates(matches)
		mor, _ := analytics.GetOrComputeMatchOfRound(ctx, pool, &info, candidates)

		// Find central match
		var featuredMatch *analytics.NextRoundMatch
		for i := range matches {
			if matches[i].FixtureID == mor.FixtureID {
				featuredMatch = &matches[i]
				break
			}
		}
		if featuredMatch == nil && len(matches) > 0 {
			featuredMatch = &matches[0]
		}

		// Collect all matches of this league for schedule
		for _, m := range matches {
			allMatches = append(allMatches, scheduleMatchItem{
				LeagueOrder: idx,
				LeagueID:    l.id,
				LeagueName:  l.name,
				LeagueFlag:  l.flag,
				KickoffTime: m.FixtureStart.UTC().Format("15:04"),
				KickoffDate: m.FixtureStart.UTC().Format("2006-01-02"),
				HomeTeam:    m.TeamHomeName,
				AwayTeam:    m.TeamAwayName,
				OddsHome:    m.OddsHome,
				OddsDraw:    m.OddsDraw,
				OddsAway:    m.OddsAway,
				StartUTC:    m.FixtureStart.UTC(),
			})
		}

		if featuredMatch != nil {
			cardPath := fmt.Sprintf("match_card_%d.png", l.id)
			hRank := 0
			if featuredMatch.PositionHome != nil {
				hRank = *featuredMatch.PositionHome
			} else if r, ok := standingsMap[featuredMatch.TeamHomeID]; ok {
				hRank = r
			}
			aRank := 0
			if featuredMatch.PositionAway != nil {
				aRank = *featuredMatch.PositionAway
			} else if r, ok := standingsMap[featuredMatch.TeamAwayID]; ok {
				aRank = r
			}

			venue := "Стадион"
			if featuredMatch.VenueName != nil && *featuredMatch.VenueName != "" {
				venue = *featuredMatch.VenueName
			}

			// Render match card PNG
			headerVM := viewmodel.MatchHeaderViewModel{
				LeagueName:   strings.ToUpper(l.name),
				RoundTitle:   fmt.Sprintf("ТУР %s", featuredMatch.RoundName),
				BadgeTitle:   "МАТЧ ТУРА",
				HomeTeamName: strings.ToUpper(featuredMatch.TeamHomeName),
				HomeTeamCode: getTeamShortCode(featuredMatch.TeamHomeName),
				HomeTeamLogo: featuredMatch.TeamHomeLogo,
				AwayTeamName: strings.ToUpper(featuredMatch.TeamAwayName),
				AwayTeamCode: getTeamShortCode(featuredMatch.TeamAwayName),
				AwayTeamLogo: featuredMatch.TeamAwayLogo,
				KickoffTime:  featuredMatch.FixtureStart.UTC().Format("15:04"),
				KickoffDate:  featuredMatch.FixtureStart.UTC().Format("02.01"),
				Venue:        venue,
				Timezone:     "UTC",
				HomeRank:     hRank,
				AwayRank:     aRank,
				OddsHome:     featuredMatch.OddsHome,
				OddsDraw:     featuredMatch.OddsDraw,
				OddsAway:     featuredMatch.OddsAway,
			}

			tmplPath := filepath.Join("template", "match_header_template.html")
			if err := renderer.RenderMatchHeaderPNG(headerVM, tmplPath, cardPath); err != nil {
				log.Warn("failed to render match card PNG", zap.Int("league_id", l.id), zap.Error(err))
			}

			cardPaths = append(cardPaths, cardPath)
			centralMatches = append(centralMatches, centralMatchInfo{
				LeagueName:  l.name,
				RoundTitle:  fmt.Sprintf("ТУР %s", featuredMatch.RoundName),
				Flag:        l.flag,
				HomeTeam:    featuredMatch.TeamHomeName,
				AwayTeam:    featuredMatch.TeamAwayName,
				HomeRank:    hRank,
				AwayRank:    aRank,
				KickoffTime: featuredMatch.FixtureStart.UTC().Format("15:04"),
				Venue:       venue,
				OddsHome:    featuredMatch.OddsHome,
				OddsDraw:    featuredMatch.OddsDraw,
				OddsAway:    featuredMatch.OddsAway,
				CardPath:    cardPath,
			})
		}
	}

	if len(cardPaths) == 0 && !scheduleOnly {
		if !taskData.GetForce() {
			fmt.Println("ℹ️ Пауза в чемпионатах (игры сборных): матчей Топ-5 лиг на ближайший уикенд нет, публикация пропущена.")
			return nil
		}
		return fmt.Errorf("no match cards could be generated for weekend hub")
	}
	if len(allMatches) == 0 {
		fmt.Println("ℹ️ Пауза в чемпионатах (игры сборных): матчей Топ-5 лиг на ближайший уикенд нет, публикация пропущена.")
		return nil
	}

	// 1. Формирование подписи к медиа-альбому (< 750 символов)
	var albumCap strings.Builder
	if !scheduleOnly {
		if topMatchesHub || !albumOnly {
			albumCap.WriteString("🔥 <b>ГЛАВНЫЕ МАТЧИ УИКЕНДА В ТОП-5 ЛИГАХ</b>\n")
			albumCap.WriteString("⬇️ <i>Время начала (UTC) и котировки 1·X·2 — в таблице ниже.</i>")
		} else {
			albumCap.WriteString("🔥 <b>ГЛАВНЫЕ МАТЧИ УИКЕНДА В ТОП-5 ЛИГАХ</b>\n")
			albumCap.WriteString("<i>Время начала указано по UTC</i>\n\n")

			teamColWidth := 11
			albumCap.WriteString("<pre>┌──────┬─────────────┬───┬─────────────┬────────────────┐\n")
			albumCap.WriteString("│Время │   Хозяева   │   │    Гости    │    1   X   2   │\n")
			albumCap.WriteString("├──────┼─────────────┼───┼─────────────┼────────────────┤\n")
			for _, cm := range centralMatches {
				oddsStr := "- - -"
				if cm.OddsHome != "" && cm.OddsDraw != "" && cm.OddsAway != "" {
					oddsStr = fmt.Sprintf("%s %s %s", cm.OddsHome, cm.OddsDraw, cm.OddsAway)
				}
				hPadded := padLeft(cm.HomeTeam, teamColWidth)
				aPadded := padRight(cm.AwayTeam, teamColWidth)
				albumCap.WriteString(fmt.Sprintf("│%s │ %s │ : │ %s │ %s │\n", cm.KickoffTime, hPadded, aPadded, oddsStr))
			}
			albumCap.WriteString("└──────┴─────────────┴───┴─────────────┴────────────────┘</pre>\n")
			albumCap.WriteString("⚽ <i>Подробная аналитика и xG всех матчей — в нашем Mini App.</i>")
		}
	}

	// 2. Сортировка матчей уикенда: Дата -> Порядок лиги (АПЛ, Ла Лига, Серия А, Бундеслига, Лига 1) -> Время (UTC)
	sort.Slice(allMatches, func(i, j int) bool {
		if allMatches[i].KickoffDate != allMatches[j].KickoffDate {
			return allMatches[i].KickoffDate < allMatches[j].KickoffDate
		}
		if allMatches[i].LeagueOrder != allMatches[j].LeagueOrder {
			return allMatches[i].LeagueOrder < allMatches[j].LeagueOrder
		}
		return allMatches[i].StartUTC.Before(allMatches[j].StartUTC)
	})

	// Инлайн-клавиатура отключена (nil), чтобы отображалась нативная плашка комментариев Telegram.
	var keyboard *InlineKeyboardMarkup = nil

	if taskData.GetPublish() {
		channelID := tg.ChannelID
		botToken := tg.BotToken

		var albumMsgIDs []int64
		if !scheduleOnly {
			fmt.Printf("📤 Отправка Этапа 1: Медиа-альбом из %d карточек центральных матчей с табличной подписью...\n", len(cardPaths))
			var err error
			albumMsgIDs, err = SendMediaGroup(ctx, botToken, channelID, cardPaths, albumCap.String())
			if err != nil {
				return fmt.Errorf("sendMediaGroup failed: %w", err)
			}
			fmt.Printf("✅ Альбом опубликован (сообщений: %d, первое ID=%d)\n", len(albumMsgIDs), albumMsgIDs[0])

			time.Sleep(1500 * time.Millisecond)
		} else {
			fmt.Println("ℹ️ Пропуск Этапа 1 (Альбом): режим публикации только расписания.")
		}

		var schedMsgID int64
		if !albumOnly {
			if topMatchesTableOnly || topMatchesHub {
				fmt.Printf("📤 Отправка нативной таблицы топ-матчей...\n")
				richPayload := buildTopMatchesOnlyRichTable(centralMatches)
				var err error
				schedMsgID, err = SendRichMessage(ctx, botToken, channelID, richPayload, keyboard)
				if err != nil {
					return fmt.Errorf("sendRichMessage failed: %w", err)
				}
				fmt.Printf("✅ Нативная таблица топ-матчей опубликована (ID=%d)\n", schedMsgID)
			} else {
				fmt.Printf("📤 Отправка Этапа 2: Полное расписание уикенда в нативной таблице...\n")
				richPayload := buildWeekendScheduleRichTables(centralMatches, allMatches)
				var err error
				schedMsgID, err = SendRichMessage(ctx, botToken, channelID, richPayload, keyboard)
				if err != nil {
					return fmt.Errorf("sendRichMessage failed: %w", err)
				}
				fmt.Printf("✅ Нативная таблица расписания опубликована (ID=%d)\n", schedMsgID)

				// Тихий закреп расписания
				if pinErr := PinChatMessage(ctx, botToken, channelID, schedMsgID, true); pinErr != nil {
					log.Warn("failed to pin schedule message", zap.Error(pinErr))
				} else {
					fmt.Printf("📌 Сообщение с нативной таблицей тихо закреплено в канале (Silent Pin)\n")
				}
			}
		} else {
			fmt.Println("ℹ️ Пропуск Этапа 2 (Расписание): режим публикации только альбома.")
		}

		// Фиксация в БД telegram_publications
		storage := getStorage(ctx)
		if storage != nil {
			cID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
			if !scheduleOnly && len(albumMsgIDs) > 0 {
				firstMsg := albumMsgIDs[0]
				_, _ = storage.Create(ctx, &Publication{
					ChannelID:       cID,
					MessageID:       &firstMsg,
					PublicationType: PubTypeRoundFixtures,
					LeagueID:        &flagshipLeagueID,
					SeasonID:        &flagshipSeasonID,
					RoundID:         &flagshipRoundID,
					Title:           "Главные матчи уикенда (Топ-5 лиг) — Альбом",
					TextContent:     albumCap.String(),
					MediaType:       MediaTypePhoto,
					Status:          StatusPublished,
				})
			}
			if !albumOnly && schedMsgID > 0 {
				schedTitle := "Расписание уикенда (Топ-5 лиг) — Нативная таблица"
				if topMatchesTableOnly || topMatchesHub {
					schedTitle = "Главные матчи уикенда (Топ-5 лиг) — Нативная таблица"
				}
				_, _ = storage.Create(ctx, &Publication{
					ChannelID:       cID,
					MessageID:       &schedMsgID,
					PublicationType: PubTypeRoundFixtures,
					LeagueID:        &flagshipLeagueID,
					SeasonID:        &flagshipSeasonID,
					RoundID:         &flagshipRoundID,
					Title:           schedTitle,
					TextContent:     "Native Rich Message Table",
					MediaType:       MediaTypeText,
					Status:          StatusPublished,
				})
			}
		}
	} else {
		fmt.Println("ℹ️ [DRY RUN] Публикация отключена (-publish=false).")
		if !scheduleOnly {
			fmt.Printf("\n--- ПОДПИСЬ К АЛЬБОМУ (%d симв.) ---\n%s\n", utf8.RuneCountInString(albumCap.String()), albumCap.String())
		}
		if !albumOnly {
			if topMatchesTableOnly || topMatchesHub {
				richPayload := buildTopMatchesOnlyRichTable(centralMatches)
				fmt.Printf("\n--- НАТИВНАЯ ТАБЛИЦА ТОП-МАТЧЕЙ (%d блоков) ---\n", len(richPayload.Blocks))
			} else {
				richPayload := buildWeekendScheduleRichTables(centralMatches, allMatches)
				fmt.Printf("\n--- НАТИВНАЯ ТАБЛИЦА РАСПИСАНИЯ (%d блоков) ---\n", len(richPayload.Blocks))
			}
		}
	}

	return nil
}

func padLeft(s string, width int) string {
	rCount := utf8.RuneCountInString(s)
	if rCount >= width {
		return s
	}
	return strings.Repeat(" ", width-rCount) + s
}

func padRight(s string, width int) string {
	rCount := utf8.RuneCountInString(s)
	if rCount >= width {
		return s
	}
	return s + strings.Repeat(" ", width-rCount)
}

func buildWeekendScheduleRichTables(centralMatches []centralMatchInfo, allMatches []scheduleMatchItem) RichMessagePayload {
	var blocks []RichBlock

	if len(centralMatches) > 0 {
		blocks = append(blocks, RichBlock{
			Type: "heading",
			Size: 3,
			Text: "🔥 ГЛАВНЫЕ МАТЧИ УИКЕНДА (ТОП-5 ЛИГ)",
		})

		var centralCells [][]RichBlockTableCell
		centralCells = append(centralCells, []RichBlockTableCell{
			{Text: "Время", IsHeader: true, Align: "center"},
			{Text: "Хозяева", IsHeader: true, Align: "right"},
			{Text: "", IsHeader: true, Align: "center"},
			{Text: "Гости", IsHeader: true, Align: "left"},
			{Text: "1 · X · 2", IsHeader: true, Align: "center"},
		})

		for _, cm := range centralMatches {
			oddsStr := "-"
			if cm.OddsHome != "" && cm.OddsDraw != "" && cm.OddsAway != "" {
				oddsStr = fmt.Sprintf("%s · %s · %s", cm.OddsHome, cm.OddsDraw, cm.OddsAway)
			}
			timeBadge := fmt.Sprintf("%s %s", cm.Flag, cm.KickoffTime)
			centralCells = append(centralCells, []RichBlockTableCell{
				{Text: timeBadge, Align: "center"},
				{Text: cm.HomeTeam, Align: "right"},
				{Text: ":", Align: "center"},
				{Text: cm.AwayTeam, Align: "left"},
				{Text: oddsStr, Align: "center"},
			})
		}

		blocks = append(blocks, RichBlock{
			Type:       "table",
			IsBordered: true,
			IsStriped:  true,
			IsCompact:  true,
			Cells:      centralCells,
		})
	}

	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 3,
		Text: "📅 РАСПИСАНИЕ МАТЧЕЙ УИКЕНДА (UTC)",
	})

	daysRu := []string{"Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота"}
	monthsRu := []string{
		"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}

	matchesByDate := make(map[string][]scheduleMatchItem)
	var orderedDates []string
	for _, m := range allMatches {
		if _, exists := matchesByDate[m.KickoffDate]; !exists {
			orderedDates = append(orderedDates, m.KickoffDate)
		}
		matchesByDate[m.KickoffDate] = append(matchesByDate[m.KickoffDate], m)
	}

	for _, date := range orderedDates {
		dayMatches := matchesByDate[date]
		firstMatch := dayMatches[0]
		weekday := daysRu[int(firstMatch.StartUTC.Weekday())]
		mNum := int(firstMatch.StartUTC.Month())
		monthStr := ""
		if mNum >= 1 && mNum <= 12 {
			monthStr = monthsRu[mNum]
		}
		dayTitle := fmt.Sprintf("🗓 %s, %d %s", strings.ToUpper(weekday), firstMatch.StartUTC.Day(), strings.ToUpper(monthStr))

		blocks = append(blocks, RichBlock{
			Type: "heading",
			Size: 4,
			Text: dayTitle,
		})

		var tableCells [][]RichBlockTableCell
		currentLeague := -1

		for _, m := range dayMatches {
			if m.LeagueOrder != currentLeague {
				currentLeague = m.LeagueOrder
				leagueTitle := fmt.Sprintf("%s %s", m.LeagueFlag, strings.ToUpper(m.LeagueName))
				tableCells = append(tableCells, []RichBlockTableCell{
					{
						Text:     leagueTitle,
						IsHeader: true,
						Colspan:  5,
						Align:    "left",
					},
				})
			}

			oddsStr := "-"
			if m.OddsHome != "" && m.OddsDraw != "" && m.OddsAway != "" {
				oddsStr = fmt.Sprintf("%s · %s · %s", m.OddsHome, m.OddsDraw, m.OddsAway)
			}

			tableCells = append(tableCells, []RichBlockTableCell{
				{Text: m.KickoffTime, Align: "center"},
				{Text: m.HomeTeam, Align: "right"},
				{Text: ":", Align: "center"},
				{Text: m.AwayTeam, Align: "left"},
				{Text: oddsStr, Align: "center"},
			})
		}

		blocks = append(blocks, RichBlock{
			Type:       "table",
			IsBordered: true,
			IsStriped:  true,
			IsCompact:  true,
			Cells:      tableCells,
		})
	}

	return RichMessagePayload{Blocks: blocks}
}

func buildTopMatchesOnlyRichTable(centralMatches []centralMatchInfo) RichMessagePayload {
	var blocks []RichBlock

	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 3,
		Text: "🔥 ГЛАВНЫЕ МАТЧИ УИКЕНДА (ТОП-5 ЛИГ)",
	})

	var centralCells [][]RichBlockTableCell
	centralCells = append(centralCells, []RichBlockTableCell{
		{Text: "Время", IsHeader: true, Align: "center"},
		{Text: "Хозяева", IsHeader: true, Align: "right"},
		{Text: "", IsHeader: true, Align: "center"},
		{Text: "Гости", IsHeader: true, Align: "left"},
		{Text: "1 · X · 2", IsHeader: true, Align: "center"},
	})

	for _, cm := range centralMatches {
		oddsStr := "-"
		if cm.OddsHome != "" && cm.OddsDraw != "" && cm.OddsAway != "" {
			oddsStr = fmt.Sprintf("%s · %s · %s", cm.OddsHome, cm.OddsDraw, cm.OddsAway)
		}
		timeBadge := fmt.Sprintf("%s %s", cm.Flag, cm.KickoffTime)
		centralCells = append(centralCells, []RichBlockTableCell{
			{Text: timeBadge, Align: "center"},
			{Text: cm.HomeTeam, Align: "right"},
			{Text: ":", Align: "center"},
			{Text: cm.AwayTeam, Align: "left"},
			{Text: oddsStr, Align: "center"},
		})
	}

	blocks = append(blocks, RichBlock{
		Type:       "table",
		IsBordered: true,
		IsStriped:  true,
		IsCompact:  true,
		Cells:      centralCells,
	})

	return RichMessagePayload{Blocks: blocks}
}

