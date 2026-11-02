package press

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Storage обеспечивает взаимодействие с таблицей press_articles через pgxpool.Pool
type Storage struct {
	pool *pgxpool.Pool
}

// NewStorage создает экземпляр хранилища статей прессы
func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

// SaveArticles сохраняет пачку статей с дедупликацией по original_url
func (s *Storage) SaveArticles(ctx context.Context, articles []Article) (int, error) {
	if len(articles) == 0 {
		return 0, nil
	}

	query := `
		INSERT INTO press_articles (
			source,
			league_id,
			original_url,
			original_title,
			original_lead,
			image_url,
			status,
			published_at,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
		ON CONFLICT (original_url) DO NOTHING
	`

	insertedCount := 0
	now := time.Now()

	for _, a := range articles {
		cmdTag, err := s.pool.Exec(
			ctx,
			query,
			a.Source,
			a.LeagueID,
			a.OriginalURL,
			a.OriginalTitle,
			a.OriginalLead,
			a.ImageURL,
			a.Status,
			a.PublishedAt,
			now,
			now,
		)
		if err != nil {
			continue // Игнорируем единичные сбои
		}
		if cmdTag.RowsAffected() > 0 {
			insertedCount++
		}
	}

	return insertedCount, nil
}

// GetTopDraftArticles возвращает свежие не обработанные статьи со статусом 'draft'
func (s *Storage) GetTopDraftArticles(ctx context.Context, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT 
			id,
			source,
			league_id,
			original_url,
			original_title,
			COALESCE(original_lead, ''),
			COALESCE(image_url, ''),
			COALESCE(summary_ru, ''),
			status,
			tg_message_id,
			published_at,
			created_at,
			updated_at
		FROM press_articles
		WHERE status = 'draft' AND summary_ru IS NULL
		ORDER BY published_at DESC, id DESC
		LIMIT $1
	`

	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query top drafts: %w", err)
	}
	defer rows.Close()

	var list []Article
	for rows.Next() {
		var a Article
		var lead, img, sum string
		var tgMsg *int64

		err := rows.Scan(
			&a.ID,
			&a.Source,
			&a.LeagueID,
			&a.OriginalURL,
			&a.OriginalTitle,
			&lead,
			&img,
			&sum,
			&a.Status,
			&tgMsg,
			&a.PublishedAt,
			&a.CreatedAt,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}

		a.OriginalLead = lead
		a.ImageURL = img
		a.SummaryRU = sum
		a.TGMessageID = tgMsg
		a.HypeScore = CalculateHypeScore(a.OriginalTitle, a.OriginalLead)

		list = append(list, a)
	}

	return list, nil
}

// UpdateArticleSummary сохраняет сгенерированный текст выжимки
func (s *Storage) UpdateArticleSummary(ctx context.Context, id int64, summaryRu string, status string) error {
	query := `
		UPDATE press_articles
		SET summary_ru = $1, status = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := s.pool.Exec(ctx, query, summaryRu, status, id)
	return err
}

// MarkArticlePublished переводит статью в статус published с привязкой message_id
func (s *Storage) MarkArticlePublished(ctx context.Context, id int64, tgMsgID int64) error {
	query := `
		UPDATE press_articles
		SET status = 'published', tg_message_id = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := s.pool.Exec(ctx, query, tgMsgID, id)
	return err
}
