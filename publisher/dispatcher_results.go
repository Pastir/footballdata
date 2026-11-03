package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/copywriter"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/renderer"
	"github.com/Pastir/footballdata/viewmodel"
	"go.uber.org/zap"
)

// dispatchTable формирует и публикует турнирную таблицу.
func dispatchTable(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	leagueID := taskData.GetLeagueId()
	if leagueID == 0 {
		leagueID = 8 // Default: Premier League
	}

	// Один вызов — все данные лиги, сезона и туров
	info, err := analytics.GetLeagueSeasonInfo(ctx, pool, leagueID)
	if err != nil {
		return fmt.Errorf("GetLeagueSeasonInfo: %w", err)
	}
	info.Locale = taskData.GetLocale()

	// Ручное переопределение тура через CLI-флаг
	if taskData.GetRoundId() > 0 {
		info.CurrRoundID = taskData.GetRoundId()
		info.PrevRoundID = 0 // при ручном переопределении trend недоступен
	}

	rows, err := analytics.GetCurrentStandings(ctx, pool, &info)
	if err != nil {
		return fmt.Errorf("GetCurrentStandings: %w", err)
	}

	stats := analytics.GetKeyStats(rows)
	insight := analytics.BuildInsight(rows)

	vm := viewmodel.BuildTableViewModel(
		info,
		rows,
		nil,
		nil,
		nil,
		insight,
		stats,
	)

	outputPath := taskData.GetOutput()
	if outputPath == "" {
		outputPath = "standings.png"
	}

	templatePath := filepath.Join("template", "football_table_template.html")
	log.Info("rendering standings PNG", zap.String("output", outputPath))

	if err = renderer.RenderStandingsPNG(vm, templatePath, outputPath); err != nil {
		return fmt.Errorf("RenderStandingsPNG: %w", err)
	}
	log.Info("standings PNG rendered", zap.String("file", outputPath))

	if !taskData.GetPublish() {
		return nil
	}

	caption := fmt.Sprintf("📊 <b>%s — Турнирная таблица</b>\nСезон: %s", info.LeagueName, info.SeasonName)
	return publishToTelegram(ctx, taskData, PubTypeTable, info.LeagueID, info.SeasonID, info.CurrRoundID, 0, info.LeagueName+" — Турнирная таблица", caption, outputPath, log)
}


