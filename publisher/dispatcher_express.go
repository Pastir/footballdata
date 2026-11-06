package publisher

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/copywriter"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/prediction"
	"github.com/Pastir/footballdata/renderer"
	"github.com/Pastir/footballdata/viewmodel"
	"go.uber.org/zap"
)

// dispatchExpress формирует и публикует умный экспресс тура на основе xG-моделей и котировок.
func dispatchExpress(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	leagueID := taskData.GetLeagueId()
	isMultiLeague := (leagueID == 0)

	var candidates []analytics.ExpressCandidateMatch
	var info analytics.LeagueSeasonInfo
	var roundID int
	var roundName string

	if isMultiLeague {
		info = analytics.LeagueSeasonInfo{
			LeagueID:   0,
			LeagueName: "Топ-чемпионаты Европы",
			SeasonName: "2024/2025",
		}
		roundName = "Уикенд"
		var fetchErr error
		candidates, fetchErr = analytics.FetchAllLeaguesCandidateMatches(ctx, pool)
		if fetchErr != nil {
			return fmt.Errorf("FetchAllLeaguesCandidateMatches: %w", fetchErr)
		}
	} else {
		var lErr error
		info, lErr = analytics.GetLeagueSeasonInfo(ctx, pool, leagueID)
		if lErr != nil {
			return fmt.Errorf("GetLeagueSeasonInfo: %w", lErr)
		}

		roundID = taskData.GetRoundId()
		if roundID == 0 {
			uRoundID, uRoundName, uErr := analytics.GetUpcomingRound(ctx, pool, leagueID, time.Now().UTC())
			if uErr == nil && uRoundID > 0 {
				roundID = uRoundID
				roundName = uRoundName
			} else if info.NextRoundID > 0 {
				roundID = info.NextRoundID
				roundName = info.NextRoundName
			} else {
				roundID = info.CurrRoundID
				roundName = info.CurrRoundName
			}
		}

		var cErr error
		candidates, cErr = analytics.FetchRoundCandidateMatches(ctx, pool, leagueID, roundID)
		if cErr != nil || len(candidates) < 3 {
			// Fallback: пробуем соседний тур
			fallbackRoundID := info.NextRoundID
			if fallbackRoundID == roundID || fallbackRoundID == 0 {
				fallbackRoundID = info.CurrRoundID
			}
			if fallbackRoundID > 0 && fallbackRoundID != roundID {
				if fbMatches, fbErr := analytics.FetchRoundCandidateMatches(ctx, pool, leagueID, fallbackRoundID); fbErr == nil && len(fbMatches) >= 3 {
					candidates = fbMatches
					roundID = fallbackRoundID
				}
			}
		}
	}

	log.Info("building smart express",
		zap.Bool("isMultiLeague", isMultiLeague),
		zap.Int("leagueId", leagueID),
		zap.Int("roundId", roundID),
		zap.Int("candidatesCount", len(candidates)),
	)

	if len(candidates) < 3 {
		return fmt.Errorf("insufficient candidate matches with odds for league %d, round %d (found %d, need >= 3)", leagueID, roundID, len(candidates))
	}

	plan, err := analytics.BuildSmartExpress(candidates)
	if err != nil {
		return fmt.Errorf("build smart express: %w", err)
	}

	// Настройки партнёра / букмекера
	bkName := "FONBET"
	promo := "FDL"
	bonus := "Фрибет до 15 000 ₽"
	bkURL := "https://fon.bet"
	if config.Config.Bookmaker != nil {
		if config.Config.Bookmaker.Name != "" {
			bkName = config.Config.Bookmaker.Name
		}
		if config.Config.Bookmaker.Promo != "" {
			promo = config.Config.Bookmaker.Promo
		}
		if config.Config.Bookmaker.Bonus != "" {
			bonus = config.Config.Bookmaker.Bonus
		}
		if config.Config.Bookmaker.URL != "" {
			bkURL = config.Config.Bookmaker.URL
		}
	}

	// Настройки генератора постов
	postCfg := copywriter.ExpressPostConfig{
		LeagueName:    info.LeagueName,
		RoundName:     roundName,
		BookmakerName: bkName,
		PromoCode:     promo,
		BonusText:     bonus,
		BookmakerURL:  bkURL,
	}

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
	fixturesURL := fmt.Sprintf("https://t.me/%s/%s?startapp=all_matches", botUser, shortName)
	if !isMultiLeague {
		fixturesURL = fmt.Sprintf("https://t.me/%s/%s?startapp=%d_matches", botUser, shortName, leagueID)
	}
	postCfg.MiniAppURL = fixturesURL

	style1Text := copywriter.GenerateExpressPostStyle1(postCfg, plan)
	style2Text := copywriter.GenerateExpressPostStyle2(postCfg, plan)
	style3Caption := copywriter.GenerateExpressPostStyle3Caption(postCfg, plan)

	// В нативном стиле Telegram инлайн-клавиатура отключена (nil),
	// чтобы отображалась родная плашка комментариев Telegram с динамическим счетчиком и изолированным тредом.
	var keyboard *InlineKeyboardMarkup = nil

	style := strings.ToLower(strings.TrimSpace(taskData.GetStyle()))

	richUserPayload := buildExpressRichTableUserVariant(postCfg, plan)
	richMonolithPayload := buildExpressRichTableVariantMonolith(postCfg, plan)
	richSubrowPayload := buildExpressRichTableVariantSubrow(postCfg, plan)
	richMobilePayload := buildExpressRichTableVariantMobileCards(postCfg, plan)

	fmt.Printf("\n=== [1 ТЕСТ] СТИЛЬ 1: THE ATHLETIC / EDITORIAL ===\n%s\n\n", style1Text)
	fmt.Printf("=== [2 ТЕСТ] СТИЛЬ 2: BLOOMBERG TERMINAL ===\n%s\n\n", style2Text)
	fmt.Printf("=== [3 ТЕСТ] СТИЛЬ 3: VISUAL SLIP POSTER CAPTION ===\n%s\n", style3Caption)
	fmt.Printf("=== [НАТИВНЫЕ ТАБЛИЦЫ TELEGRAM (Bot API 10.1+)] ===\n")
	fmt.Printf("  • [ВАРИАНТ ПОЛЬЗОВАТЕЛЯ] Match Cards Tables (%d блоков)\n", len(richUserPayload.Blocks))
	fmt.Printf("  • [1 ТЕСТ • СВОЙ] Монолитный стек-купон (%d блоков)\n", len(richMonolithPayload.Blocks))
	fmt.Printf("  • [2 ТЕСТ • СВОЙ] Биржевая матрица с подстрочником (%d блоков)\n", len(richSubrowPayload.Blocks))
	fmt.Printf("  • [3 ТЕСТ • СВОЙ] Мобильный аналитический паспорт (%d блоков)\n", len(richMobilePayload.Blocks))
	fmt.Printf("===================================================\n\n")

	outputPath := taskData.GetOutput()
	if outputPath == "" {
		outputPath = "express_slip.png"
	}

	renderSlipPNG := func() error {
		promoLine := fmt.Sprintf("%s • ФРИБЕТ ДО 15 000 ₽ ПО КОДУ %s", bkName, promo)
		vm := viewmodel.BuildExpressSlipViewModel(info.LeagueName, roundName, plan, promoLine)
		tmplPath := filepath.Join("template", "express_slip_template.html")
		return renderer.RenderExpressSlipPNG(vm, tmplPath, outputPath)
	}

	if !taskData.GetPublish() {
		fmt.Printf("🎨 Рендеринг визуального постера-купона в режиме проверки: %s...\n", outputPath)
		if rErr := renderSlipPNG(); rErr != nil {
			fmt.Printf("⚠️ Ошибка рендеринга постера: %v\n", rErr)
		} else {
			fmt.Printf("✅ Постер успешно сгенерирован: %s\n", outputPath)
		}
		fmt.Println("ℹ️ Публикация пропущена (-publish=false). Экспресс сформирован в режиме симуляции.")
		return nil
	}

	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return fmt.Errorf("telegram configuration missing")
	}

	channelID, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
	storage := getStorage(ctx)

	// Проверка идемпотентности
	if storage != nil && !taskData.GetForce() {
		already, err := storage.IsAlreadyPublished(ctx, channelID, PubTypeExpress, leagueID, roundID)
		if err == nil && already {
			log.Info("express already published, skipping duplicate",
				zap.Int("league_id", leagueID),
				zap.Int("round_id", roundID),
			)
			fmt.Printf("ℹ️ Экспресс тура (лига %d, тур %d) уже был опубликован. Для повторной отправки укажите -force\n", leagueID, roundID)
			return nil
		}
	}

	pubCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var msgID int64
	postText := style1Text

	switch style {
	case "user":
		fmt.Printf("📤 Отправка [ВАРИАНТ ПОЛЬЗОВАТЕЛЯ] (Нативная таблица: Карточки матчей) в Telegram %s...\n", tg.ChannelID)
		id, err := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richUserPayload, keyboard)
		if err != nil {
			return fmt.Errorf("send user variant: %w", err)
		}
		msgID = id
		postText = "[ВАРИАНТ ПОЛЬЗОВАТЕЛЯ] Нативная таблица: Карточки матчей"
		fmt.Printf("✅ [ВАРИАНТ ПОЛЬЗОВАТЕЛЯ] успешно опубликован в Telegram (Message ID: %d)!\n", id)

	case "rich1", "table1", "monolith":
		fmt.Printf("📤 Отправка [1 ТЕСТ • СВОЙ] (Нативная таблица: Монолитный купон) в Telegram-канал %s...\n", tg.ChannelID)
		id, err := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richMonolithPayload, keyboard)
		if err != nil {
			return fmt.Errorf("send rich 1: %w", err)
		}
		msgID = id
		postText = "[1 ТЕСТ • СВОЙ] Нативная таблица: Монолитный купон"
		fmt.Printf("✅ [1 ТЕСТ • СВОЙ] успешно опубликован в Telegram (Message ID: %d)!\n", id)

	case "rich2", "table2", "subrow":
		fmt.Printf("📤 Отправка [2 ТЕСТ • СВОЙ] (Нативная таблица: Биржевая матрица) в Telegram-канал %s...\n", tg.ChannelID)
		id, err := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richSubrowPayload, keyboard)
		if err != nil {
			return fmt.Errorf("send rich 2: %w", err)
		}
		msgID = id
		postText = "[2 ТЕСТ • СВОЙ] Нативная таблица: Биржевая матрица"
		fmt.Printf("✅ [2 ТЕСТ • СВОЙ] успешно опубликован в Telegram (Message ID: %d)!\n", id)

	case "rich3", "table3", "mobile":
		fmt.Printf("📤 Отправка [3 ТЕСТ • СВОЙ] (Нативная таблица: Мобильный паспорт) в Telegram-канал %s...\n", tg.ChannelID)
		id, err := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richMobilePayload, keyboard)
		if err != nil {
			return fmt.Errorf("send rich 3: %w", err)
		}
		msgID = id
		postText = "[3 ТЕСТ • СВОЙ] Нативная таблица: Мобильный паспорт"
		fmt.Printf("✅ [3 ТЕСТ • СВОЙ] успешно опубликован в Telegram (Message ID: %d)!\n", id)

	case "rich", "richall", "tables", "all4":
		fmt.Printf("🚀 Режим тестирования нативных таблиц: последовательная публикация 1 варианта пользователя + 3 вариантов своих в %s...\n", tg.ChannelID)

		// 1. Отправляем [ВАРИАНТ ПОЛЬЗОВАТЕЛЯ]
		fmt.Println("📤 Отправка [ВАРИАНТ ПОЛЬЗОВАТЕЛЯ] (Нативная таблица: Карточки матчей)...")
		idUser, err := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richUserPayload, keyboard)
		if err != nil {
			log.Warn("failed to send user variant", zap.Error(err))
		} else {
			msgID = idUser
			fmt.Printf("✅ [ВАРИАНТ ПОЛЬЗОВАТЕЛЯ] опубликован (Message ID: %d)\n", idUser)
		}
		time.Sleep(2 * time.Second)

		// 2. Отправляем [1 ТЕСТ • СВОЙ] (Монолитный стек-купон)
		fmt.Println("📤 Отправка [1 ТЕСТ • СВОЙ] (Нативная таблица: Монолитный купон)...")
		id1, err := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richMonolithPayload, keyboard)
		if err != nil {
			log.Warn("failed to send rich 1", zap.Error(err))
		} else {
			fmt.Printf("✅ [1 ТЕСТ • СВОЙ] опубликован (Message ID: %d)\n", id1)
		}
		time.Sleep(2 * time.Second)

		// 3. Отправляем [2 ТЕСТ • СВОЙ] (Биржевая матрица)
		fmt.Println("📤 Отправка [2 ТЕСТ • СВОЙ] (Нативная таблица: Биржевая матрица)...")
		id2, err := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richSubrowPayload, keyboard)
		if err != nil {
			log.Warn("failed to send rich 2", zap.Error(err))
		} else {
			fmt.Printf("✅ [2 ТЕСТ • СВОЙ] опубликован (Message ID: %d)\n", id2)
		}
		time.Sleep(2 * time.Second)

		// 4. Отправляем [3 ТЕСТ • СВОЙ] (Мобильный паспорт)
		fmt.Println("📤 Отправка [3 ТЕСТ • СВОЙ] (Нативная таблица: Мобильный паспорт)...")
		id3, err := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richMobilePayload, keyboard)
		if err != nil {
			log.Warn("failed to send rich 3", zap.Error(err))
		} else {
			fmt.Printf("✅ [3 ТЕСТ • СВОЙ] опубликован (Message ID: %d)\n", id3)
		}
		postText = "Нативные таблицы Telegram: [ВАРИАНТ ПОЛЬЗОВАТЕЛЯ] + 3 авторских варианта"

	case "1":
		fmt.Printf("📤 Отправка [1 ТЕСТ] (Editorial Style) в Telegram-канал %s...\n", tg.ChannelID)
		id, err := SendTextMessageWithKeyboardAndPreview(pubCtx, tg.BotToken, tg.ChannelID, style1Text, keyboard, &LinkPreviewOptions{IsDisabled: true})
		if err != nil {
			return fmt.Errorf("send style 1: %w", err)
		}
		msgID = id
		postText = style1Text
		fmt.Printf("✅ [1 ТЕСТ] успешно опубликован в Telegram (Message ID: %d)!\n", id)

	case "2":
		fmt.Printf("📤 Отправка [2 ТЕСТ] (Terminal Style) в Telegram-канал %s...\n", tg.ChannelID)
		id, err := SendTextMessageWithKeyboardAndPreview(pubCtx, tg.BotToken, tg.ChannelID, style2Text, keyboard, &LinkPreviewOptions{IsDisabled: true})
		if err != nil {
			return fmt.Errorf("send style 2: %w", err)
		}
		msgID = id
		postText = style2Text
		fmt.Printf("✅ [2 ТЕСТ] успешно опубликован в Telegram (Message ID: %d)!\n", id)

	case "3":
		fmt.Printf("🎨 Рендеринг визуального постера-купона %s...\n", outputPath)
		if rErr := renderSlipPNG(); rErr != nil {
			return fmt.Errorf("render slip png: %w", rErr)
		}
		fmt.Printf("📤 Отправка [3 ТЕСТ] (Visual Slip Poster) в Telegram-канал %s...\n", tg.ChannelID)
		id, err := SendPhotoWithKeyboard(pubCtx, tg.BotToken, tg.ChannelID, style3Caption, outputPath, keyboard)
		if err != nil {
			return fmt.Errorf("send style 3: %w", err)
		}
		msgID = id
		postText = style3Caption
		fmt.Printf("✅ [3 ТЕСТ] успешно опубликован в Telegram (Message ID: %d)!\n", id)

	default: // "all", "test", "tests" или пустой стиль — публикуем все 3 теста последовательно!
		fmt.Printf("🚀 Режим тестирования подачи: последовательная публикация всех 3 вариантов стиля в %s...\n", tg.ChannelID)

		// 1. Отправляем [1 ТЕСТ]
		fmt.Println("📤 Отправка [1 ТЕСТ] (The Athletic / Editorial Longread)...")
		id1, err := SendTextMessageWithKeyboardAndPreview(pubCtx, tg.BotToken, tg.ChannelID, style1Text, keyboard, &LinkPreviewOptions{IsDisabled: true})
		if err != nil {
			log.Warn("failed to send style 1", zap.Error(err))
		} else {
			msgID = id1
			fmt.Printf("✅ [1 ТЕСТ] опубликован (Message ID: %d)\n", id1)
		}
		time.Sleep(2 * time.Second)

		// 2. Отправляем [2 ТЕСТ]
		fmt.Println("📤 Отправка [2 ТЕСТ] (Bloomberg Terminal / ASCII Table)...")
		id2, err := SendTextMessageWithKeyboardAndPreview(pubCtx, tg.BotToken, tg.ChannelID, style2Text, keyboard, &LinkPreviewOptions{IsDisabled: true})
		if err != nil {
			log.Warn("failed to send style 2", zap.Error(err))
		} else {
			fmt.Printf("✅ [2 ТЕСТ] опубликован (Message ID: %d)\n", id2)
		}
		time.Sleep(2 * time.Second)

		// 3. Отправляем [3 ТЕСТ]
		fmt.Printf("🎨 Рендеринг визуального постера-купона %s...\n", outputPath)
		if rErr := renderSlipPNG(); rErr != nil {
			log.Warn("render slip png failed", zap.Error(rErr))
		} else {
			fmt.Println("📤 Отправка [3 ТЕСТ] (Visual Slip Poster)...")
			id3, err := SendPhotoWithKeyboard(pubCtx, tg.BotToken, tg.ChannelID, style3Caption, outputPath, keyboard)
			if err != nil {
				log.Warn("failed to send style 3", zap.Error(err))
			} else {
				fmt.Printf("✅ [3 ТЕСТ] опубликован (Message ID: %d)\n", id3)
			}
		}
	}

	// Инициализируем и сохраняем прогноз в prediction_storage
	predStorage := prediction.NewPGStorage(pool)
	_ = predStorage.EnsureSchema(ctx)

	var predItems []prediction.PredictionItem
	for _, sel := range plan.Selections {
		predItems = append(predItems, prediction.PredictionItem{
			FixtureID:        sel.Match.FixtureID,
			LeagueID:         leagueID,
			HomeTeamID:       sel.Match.HomeTeamID,
			AwayTeamID:       sel.Match.AwayTeamID,
			MatchStartingAt:  sel.Match.StartingAt,
			MarketName:       sel.MarketName,
			MarketKey:        sel.MarketKey,
			Selection:        sel.Selection,
			Odds:             sel.Odds,
			BookmakerName:    bkName,
			SettlementStatus: prediction.StatusPending,
		})
	}

	pubID := int64(0)
	if storage != nil {
		pub := &Publication{
			ChannelID:       channelID,
			MessageID:       &msgID,
			PublicationType: PubTypeExpress,
			LeagueID:        &leagueID,
			SeasonID:        &info.SeasonID,
			RoundID:         &roundID,
			Title:           fmt.Sprintf("%s — Экспресс тура (кэф %.2f)", info.LeagueName, plan.TotalOdds),
			TextContent:     postText,
			MediaType:       MediaTypeText,
			Status:          StatusPublished,
		}
		if pID, sErr := storage.Create(ctx, pub); sErr == nil {
			pubID = pID
		}
	}

	pred := &prediction.Prediction{
		BetType:          prediction.BetTypeAccumulator,
		Title:            fmt.Sprintf("%s — Экспресс тура (кэф %.2f)", info.LeagueName, plan.TotalOdds),
		AnalysisText:     postText,
		TotalOdds:        plan.TotalOdds,
		Confidence:       plan.Confidence,
		ValueEdge:        plan.ValueEdge,
		RecommendedStake: plan.RecommendedStake,
		Status:           prediction.StatusPending,
		PublicationID:    &pubID,
		Items:            predItems,
	}

	if predID, pErr := predStorage.CreatePrediction(ctx, pred); pErr == nil {
		fmt.Printf("📝 Карточка экспресса зарегистрирована в базе прогнозов (ID=%d, статус: pending)\n", predID)
	} else {
		log.Warn("failed to record prediction to database", zap.Error(pErr))
	}

	return nil
}



