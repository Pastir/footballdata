package publisher

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/copywriter"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/renderer"
	"github.com/Pastir/footballdata/viewmodel"
	"go.uber.org/zap"
)

func buildMiniAppDeepLink(baseURL string, leagueID, roundID int, fixtureID int64) string {
	if baseURL == "" {
		baseURL = "https://data-lab.live/"
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	q := u.Query()
	if leagueID > 0 {
		q.Set("league", strconv.Itoa(leagueID))
	}
	if roundID > 0 {
		q.Set("round", strconv.Itoa(roundID))
	}
	if fixtureID > 0 {
		q.Set("match", strconv.FormatInt(fixtureID, 10))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// dispatchBanter обнаруживает аномалии/драму тура и генерирует инфографическую карточку 1200x1200px с постом
func dispatchBanter(ctx context.Context, taskData config.TaskData, log *zap.Logger) error {
	pool := db.DBx.PGxPool
	if pool == nil {
		return fmt.Errorf("database connection pool is not initialized")
	}

	leagueID := taskData.GetLeagueId()
	if leagueID == 0 {
		leagueID = 8 // По умолчанию АПЛ
	}
	roundID := taskData.GetRoundId()

	log.Info("scanning round for anomalies and banter events", zap.Int("leagueId", leagueID), zap.Int("roundId", roundID))
	anomalies, err := analytics.FindRoundAnomalies(ctx, pool, leagueID, roundID)
	if err != nil {
		return fmt.Errorf("find round anomalies: %w", err)
	}

	if len(anomalies) == 0 {
		fmt.Printf("ℹ️ В лиге %d (тур %d) не зафиксировано явных аномалий или драмы.\n", leagueID, roundID)
		return nil
	}

	bestAnomaly := anomalies[0]
	fmt.Printf("🎯 Найдена главная аномалия тура: %s (%s, Score: %.1f)\n",
		bestAnomaly.BadgeText, bestAnomaly.AnomalySummary, bestAnomaly.SeverityScore)

	vm := viewmodel.BuildBanterCardViewModel(bestAnomaly)

	templatePath := filepath.Join("template", "banter_card_template.html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		templatePath = filepath.Join("..", "template", "banter_card_template.html")
	}

	outputPath := taskData.GetOutput()
	if outputPath == "" {
		outputPath = fmt.Sprintf("banter_%d_%d.png", bestAnomaly.LeagueID, bestAnomaly.FixtureID)
	}

	webAppURL := "https://data-lab.live/"
	if config.Config.WebApp != nil && config.Config.WebApp.BaseURL != "" {
		webAppURL = config.Config.WebApp.BaseURL
	}

	miniAppURL := buildMiniAppDeepLink(webAppURL, bestAnomaly.LeagueID, bestAnomaly.RoundID, bestAnomaly.FixtureID)
	postHTML := copywriter.GenerateBanterPostHTML(bestAnomaly, miniAppURL)

	if !taskData.GetPublish() {
		fmt.Println("==================================================")
		fmt.Printf("[DRY-RUN] КАРТОЧКА АНОМАЛИИ (%s):\n", bestAnomaly.BadgeText)
		fmt.Println("--------------------------------------------------")
		fmt.Println(postHTML)
		fmt.Printf("Файл постера: %s (Шаблон: %s)\n", outputPath, templatePath)
		fmt.Println("==================================================")
		return nil
	}

	// Рендерим постер 1200x1200px через Chromium
	fmt.Printf("🎨 Рендеринг постера аномалии в %s...\n", outputPath)
	if err := renderer.RenderBanterCardPNG(vm, templatePath, outputPath); err != nil {
		return fmt.Errorf("render banter card png: %w", err)
	}

	tg := config.Config.TG
	if tg == nil || tg.BotToken == "" || tg.ChannelID == "" {
		return fmt.Errorf("telegram configuration missing")
	}

	// Вариант А: отправка без инлайн-кнопок для включения нативного треда комментариев Telegram
	fmt.Printf("🚀 Отправка карточки аномалии в Telegram (%s)...\n", tg.ChannelID)
	msgID, err := SendPhotoWithResult(ctx, tg.BotToken, tg.ChannelID, postHTML, outputPath)
	if err != nil {
		return fmt.Errorf("send banter photo to telegram: %w", err)
	}

	channelIDInt, _ := strconv.ParseInt(tg.ChannelID, 10, 64)
	storage := getStorage(ctx)
	if storage != nil {
		lID := bestAnomaly.LeagueID
		rID := bestAnomaly.RoundID
		pub := &Publication{
			ChannelID:       channelIDInt,
			LeagueID:        &lID,
			RoundID:         &rID,
			PublicationType: PublicationType("banter"),
			Status:          StatusPublished,
			MediaType:       MediaTypePhoto,
			MediaURL:        outputPath,
			TextContent:     postHTML,
			MessageID:       &msgID,
		}
		_, _ = storage.Create(ctx, pub)
	}

	fmt.Printf("✅ Карточка аномалии успешно опубликована! Telegram Message ID: %d\n", msgID)
	return nil
}
