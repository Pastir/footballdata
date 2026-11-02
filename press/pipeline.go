package press

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/Pastir/footballdata/config"
	"github.com/jackc/pgx/v4/pgxpool"
)

// RunPipeline запускает полный цикл европейского медиа-дайджеста:
// 1. Опрос RSS-источников (BBC, Marca, The Guardian);
// 2. Дедупликация и сохранение в PostgreSQL;
// 3. Выборка самых горячих инсайдов (по HypeScore);
// 4. Генерация русскоязычной выжимки через Gemini (LLM);
// 5. Публикация в Telegram-канал (или dry-run вывод).
func RunPipeline(ctx context.Context, pool *pgxpool.Pool, cfg *config.Conf, publish bool, limit int) error {
	if limit <= 0 {
		limit = 1 // По умолчанию берем 1 самый горячий материал за запуск
	}

	storage := NewStorage(pool)
	fetcher := NewFetcher(15 * time.Second)
	summarizer := NewSummarizer(cfg.LLM)
	dispatcher := NewDispatcher(storage, summarizer, cfg)

	fmt.Println("📡 [1/4] Запуск сбора европейских RSS-лент...")
	feeds := DefaultFeeds()
	var collected []Article

	for _, feed := range feeds {
		feedCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		articles, err := fetcher.FetchFeed(feedCtx, feed)
		cancel()

		if err != nil {
			log.Printf("⚠️ Ошибка сбора %s: %v", feed.DisplayName, err)
			continue
		}

		collected = append(collected, articles...)
	}

	fmt.Printf("📥 [2/4] Загружено %d статей. Сохранение в БД с дедупликацией...\n", len(collected))
	inserted, err := storage.SaveArticles(ctx, collected)
	if err != nil {
		return fmt.Errorf("save articles to db: %w", err)
	}
	fmt.Printf("💾 Новых уникальных статей добавлено в БД: %d\n", inserted)

	fmt.Println("🔍 [3/4] Выборка топовых инсайдов для обработки...")
	topDrafts, err := storage.GetTopDraftArticles(ctx, limit*3)
	if err != nil {
		return fmt.Errorf("get top draft articles: %w", err)
	}

	if len(topDrafts) == 0 {
		fmt.Println("ℹ️ Нет новых статей для публикации (все актуальные материалы уже обработаны).")
		return nil
	}

	// Сортируем кандидатов по HypeScore (самые кликабельные первыми)
	sort.Slice(topDrafts, func(i, j int) bool {
		return topDrafts[i].HypeScore > topDrafts[j].HypeScore
	})

	processCount := limit
	if processCount > len(topDrafts) {
		processCount = len(topDrafts)
	}

	fmt.Printf("⚡ [4/4] Генерация выжимки и публикация (Выбрано статей: %d, Publish: %v)...\n", processCount, publish)

	for i := 0; i < processCount; i++ {
		article := topDrafts[i]
		fmt.Printf("👉 Обработка #%d (ID: %d, Score: %d): %s\n", i+1, article.ID, article.HypeScore, article.OriginalTitle)

		_, err := dispatcher.ProcessAndPublishArticle(ctx, article, publish)
		if err != nil {
			log.Printf("❌ Ошибка обработки статьи ID %d: %v", article.ID, err)
			continue
		}
	}

	return nil
}