// dispatchRoundResults формирует и публикует результаты завершённого тура.
// dispatchRoundResults формирует и публикует интерактивные результаты завершённого тура.
func dispatchRoundResults(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	leagueID := taskData.GetLeagueId()
	if leagueID == 0 {
		leagueID = 8
	}

	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	info, err := analytics.GetLeagueSeasonInfo(ctx, pool, leagueID)
	if err != nil {
		return fmt.Errorf("GetLeagueSeasonInfo: %w", err)
	}
	info.Locale = taskData.GetLocale()

	// Ручное переопределение тура через CLI-флаг
	if taskData.GetRoundId() > 0 {
		info.CurrRoundID = taskData.GetRoundId()
	}

	log.Info("fetching round results data",
		zap.Int("leagueId", leagueID),
		zap.Int("seasonId", info.SeasonID),
		zap.Int("roundId", info.CurrRoundID),
	)

	matches, err := analytics.GetRoundResults(ctx, pool, &info)
	if err != nil {
		return fmt.Errorf("GetRoundResults: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no matches found for league %d, season %d, round %d", leagueID, info.SeasonID, info.CurrRoundID)
	}

	roundID := matches[0].RoundID
	roundName := matches[0].RoundName
	summary := analytics.ComputeRoundSummary(matches)

	// Получаем актуальную турнирную таблицу тура из БД
	standings, sErr := analytics.GetCurrentStandings(ctx, pool, &info)
	if sErr != nil {
		log.Warn("GetCurrentStandings failed in roundResults", zap.Error(sErr))
	}

	// Рендерим компактный постер «Топ-5 лидеров + Зона вылета» (Concept 1: TV Leaderboard)
	top5Path := fmt.Sprintf("top5_standings_%d.png", info.LeagueID)
	if len(standings) > 0 {
		insight := analytics.BuildInsight(standings)
		top5VM := viewmodel.BuildTop5TableViewModel(info, roundName, standings, insight)
		top5Tmpl := filepath.Join("template", "top5_standings_template.html")
		if rErr := renderer.RenderTop5StandingsPNG(top5VM, top5Tmpl, top5Path); rErr != nil {
			log.Warn("failed to render top5_standings.png", zap.Error(rErr))
		} else {
			_ = copyFile(top5Path, "top5_standings.png")
			log.Info("top5 standings poster rendered successfully", zap.String("path", top5Path))
		}
	}

	// 1. Экспорт данных в JSON для Telegram Mini App
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

	payloadJSON, err := json.MarshalIndent(payload, "", "  ")
	if err == nil {
		// Сохраняем в локальную директорию веб-просмотрщика
		_ = os.MkdirAll("web/viewer/data", 0755)
		_ = os.WriteFile("web/viewer/round_data.json", payloadJSON, 0644)
		_ = os.WriteFile(fmt.Sprintf("web/viewer/data/%d_%d_%d.json", info.LeagueID, info.SeasonID, roundID), payloadJSON, 0644)

		// Сохраняем в публичную директорию WebApp (если настроена)
		if config.Config.WebApp != nil && config.Config.WebApp.MediaDir != "" {
			publicDir := filepath.Dir(config.Config.WebApp.MediaDir)
			viewerDir := filepath.Join(publicDir, "viewer")
			_ = os.MkdirAll(viewerDir, 0755)

			mediaRoundDir := filepath.Join(config.Config.WebApp.MediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID), strconv.Itoa(roundID))
			_ = os.MkdirAll(mediaRoundDir, 0755)
			_ = os.WriteFile(filepath.Join(mediaRoundDir, "round_data.json"), payloadJSON, 0644)

			// Дублируем свежий round_data.json во все места веб-сервера
			_ = os.WriteFile(filepath.Join(viewerDir, "round_data.json"), payloadJSON, 0644)
			_ = os.WriteFile(filepath.Join(publicDir, "round_data.json"), payloadJSON, 0644)

			viewerDataDir := filepath.Join(viewerDir, "data")
			_ = os.MkdirAll(viewerDataDir, 0755)
			_ = os.WriteFile(filepath.Join(viewerDataDir, fmt.Sprintf("%d_%d_%d.json", info.LeagueID, info.SeasonID, roundID)), payloadJSON, 0644)

			syncViewerFiles(publicDir)

			// Обновляем standings.png свежими данными тура
			if len(standings) > 0 {
				stats := analytics.GetKeyStats(standings)
				insight := analytics.BuildInsight(standings)
				tableVM := viewmodel.BuildTableViewModel(info, standings, nil, nil, nil, insight, stats)
				tableTmpl := filepath.Join("template", "football_table_template.html")
				_ = renderer.RenderStandingsPNG(tableVM, tableTmpl, filepath.Join(mediaRoundDir, "standings.png"))
				_ = copyFile(filepath.Join(mediaRoundDir, "standings.png"), "standings.png")
			}

			// Проверяем / генерируем расписание следующего тура
			if info.NextRoundID > 0 {
				nextRoundDir := filepath.Join(config.Config.WebApp.MediaDir, strconv.Itoa(info.LeagueID), strconv.Itoa(info.SeasonID), strconv.Itoa(info.NextRoundID))
				_ = os.MkdirAll(nextRoundDir, 0755)
				nextFixturesPath := filepath.Join(nextRoundDir, "round_fixtures.png")
				if _, statErr := os.Stat(nextFixturesPath); os.IsNotExist(statErr) {
					nextMatches, nextErr := analytics.GetNextRoundFixtures(ctx, pool, &info)
					if nextErr == nil && len(nextMatches) > 0 {
						fVM := viewmodel.BuildRoundFixturesViewModel(info, nextMatches, "")
						fTmpl := filepath.Join("template", "round_fixtures_template.html")
						_ = renderer.RenderRoundFixturesPNG(fVM, fTmpl, nextFixturesPath)
					}
				}
			}
		}
	}

	// 2. Генерация текста поста с гиперссылками на счёт каждого матча
	postCfg := copywriter.RoundResultsPostConfig{
		LeagueName:   info.LeagueName,
		RoundName:    roundName,
		LeagueID:     info.LeagueID,
		SeasonID:     info.SeasonID,
		RoundID:      roundID,
		BotUsername:  "",
		BotShortName: "",
	}
	if config.Config.WebApp != nil {
		postCfg.BotUsername = config.Config.WebApp.BotUsername
		postCfg.BotShortName = config.Config.WebApp.BotShortName
		postCfg.BaseURL = config.Config.WebApp.BaseURL
	}

	postText := copywriter.GenerateRoundResultsHTMLPost(postCfg, matches, summary)

	// 3. Формирование инлайн-кнопок Mini App (для каналов используются URL deep links)
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

	resultsURL := fmt.Sprintf("https://t.me/%s/%s?startapp=%d_matches", botUser, shortName, info.LeagueID)
	tableURL := fmt.Sprintf("https://t.me/%s/%s?startapp=%d_table", botUser, shortName, info.LeagueID)

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "📱 Все матчи тура в Mini App", URL: resultsURL},
				{Text: "📊 Турнирная таблица", URL: tableURL},
			},
		},
	}

	fmt.Printf("\n=== ПОСТЕР ТОП-5 ЛИДЕРОВ ===\nФайл: %s\n", top5Path)
	fmt.Printf("\n=== СГЕНЕРИРОВАННЫЙ ПОСТ РЕЗУЛЬТАТОВ ТУРА ===\n%s\n===========================================\n\n", postText)

	if !taskData.GetPublish() {
		fmt.Printf("Публикация пропущена (-publish=false). Постер лидеров сохранен в %s, данные тура в web/viewer/round_data.json\n", top5Path)
		return nil
	}

	// 4. Публикация в Telegram
	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return fmt.Errorf("telegram configuration missing (BotToken or ChannelID is empty)")
	}

	channelID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
	storage := getStorage(ctx)

	// Проверка идемпотентности: защита от случайных дубликатов публикаций
	if storage != nil && !taskData.GetForce() {
		already, err := storage.IsAlreadyPublished(ctx, channelID, PubTypeRoundResults, info.LeagueID, roundID)
		if err == nil && already {
			log.Info("round results already published, skipping duplicate publication (use -force to override)",
				zap.String("type", string(PubTypeRoundResults)),
				zap.Int("league_id", info.LeagueID),
				zap.Int("round_id", roundID),
			)
			fmt.Printf("ℹ️ Результаты тура (лига %d, тур %d) уже были опубликованы в канале %s. Для повторной отправки укажите флаг -force\n", info.LeagueID, roundID, tg.ChannelID)
			return nil
		}
	}

	pubCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	log.Info("publishing round results to telegram",
		zap.String("channel", tg.ChannelID),
		zap.String("league", info.LeagueName),
		zap.String("round", roundName),
	)

	// Этап 1: Постер «Топ-5 лидеров + Зона вылета»
	var photoMsgID int64
	if _, statErr := os.Stat(top5Path); statErr == nil {
		photoCap := fmt.Sprintf("🏆 <b>%s • %s — ИТОГИ ТУРА И ТОП-5 ЛИДЕРОВ</b>\n⬇️ <i>Результаты матчей и показатели xG — в интерактивной таблице ниже.</i>", info.LeagueName, roundName)
		fmt.Printf("📤 Отправка постера Топ-5 лидеров (%s)...\n", top5Path)
		pID, pErr := SendPhotoWithResult(pubCtx, tg.BotToken, tg.ChannelID, photoCap, top5Path)
		if pErr != nil {
			log.Warn("failed to send top5 photo, continuing to table", zap.Error(pErr))
		} else {
			photoMsgID = pID
			fmt.Printf("✅ Постер Топ-5 лидеров опубликован в Telegram (ID=%d)\n", photoMsgID)
			if storage != nil {
				_, _ = storage.Create(ctx, &Publication{
					ChannelID:       channelID,
					MessageID:       &photoMsgID,
					PublicationType: PubTypeRoundResults,
					LeagueID:        &info.LeagueID,
					SeasonID:        &info.SeasonID,
					RoundID:         &roundID,
					Title:           fmt.Sprintf("%s — %s Постер лидеров", info.LeagueName, roundName),
					TextContent:     photoCap,
					MediaType:       MediaTypePhoto,
					Status:          StatusPublished,
				})
			}
			time.Sleep(1500 * time.Millisecond)
		}
	}

	// Этап 2: Нативная интерактивная таблица результатов с xG
	richPayload := buildRoundResultsRichTable(info.LeagueName, roundName, matches)
	fmt.Printf("📤 Отправка нативной интерактивной таблицы результатов с кнопками Mini App...\n")
	msgID, pubErr := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richPayload, keyboard)
	if pubErr != nil {
		log.Warn("SendRichMessage failed, fallback to text message with keyboard", zap.Error(pubErr))
		linkPreviewOpt := &LinkPreviewOptions{IsDisabled: true}
		msgID, pubErr = SendTextMessageWithKeyboardAndPreview(pubCtx, tg.BotToken, tg.ChannelID, postText, keyboard, linkPreviewOpt)
		if pubErr != nil {
			return fmt.Errorf("failed to send round results message: %w", pubErr)
		}
	}

	fmt.Printf("✅ Интерактивные результаты тура опубликованы в Telegram (%s)! Message ID: %d\n", tg.ChannelID, msgID)

	// Тихая ротация закрепа (Silent Pin) для флагманской лиги (АПЛ, 8)
	if info.LeagueID == 8 || taskData.GetLeagueId() == 0 {
		if pinErr := PinChatMessage(pubCtx, tg.BotToken, tg.ChannelID, msgID, true); pinErr != nil {
			log.Warn("failed to pin round results message", zap.Error(pinErr))
		} else {
			fmt.Printf("📌 Сообщение с результатами тура тихо закреплено в канале (Silent Pin)\n")
		}
	}

	// Запись в базу данных
	if storage != nil {
		pub := &Publication{
			ChannelID:       channelID,
			MessageID:       &msgID,
			PublicationType: PubTypeRoundResults,
			LeagueID:        &info.LeagueID,
			SeasonID:        &info.SeasonID,
			RoundID:         &roundID,
			Title:           fmt.Sprintf("%s — %s Результаты", info.LeagueName, roundName),
			TextContent:     postText,
			MediaType:       "rich_message",
			Status:          StatusPublished,
		}
		if _, saveErr := storage.Create(ctx, pub); saveErr != nil {
			log.Warn("failed to record publication to database", zap.Error(saveErr))
		}
	}

	return nil
}


