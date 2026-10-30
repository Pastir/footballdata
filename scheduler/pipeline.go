package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/Pastir/footballdata/copywriter"
	"github.com/Pastir/footballdata/publisher"
)

// ContentRenderer — функция рендеринга изображения в указанный файл.
type ContentRenderer func(outputPath string) error

// PublicationTask описывает полную задачу на генерацию и выпуск контента в Telegram.
type PublicationTask struct {
	ChannelID       int64
	BotToken        string
	PublicationType publisher.PublicationType
	LeagueID        int
	SeasonID        int
	RoundID         int
	Title           string
	OutputPath      string
	FactPack        copywriter.FactPack
	RenderFunc      ContentRenderer
	ExecuteFunc     func(ctx context.Context) error
}

// Pipeline координирует сквозной цикл публикации:
// Идемпотентность → Рендеринг (PNG) → AI Copywriter (Текст) → Telegram Publisher → Фиксация в БД
type Pipeline struct {
	storage    publisher.Storage
	copywriter copywriter.Client
	maxRetries int
}

// NewPipeline создаёт новый конвейер публикации.
func NewPipeline(storage publisher.Storage, copywriter copywriter.Client, maxRetries int) *Pipeline {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &Pipeline{
		storage:    storage,
		copywriter: copywriter,
		maxRetries: maxRetries,
	}
}

// Execute выполняет задачу публикации со строгой защитой от дублей и переходом по статусам.
func (p *Pipeline) Execute(ctx context.Context, task PublicationTask) (*publisher.Publication, error) {
	if task.ExecuteFunc != nil {
		if err := task.ExecuteFunc(ctx); err != nil {
			return nil, err
		}
		return &publisher.Publication{
			ChannelID:       task.ChannelID,
			PublicationType: task.PublicationType,
			LeagueID:        &task.LeagueID,
			SeasonID:        &task.SeasonID,
			RoundID:         &task.RoundID,
			Title:           task.Title,
			Status:          publisher.StatusPublished,
		}, nil
	}

	// 1. Проверка идемпотентности (не публиковать повторно уже выпущенный материал)
	alreadyPublished, err := p.storage.IsAlreadyPublished(
		ctx,
		task.ChannelID,
		task.PublicationType,
		task.LeagueID,
		task.RoundID,
	)
	if err != nil {
		return nil, fmt.Errorf("idempotency check: %w", err)
	}
	if alreadyPublished {
		return nil, fmt.Errorf("already published: type %s, league %d, round %d", task.PublicationType, task.LeagueID, task.RoundID)
	}

	// 2. Создание записи в БД со статусом scheduled
	leagueID := task.LeagueID
	seasonID := task.SeasonID
	roundID := task.RoundID

	pub := &publisher.Publication{
		ChannelID:       task.ChannelID,
		PublicationType: task.PublicationType,
		LeagueID:        &leagueID,
		SeasonID:        &seasonID,
		RoundID:         &roundID,
		Title:           task.Title,
		MediaURL:        task.OutputPath,
		MediaType:       publisher.MediaTypePhoto,
		Status:          publisher.StatusScheduled,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	pubID, err := p.storage.Create(ctx, pub)
	if err != nil {
		return nil, fmt.Errorf("create scheduled publication: %w", err)
	}

	// 3. Рендеринг инфографики (PNG 1920px)
	_ = p.storage.UpdateStatus(ctx, pubID, publisher.StatusRendering)
	if task.RenderFunc != nil {
		if err = task.RenderFunc(task.OutputPath); err != nil {
			_ = p.storage.MarkFailed(ctx, pubID, fmt.Sprintf("render error: %v", err))
			return nil, fmt.Errorf("render infographic: %w", err)
		}
	}

	// 4. Генерация сопроводительного текста через AI Copywriter
	_ = p.storage.UpdateStatus(ctx, pubID, publisher.StatusReady)
	caption := ""
	if p.copywriter != nil {
		text, err := p.copywriter.GeneratePost(ctx, task.FactPack)
		if err == nil {
			caption = text
		} else {
			// При ошибке используем детерминированный fallback
			caption = copywriter.GenerateFallbackPost(task.FactPack)
		}
	} else {
		caption = copywriter.GenerateFallbackPost(task.FactPack)
	}
	pub.TextContent = caption

	// 5. Публикация в Telegram через Bot API
	msgID, err := publisher.SendPhotoWithResult(
		ctx,
		task.BotToken,
		fmt.Sprintf("%d", task.ChannelID),
		caption,
		task.OutputPath,
	)
	if err != nil {
		_ = p.storage.MarkFailed(ctx, pubID, fmt.Sprintf("telegram publish: %v", err))
		return nil, fmt.Errorf("telegram publish: %w", err)
	}

	// 6. Фиксация факта успешной публикации в БД
	if err = p.storage.MarkPublished(ctx, pubID, msgID); err != nil {
		return nil, fmt.Errorf("mark published: %w", err)
	}

	pub.Status = publisher.StatusPublished
	pub.MessageID = &msgID
	now := time.Now()
	pub.PublishedAt = &now

	return pub, nil
}