// buildExpressRichTableUserVariant формирует вариант пользователя:
// Отдельная нативная 3-колоночная таблица-карточка для каждого матча
// (Хозяева : Гости / Время Исход Кэф / Команда: xG-обоснование) + сводный подвал купона.
func buildExpressRichTableUserVariant(cfg copywriter.ExpressPostConfig, plan *analytics.SmartExpressPlan) RichMessagePayload {
	var blocks []RichBlock

	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 3,
		Text: fmt.Sprintf("🎯 [ВАРИАНТ ПОЛЬЗОВАТЕЛЯ] ЭКСПРЕСС ТУРА: КОЭФФИЦИЕНТ %.2f", plan.TotalOdds),
	})

	leagueRound := fmt.Sprintf("🏆 %s · %s", strings.ToUpper(cfg.LeagueName), strings.ToUpper(cfg.RoundName))
	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 4,
		Text: leagueRound,
	})

	for _, sel := range plan.Selections {
		var matchCells [][]RichBlockTableCell

		// Строка 1: Хозяева : Гости
		matchCells = append(matchCells, []RichBlockTableCell{
			{Text: sel.Match.HomeDisplayName(), Align: "right", IsHeader: true},
			{Text: ":", Align: "center", IsHeader: true},
			{Text: sel.Match.AwayDisplayName(), Align: "left", IsHeader: true},
		})

		// Строка 2: 24.10 14:00 UTC | Исход | Кэф
		timeBadge := sel.Match.StartingAt.UTC().Format("02.01 15:04 UTC")
		matchCells = append(matchCells, []RichBlockTableCell{
			{Text: timeBadge, Align: "center"},
			{Text: sel.OutcomeShortName(), Align: "center", IsHeader: true},
			{Text: fmt.Sprintf("%.2f", sel.Odds), Align: "center"},
		})

		// Строка 3: Команда: | xG-фактура и обоснование
		matchCells = append(matchCells, []RichBlockTableCell{
			{Text: sel.Match.HomeDisplayName() + ":", Colspan: 1, Align: "left", IsHeader: true},
			{Text: sel.Reason, Colspan: 2, Align: "left"},
		})

		blocks = append(blocks, RichBlock{
			Type:       "table",
			IsBordered: true,
			IsStriped:  true,
			IsCompact:  true,
			Cells:      matchCells,
		})
	}

	// Сводная таблица параметров купона
	var summaryCells [][]RichBlockTableCell
	summaryCells = append(summaryCells, []RichBlockTableCell{
		{Text: "Итоговый коэффициент купона", IsHeader: true, Align: "left"},
		{Text: fmt.Sprintf("%.2f", plan.TotalOdds), IsHeader: true, Align: "center"},
	})
	summaryCells = append(summaryCells, []RichBlockTableCell{
		{Text: "Математический Value Edge", Align: "left"},
		{Text: fmt.Sprintf("+%.1f%%", plan.ValueEdge), Align: "center"},
	})
	summaryCells = append(summaryCells, []RichBlockTableCell{
		{Text: "Рекомендуемый флэт", Align: "left"},
		{Text: fmt.Sprintf("%.1f юнит (1%%)", plan.RecommendedStake), Align: "center"},
	})
	if cfg.BookmakerName != "" {
		summaryCells = append(summaryCells, []RichBlockTableCell{
			{Text: fmt.Sprintf("Партнёрский бонус (%s)", cfg.BookmakerName), Align: "left"},
			{Text: fmt.Sprintf("%s (код %s)", cfg.BonusText, cfg.PromoCode), Align: "center"},
		})
	}

	blocks = append(blocks, RichBlock{
		Type:       "table",
		IsBordered: true,
		IsStriped:  true,
		IsCompact:  true,
		Cells:      summaryCells,
	})

	return RichMessagePayload{Blocks: blocks}
}