// buildRoundResultsRichTable формирует нативную интерактивную таблицу сыгранных матчей тура с xG.
func buildRoundResultsRichTable(leagueName, roundName string, matches []analytics.RoundResultMatch) RichMessagePayload {
	var blocks []RichBlock

	title := fmt.Sprintf("⚽ РЕЗУЛЬТАТЫ МАТЧЕЙ: %s", strings.ToUpper(leagueName))
	if roundName != "" {
		title = fmt.Sprintf("⚽ РЕЗУЛЬТАТЫ МАТЧЕЙ: %s (%s)", strings.ToUpper(leagueName), roundName)
	}

	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 3,
		Text: title,
	})

	var cells [][]RichBlockTableCell
	cells = append(cells, []RichBlockTableCell{
		{Text: "Хозяева", IsHeader: true, Align: "right"},
		{Text: "Счёт", IsHeader: true, Align: "center"},
		{Text: "Гости", IsHeader: true, Align: "left"},
		{Text: "xG", IsHeader: true, Align: "center"},
	})

	for _, m := range matches {
		scoreStr := m.Scores.FullScoreString()
		if scoreStr == "-:-" {
			scoreStr = "—"
		}

		xgStr := m.XGString()
		if xgStr == "" {
			xgStr = "—"
		}

		cells = append(cells, []RichBlockTableCell{
			{Text: m.TeamHomeName, Align: "right"},
			{Text: scoreStr, Align: "center"},
			{Text: m.TeamAwayName, Align: "left"},
			{Text: xgStr, Align: "center"},
		})
	}

	blocks = append(blocks, RichBlock{
		Type:       "table",
		IsBordered: true,
		IsStriped:  true,
		IsCompact:  true,
		Cells:      cells,
	})

	return RichMessagePayload{Blocks: blocks}
}
