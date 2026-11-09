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
		roundID = taskData.GetRoundId()
		if roundID == 0 {
			year, week := time.Now().UTC().ISOWeek()
			roundID = year*100 + week
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
		fmt.Printf("\n=== ЭКСПРЕСС ТУРА (Нативная таблица: %d блоков) ===\n%s\n\n", len(richUserPayload.Blocks), style1Text)
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
	case "1":
		fmt.Printf("📤 Отправка (Editorial Style) в Telegram-канал %s...\n", tg.ChannelID)
		id, err := SendTextMessageWithKeyboardAndPreview(pubCtx, tg.BotToken, tg.ChannelID, style1Text, keyboard, &LinkPreviewOptions{IsDisabled: true})
		if err != nil {
			return fmt.Errorf("send style 1: %w", err)
		}
		msgID = id
		postText = style1Text
		fmt.Printf("✅ Экспресс (Style 1) опубликован в Telegram (Message ID: %d)!\n", id)

	case "2":
		fmt.Printf("📤 Отправка (Terminal Style) в Telegram-канал %s...\n", tg.ChannelID)
		id, err := SendTextMessageWithKeyboardAndPreview(pubCtx, tg.BotToken, tg.ChannelID, style2Text, keyboard, &LinkPreviewOptions{IsDisabled: true})
		if err != nil {
			return fmt.Errorf("send style 2: %w", err)
		}
		msgID = id
		postText = style2Text
		fmt.Printf("✅ Экспресс (Style 2) опубликован в Telegram (Message ID: %d)!\n", id)

	case "3":
		fmt.Printf("🎨 Рендеринг визуального постера-купона %s...\n", outputPath)
		if rErr := renderSlipPNG(); rErr != nil {
			return fmt.Errorf("render slip png: %w", rErr)
		}
		fmt.Printf("📤 Отправка (Visual Slip Poster) в Telegram-канал %s...\n", tg.ChannelID)
		id, err := SendPhotoWithKeyboard(pubCtx, tg.BotToken, tg.ChannelID, style3Caption, outputPath, keyboard)
		if err != nil {
			return fmt.Errorf("send style 3: %w", err)
		}
		msgID = id
		postText = style3Caption
		fmt.Printf("✅ Экспресс (Style 3) опубликован в Telegram (Message ID: %d)!\n", id)

	default: // "user", "" и др. — боевой нативный табличный формат по умолчанию
		fmt.Printf("📤 Отправка экспресса тура (Нативная таблица: Карточки матчей) в Telegram %s...\n", tg.ChannelID)
		id, err := SendRichMessage(pubCtx, tg.BotToken, tg.ChannelID, richUserPayload, keyboard)
		if err != nil {
			return fmt.Errorf("send express rich table: %w", err)
		}
		msgID = id
		postText = style1Text
		fmt.Printf("✅ Экспресс тура успешно опубликован в Telegram (Message ID: %d)!\n", id)
	}

	// Инициализируем и сохраняем прогноз в prediction_storage
	predStorage := prediction.NewPGStorage(pool)
	_ = predStorage.EnsureSchema(ctx)

	var predItems []prediction.PredictionItem
	for _, sel := range plan.Selections {
		itemLeagueID := sel.Match.LeagueID
		if itemLeagueID == 0 {
			itemLeagueID = leagueID
		}
		predItems = append(predItems, prediction.PredictionItem{
			FixtureID:        sel.Match.FixtureID,
			LeagueID:         itemLeagueID,
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
		Text: fmt.Sprintf("🎯 ЭКСПРЕСС ТУРА: КОЭФФИЦИЕНТ %.2f", plan.TotalOdds),
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