// buildExpressRichTableVariantMonolith формирует вариант 1 (Свой):
// Единая монолитная таблица с группировкой матчей разделителями, рынками и сводными строками.
func buildExpressRichTableVariantMonolith(cfg copywriter.ExpressPostConfig, plan *analytics.SmartExpressPlan) RichMessagePayload {
	var blocks []RichBlock

	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 3,
		Text: "🎯 [1 ТЕСТ • СВОЙ] DATA LAB | МОНОЛИТНЫЙ КУПОН",
	})

	leagueRound := fmt.Sprintf("🏆 %s · %s", strings.ToUpper(cfg.LeagueName), strings.ToUpper(cfg.RoundName))
	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 4,
		Text: leagueRound,
	})

	var tableCells [][]RichBlockTableCell

	for i, sel := range plan.Selections {
		timeBadge := sel.Match.StartingAt.UTC().Format("15:04 UTC")
		matchHeader := fmt.Sprintf("⚽ %d. %s — %s (%s)", i+1, sel.Match.HomeDisplayName(), sel.Match.AwayDisplayName(), timeBadge)

		// Секционный заголовок матча на всю ширину
		tableCells = append(tableCells, []RichBlockTableCell{
			{Text: matchHeader, IsHeader: true, Colspan: 3, Align: "center"},
		})

		// Строка 1: Команды
		tableCells = append(tableCells, []RichBlockTableCell{
			{Text: sel.Match.HomeDisplayName(), Align: "right"},
			{Text: ":", Align: "center"},
			{Text: sel.Match.AwayDisplayName(), Align: "left"},
		})

		// Строка 2: Рынок, Выбор, Кэф
		tableCells = append(tableCells, []RichBlockTableCell{
			{Text: sel.MarketName, Align: "center"},
			{Text: sel.OutcomeShortName() + " ✅", Align: "center", IsHeader: true},
			{Text: fmt.Sprintf("%.2f", sel.Odds), Align: "center"},
		})

		// Строка 3: Фактор на всю ширину
		tableCells = append(tableCells, []RichBlockTableCell{
			{Text: "💡 " + sel.Reason, Colspan: 3, Align: "left"},
		})
	}

	// Подвальные строки купона
	tableCells = append(tableCells, []RichBlockTableCell{
		{Text: "ИТОГОВЫЙ КОЭФФИЦИЕНТ", IsHeader: true, Colspan: 2, Align: "right"},
		{Text: fmt.Sprintf("%.2f", plan.TotalOdds), IsHeader: true, Align: "center"},
	})
	tableCells = append(tableCells, []RichBlockTableCell{
		{Text: "МАТЕМАТИЧЕСКИЙ VALUE EDGE", IsHeader: true, Colspan: 2, Align: "right"},
		{Text: fmt.Sprintf("+%.1f%%", plan.ValueEdge), IsHeader: true, Align: "center"},
	})
	tableCells = append(tableCells, []RichBlockTableCell{
		{Text: "РЕКОМЕНДУЕМЫЙ ФЛЭТ", IsHeader: true, Colspan: 2, Align: "right"},
		{Text: fmt.Sprintf("%.1f юнит (1%%)", plan.RecommendedStake), IsHeader: true, Align: "center"},
	})

	blocks = append(blocks, RichBlock{
		Type:       "table",
		IsBordered: true,
		IsStriped:  true,
		IsCompact:  true,
		Cells:      tableCells,
	})

	if cfg.BookmakerName != "" {
		blocks = append(blocks, RichBlock{
			Type: "heading",
			Size: 4,
			Text: "🎁 БОНУС ПАРТНЁРА ТУРА",
		})
		var promoCells [][]RichBlockTableCell
		promoCells = append(promoCells, []RichBlockTableCell{
			{Text: "Букмекер", IsHeader: true, Align: "center"},
			{Text: "Промокод", IsHeader: true, Align: "center"},
			{Text: "Предложение", IsHeader: true, Align: "left"},
		})
		promoCells = append(promoCells, []RichBlockTableCell{
			{Text: cfg.BookmakerName, Align: "center"},
			{Text: cfg.PromoCode, Align: "center"},
			{Text: cfg.BonusText, Align: "left"},
		})
		blocks = append(blocks, RichBlock{
			Type:       "table",
			IsBordered: true,
			IsStriped:  true,
			IsCompact:  true,
			Cells:      promoCells,
		})
	}

	return RichMessagePayload{Blocks: blocks}
}

