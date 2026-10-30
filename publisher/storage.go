package publisher

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Storage определяет интерфейс хранилища публикаций.
type Storage interface {
	EnsureSchema(ctx context.Context) error
	Create(ctx context.Context, pub *Publication) (int64, error)
	GetByID(ctx context.Context, id int64) (*Publication, error)
	IsAlreadyPublished(ctx context.Context, channelID int64, pubType PublicationType, leagueID, roundID int) (bool, error)
	GetLastPublished(ctx context.Context, channelID int64, pubType PublicationType, leagueID, roundID int) (*Publication, error)
	MarkPublished(ctx context.Context, id int64, messageID int64) error
	MarkFailed(ctx context.Context, id int64, errMsg string) error
	UpdateStatus(ctx context.Context, id int64, status PublicationStatus) error
	ListPendingScheduled(ctx context.Context, dueBefore time.Time, limit int) ([]Publication, error)
}

// PGStorage реализует Storage поверх PostgreSQL (pgxpool.Pool).
type PGStorage struct {
	pool *pgxpool.Pool
}

// NewPGStorage создаёт новый экземпляр PGStorage.
func NewPGStorage(pool *pgxpool.Pool) *PGStorage {
	return &PGStorage{pool: pool}
}

// EnsureSchema создаёт таблицу telegram_publications и индексы, если они отсутствуют.
func (s *PGStorage) EnsureSchema(ctx context.Context) error {
	const query = `
CREATE TABLE IF NOT EXISTS telegram_publications (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    message_id BIGINT,
    publication_type VARCHAR(50) NOT NULL,
    league_id INT,
    season_id INT,
    round_id INT,
    fixture_id INT,
    title VARCHAR(255) NOT NULL,
    text_content TEXT,
    media_url TEXT,
    media_type VARCHAR(20) DEFAULT 'photo',
    metadata JSONB DEFAULT '{}'::jsonb,
    status VARCHAR(30) NOT NULL DEFAULT 'draft',
    scheduled_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    retry_count INT DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tg_pub_status_scheduled ON telegram_publications(status, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_tg_pub_league_round ON telegram_publications(league_id, round_id);
CREATE INDEX IF NOT EXISTS idx_tg_pub_type ON telegram_publications(publication_type);
`
	_, err := s.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ensure telegram_publications schema: %w", err)
	}
	return nil
}

// Create сохраняет новую публикацию в БД и возвращает её ID.
func (s *PGStorage) Create(ctx context.Context, pub *Publication) (int64, error) {
	if pub.Status == "" {
		pub.Status = StatusDraft
	}
	if pub.MediaType == "" {
		pub.MediaType = MediaTypePhoto
	}

	metadataJSON := []byte("{}")
	if len(pub.Metadata) > 0 {
		metadataJSON = pub.Metadata
	}

	const query = `
INSERT INTO telegram_publications (
    channel_id, message_id, publication_type,
    league_id, season_id, round_id, fixture_id,
    title, text_content, media_url, media_type, metadata,
    status, scheduled_at, published_at, retry_count, error_message,
    created_at, updated_at
) VALUES (
    $1, $2, $3,
    $4, $5, $6, $7,
    $8, $9, $10, $11, $12,
    $13, $14, $15, $16, $17,
    NOW(), NOW()
) RETURNING id`

	var id int64
	err := s.pool.QueryRow(ctx, query,
		pub.ChannelID, pub.MessageID, string(pub.PublicationType),
		pub.LeagueID, pub.SeasonID, pub.RoundID, pub.FixtureID,
		pub.Title, pub.TextContent, pub.MediaURL, string(pub.MediaType), metadataJSON,
		string(pub.Status), pub.ScheduledAt, pub.PublishedAt, pub.RetryCount, pub.ErrorMessage,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert telegram publication: %w", err)
	}

	pub.ID = id
	return id, nil
}

// GetByID возвращает публикацию по ID.
func (s *PGStorage) GetByID(ctx context.Context, id int64) (*Publication, error) {
	const query = `
SELECT 
    id, channel_id, message_id, publication_type,
    league_id, season_id, round_id, fixture_id,
    title, COALESCE(text_content, ''), COALESCE(media_url, ''), COALESCE(media_type, 'photo'),
    metadata, status, scheduled_at, published_at,
    retry_count, COALESCE(error_message, ''), created_at, updated_at
FROM telegram_publications
WHERE id = $1`

	row := s.pool.QueryRow(ctx, query, id)
	pub, err := scanPublication(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get publication %d: %w", id, err)
	}
	return pub, nil
}

// IsAlreadyPublished проверяет, был ли уже успешно опубликован пост для заданного канала, типа, лиги и тура.
func (s *PGStorage) IsAlreadyPublished(
	ctx context.Context,
	channelID int64,
	pubType PublicationType,
	leagueID, roundID int,
) (bool, error) {
	const query = `
SELECT EXISTS(
    SELECT 1 FROM telegram_publications
    WHERE channel_id = $1
      AND publication_type = $2
      AND league_id = $3
      AND round_id = $4
      AND status = 'published'
)`
	var exists bool
	err := s.pool.QueryRow(ctx, query, channelID, string(pubType), leagueID, roundID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check is already published: %w", err)
	}
	return exists, nil
}

