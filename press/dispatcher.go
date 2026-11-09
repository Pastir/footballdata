package press

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/publisher"
)

// Dispatcher отвечает за оформление и публикацию статьи в Telegram
type Dispatcher struct {
	storage    *Storage
	summarizer *Summarizer
	botToken   string
	channelID  string
	webAppURL  string
}

// NewDispatcher создает новый экземпляр диспетчера публикаций
func NewDispatcher(storage *Storage, summarizer *Summarizer, cfg *config.Conf) *Dispatcher {
	botToken := ""
	channelID := ""
	webAppURL := "https://data-lab.live/"

	if cfg.TG != nil {
		botToken = cfg.TG.BotToken
		channelID = cfg.TG.ChannelID
	}
	if cfg.WebApp != nil && cfg.WebApp.BaseURL != "" {
		webAppURL = cfg.WebApp.BaseURL
	}

	return &Dispatcher{
		storage:    storage,
		summarizer: summarizer,
		botToken:   botToken,
		channelID:  channelID,
		webAppURL:  webAppURL,
	}
}

// ProcessAndPublishArticle генерирует выжимку (если её нет) и отправляет пост в Telegram
func (d *Dispatcher) ProcessAndPublishArticle(ctx context.Context, a Article, publish bool) (*Article, error) {
	// 1. Генерация текста выжимки через LLM
	summaryRu := a.SummaryRU
	if summaryRu == "" {
		generated, err := d.summarizer.GenerateSummary(ctx, a)
		if err != nil {
			return nil, fmt.Errorf("generate summary for article %d: %w", a.ID, err)
		}
		summaryRu = generated
		a.SummaryRU = summaryRu

		// Сохраняем черновик выжимки в БД (при dry-run оставляем статус 'draft', чтобы не терять статью)
		nextStatus := "draft"
		if publish {
			nextStatus = "ready"
		}
		_ = d.storage.UpdateArticleSummary(ctx, a.ID, summaryRu, nextStatus)
	}

	// 2. Сборка финального HTML-текста поста для Telegram
	badge := a.SourceBadge
	if badge == "" {
		badge = strings.ToUpper(a.Source)
	}

	miniAppURL := "https://t.me/football_data_lab_bot/data_lab"
	if config.Config.WebApp != nil {
		botUser := config.Config.WebApp.BotUsername
		if botUser == "" {
			botUser = "football_data_lab_bot"
		}
		shortName := config.Config.WebApp.BotShortName
		if shortName == "" {
			shortName = "data_lab"
		}
		miniAppURL = fmt.Sprintf("https://t.me/%s/%s", botUser, shortName)
	}

	postText := fmt.Sprintf(
		"<b>%s</b>\n\n%s\n\n👉 <i>Источник: <a href=\"%s\">%s</a></i>\n📊 <i><a href=\"%s\">Открыть Football Data Lab Mini App</a></i>",
		html.EscapeString(badge),
		summaryRu,
		a.OriginalURL,
		html.EscapeString(a.OriginalTitle),
		miniAppURL,
	)

	// Опции медиа-превью (показываем крупно сверху)
	previewURL := a.ImageURL
	if previewURL == "" {
		previewURL = a.OriginalURL
	}

	linkPreview := &publisher.LinkPreviewOptions{
		URL:              previewURL,
		PreferLargeMedia: true,
		ShowAboveText:    true,
	}

	if !publish {
		// Dry-run: выводим превью поста в консоль
		fmt.Println("==================================================")
		fmt.Printf("[DRY-RUN] СТАТЬЯ ID %d (Score: %d):\n", a.ID, a.HypeScore)
		fmt.Println("--------------------------------------------------")
		fmt.Println(postText)
		fmt.Printf("Обложка: %s\n", previewURL)
		fmt.Println("==================================================")
		return &a, nil
	}

	// 3. Отправка в реальный Telegram-канал (без инлайн-кнопок для нативного треда комментариев)
	if d.botToken == "" || d.channelID == "" {
		return nil, fmt.Errorf("telegram bot token or channel id is empty in config")
	}

	msgID, err := publisher.SendTextMessageWithKeyboardAndPreview(
		ctx,
		d.botToken,
		d.channelID,
		postText,
		nil,
		linkPreview,
	)
	if err != nil {
		return nil, fmt.Errorf("send to telegram: %w", err)
	}

	a.TGMessageID = &msgID
	a.Status = "published"
	_ = d.storage.MarkArticlePublished(ctx, a.ID, msgID)

	fmt.Printf("✅ Статья #%d опубликована в Telegram! Message ID: %d\n", a.ID, msgID)
	return &a, nil
}