// buildExpressRichTableVariantSubrow формирует вариант 2 (Свой):
// 4-колоночная биржевая таблица ордеров с подстрочным разбором xG-фактора под каждым событием.
func buildExpressRichTableVariantSubrow(cfg copywriter.ExpressPostConfig, plan *analytics.SmartExpressPlan) RichMessagePayload {
	var blocks []RichBlock

	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 3,
		Text: "📈 [2 ТЕСТ • СВОЙ] QUANT LAB | БИРЖЕВАЯ МАТРИЦА",
	})

	leagueRound := fmt.Sprintf("⚡ %s · %s · Value Edge: +%.1f%%", strings.ToUpper(cfg.LeagueName), strings.ToUpper(cfg.RoundName), plan.ValueEdge)
	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 4,
		Text: leagueRound,
	})

	var tableCells [][]RichBlockTableCell
	tableCells = append(tableCells, []RichBlockTableCell{
		{Text: "Матч (UTC)", IsHeader: true, Align: "left"},
		{Text: "Рынок", IsHeader: true, Align: "center"},
		{Text: "Исход", IsHeader: true, Align: "center"},
		{Text: "Кэф", IsHeader: true, Align: "center"},
	})

	for _, sel := range plan.Selections {
		timeStr := sel.Match.StartingAt.UTC().Format("15:04")
		matchName := fmt.Sprintf("%s %s—%s", timeStr, sel.Match.HomeDisplayName(), sel.Match.AwayDisplayName())

		tableCells = append(tableCells, []RichBlockTableCell{
			{Text: matchName, Align: "left"},
			{Text: sel.MarketName, Align: "center"},
			{Text: sel.OutcomeShortName() + " 🔥", Align: "center", IsHeader: true},
			{Text: fmt.Sprintf("%.2f", sel.Odds), Align: "center"},
		})

		tableCells = append(tableCells, []RichBlockTableCell{
			{Text: "↳ " + sel.Reason, Colspan: 4, Align: "left"},
		})
	}

	// Итоговые строки портфеля
	tableCells = append(tableCells, []RichBlockTableCell{
		{Text: "ИТОГОВЫЙ КОЭФФИЦИЕНТ ПОРТФЕЛЯ", Colspan: 3, IsHeader: true, Align: "right"},
		{Text: fmt.Sprintf("%.2f", plan.TotalOdds), Colspan: 1, IsHeader: true, Align: "center"},
	})
	tableCells = append(tableCells, []RichBlockTableCell{
		{Text: "МАТЕМАТИЧЕСКИЙ VALUE EDGE", Colspan: 3, IsHeader: true, Align: "right"},
		{Text: fmt.Sprintf("+%.1f%%", plan.ValueEdge), Colspan: 1, IsHeader: true, Align: "center"},
	})
	winProb := 100.0 / plan.TotalOdds * (1.0 + plan.ValueEdge/100.0)
	if winProb > 95.0 {
		winProb = 95.0
	}
	tableCells = append(tableCells, []RichBlockTableCell{
		{Text: "РАСЧЁТНАЯ ВЕРОЯТНОСТЬ МОДЕЛИ", Colspan: 3, IsHeader: true, Align: "right"},
		{Text: fmt.Sprintf("%.1f%%", winProb), Colspan: 1, IsHeader: true, Align: "center"},
	})

	blocks = append(blocks, RichBlock{
		Type:       "table",
		IsBordered: true,
		IsStriped:  true,
		IsCompact:  true,
		Cells:      tableCells,
	})

	return RichMessagePayload{Blocks: blocks}
}