// GetLastPublished возвращает последнюю успешную публикацию для заданного тура.
func (s *PGStorage) GetLastPublished(
	ctx context.Context,
	channelID int64,
	pubType PublicationType,
	leagueID, roundID int,
) (*Publication, error) {
	const query = `
SELECT 
    id, channel_id, message_id, publication_type,
    league_id, season_id, round_id, fixture_id,
    title, COALESCE(text_content, ''), COALESCE(media_url, ''), COALESCE(media_type, 'photo'),
    metadata, status, scheduled_at, published_at,
    retry_count, COALESCE(error_message, ''), created_at, updated_at
FROM telegram_publications
WHERE channel_id = $1
  AND publication_type = $2
  AND league_id = $3
  AND round_id = $4
  AND status = 'published'
ORDER BY published_at DESC NULLS LAST
LIMIT 1`

	row := s.pool.QueryRow(ctx, query, channelID, string(pubType), leagueID, roundID)
	pub, err := scanPublication(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get last published: %w", err)
	}
	return pub, nil
}

// MarkPublished фиксирует успешную отправку сообщения в Telegram.
func (s *PGStorage) MarkPublished(ctx context.Context, id int64, messageID int64) error {
	const query = `
UPDATE telegram_publications
SET status = 'published',
    message_id = $2,
    published_at = NOW(),
    updated_at = NOW(),
    error_message = NULL
WHERE id = $1`

	res, err := s.pool.Exec(ctx, query, id, messageID)
	if err != nil {
		return fmt.Errorf("mark published %d: %w", id, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("publication %d not found", id)
	}
	return nil
}

// MarkFailed фиксирует ошибку отправки сообщения.
func (s *PGStorage) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	const query = `
UPDATE telegram_publications
SET status = 'failed',
    retry_count = retry_count + 1,
    error_message = $2,
    updated_at = NOW()
WHERE id = $1`

	res, err := s.pool.Exec(ctx, query, id, errMsg)
	if err != nil {
		return fmt.Errorf("mark failed %d: %w", id, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("publication %d not found", id)
	}
	return nil
}

// UpdateStatus обновляет статус публикации.
func (s *PGStorage) UpdateStatus(ctx context.Context, id int64, status PublicationStatus) error {
	const query = `
UPDATE telegram_publications
SET status = $2,
    updated_at = NOW()
WHERE id = $1`

	res, err := s.pool.Exec(ctx, query, id, string(status))
	if err != nil {
		return fmt.Errorf("update status %d to %s: %w", id, status, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("publication %d not found", id)
	}
	return nil
}

// ListPendingScheduled возвращает публикации со статусом 'scheduled', готовые к выпуску.
func (s *PGStorage) ListPendingScheduled(ctx context.Context, dueBefore time.Time, limit int) ([]Publication, error) {
	if limit <= 0 {
		limit = 50
	}

	const query = `
SELECT 
    id, channel_id, message_id, publication_type,
    league_id, season_id, round_id, fixture_id,
    title, COALESCE(text_content, ''), COALESCE(media_url, ''), COALESCE(media_type, 'photo'),
    metadata, status, scheduled_at, published_at,
    retry_count, COALESCE(error_message, ''), created_at, updated_at
FROM telegram_publications
WHERE status = 'scheduled'
  AND scheduled_at <= $1
ORDER BY scheduled_at ASC
LIMIT $2`

	rows, err := s.pool.Query(ctx, query, dueBefore, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending scheduled: %w", err)
	}
	defer rows.Close()

	var result []Publication
	for rows.Next() {
		pub, err := scanPublication(rows)
		if err != nil {
			return nil, fmt.Errorf("scan scheduled publication: %w", err)
		}
		result = append(result, *pub)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPublication(s rowScanner) (*Publication, error) {
	var (
		p            Publication
		pubType      string
		mediaType    string
		status       string
		metadataJSON []byte
	)

	err := s.Scan(
		&p.ID,
		&p.ChannelID,
		&p.MessageID,
		&pubType,
		&p.LeagueID,
		&p.SeasonID,
		&p.RoundID,
		&p.FixtureID,
		&p.Title,
		&p.TextContent,
		&p.MediaURL,
		&mediaType,
		&metadataJSON,
		&status,
		&p.ScheduledAt,
		&p.PublishedAt,
		&p.RetryCount,
		&p.ErrorMessage,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	p.PublicationType = PublicationType(pubType)
	p.MediaType = MediaType(mediaType)
	p.Status = PublicationStatus(status)
	if len(metadataJSON) > 0 {
		p.Metadata = metadataJSON
	}

	return &p, nil
}