// buildExpressRichTableVariantMobileCards формирует аналитический экспресс (Вариант 3):
// Двухколоночный мобильный протокол без эмодзи, с выравниванием по левому краю, xG-фактурой для обеих команд и без фразы "Алгоритмический отбор".
func buildExpressRichTableVariantMobileCards(cfg copywriter.ExpressPostConfig, plan *analytics.SmartExpressPlan) RichMessagePayload {
	var blocks []RichBlock

	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 3,
		Text: fmt.Sprintf("DATA LAB | АНАЛИТИЧЕСКИЙ ЭКСПРЕСС (КЭФ %.2f)", plan.TotalOdds),
	})

	leagueRound := fmt.Sprintf("%s · %s · Общий кэф %.2f", cfg.LeagueName, cfg.RoundName, plan.TotalOdds)
	blocks = append(blocks, RichBlock{
		Type: "heading",
		Size: 4,
		Text: leagueRound,
	})

	for i, sel := range plan.Selections {
		timeBadge := sel.Match.StartingAt.UTC().Format("02.01 15:04 UTC")
		var cardCells [][]RichBlockTableCell

		cardCells = append(cardCells, []RichBlockTableCell{
			{Text: fmt.Sprintf("СОБЫТИЕ %d · %s", i+1, timeBadge), Colspan: 2, IsHeader: true, Align: "left"},
		})
		matchTitle := fmt.Sprintf("%s — %s", sel.Match.HomeDisplayName(), sel.Match.AwayDisplayName())
		if sel.Match.CountryFlag != "" && sel.Match.LeagueName != "" {
			matchTitle = fmt.Sprintf("%s %s · %s", sel.Match.CountryFlag, sel.Match.LeagueName, matchTitle)
		} else if sel.Match.CountryFlag != "" {
			matchTitle = fmt.Sprintf("%s %s", sel.Match.CountryFlag, matchTitle)
		}

		cardCells = append(cardCells, []RichBlockTableCell{
			{Text: "Матч", IsHeader: true, Align: "left"},
			{Text: matchTitle, Align: "left"},
		})
		cardCells = append(cardCells, []RichBlockTableCell{
			{Text: "Прогноз (+EV)", IsHeader: true, Align: "left"},
			{Text: fmt.Sprintf("%s (кэф %.2f)", sel.OutcomeDisplayName(), sel.Odds), Align: "left"},
		})
		cardCells = append(cardCells, []RichBlockTableCell{
			{Text: "xG-фактура", IsHeader: true, Align: "left"},
			{Text: sel.Reason, Align: "left"},
		})

		blocks = append(blocks, RichBlock{
			Type:       "table",
			IsBordered: true,
			IsStriped:  true,
			IsCompact:  true,
			Cells:      cardCells,
		})
	}

	var teleCells [][]RichBlockTableCell
	teleCells = append(teleCells, []RichBlockTableCell{
		{Text: "ИТОГОВЫЙ ЭКСПРЕСС", Colspan: 2, IsHeader: true, Align: "left"},
	})
	teleCells = append(teleCells, []RichBlockTableCell{
		{Text: "Итоговый коэффициент", Align: "left"},
		{Text: fmt.Sprintf("%.2f", plan.TotalOdds), Align: "center", IsHeader: true},
	})
	teleCells = append(teleCells, []RichBlockTableCell{
		{Text: "Математический перевес", Align: "left"},
		{Text: fmt.Sprintf("+%.1f%% (+EV)", plan.ValueEdge), Align: "center"},
	})
	winProb := 100.0 / plan.TotalOdds * (1.0 + plan.ValueEdge/100.0)
	bkProb := 100.0 / plan.TotalOdds
	if winProb > 95.0 {
		winProb = 95.0
	}
	teleCells = append(teleCells, []RichBlockTableCell{
		{Text: "Вероятность модели vs БК", Align: "left"},
		{Text: fmt.Sprintf("%.1f%% vs %.1f%%", winProb, bkProb), Align: "center"},
	})
	teleCells = append(teleCells, []RichBlockTableCell{
		{Text: "Банк-менеджмент", Align: "left"},
		{Text: fmt.Sprintf("%.1f unit (1%% флэт)", plan.RecommendedStake), Align: "center"},
	})
	if cfg.BookmakerName != "" {
		teleCells = append(teleCells, []RichBlockTableCell{
			{Text: "Партнёрский бонус", Align: "left"},
			{Text: fmt.Sprintf("%s: %s (промокод %s)", cfg.BookmakerName, cfg.BonusText, cfg.PromoCode), Align: "center"},
		})
	}
	teleCells = append(teleCells, []RichBlockTableCell{
		{Text: "Интерактивная модель", Align: "left"},
		{Text: "Все матчи тура и xG в Football Data Lab Mini App", Align: "center"},
	})

	blocks = append(blocks, RichBlock{
		Type:       "table",
		IsBordered: true,
		IsStriped:  true,
		IsCompact:  true,
		Cells:      teleCells,
	})

	return RichMessagePayload{Blocks: blocks}
}

