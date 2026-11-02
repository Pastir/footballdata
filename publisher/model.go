package publisher

import (
	"encoding/json"
	"time"
)

// PublicationType определяет категорию футбольной публикации.
type PublicationType string

const (
	PubTypeTable         PublicationType = "table"
	PubTypeRoundResults  PublicationType = "round_results"
	PubTypeRoundFixtures PublicationType = "round_fixtures"
	PubTypeRoundDigest   PublicationType = "round_digest"
	PubTypeLeaders       PublicationType = "leaders"
	PubTypeFacts         PublicationType = "facts"
	PubTypePrediction    PublicationType = "prediction"
	PubTypeExpress       PublicationType = "express"
	PubTypeInsight       PublicationType = "insight"
	PubTypeBanter        PublicationType = "banter"
)

// PublicationStatus определяет статус жизненного цикла публикации.
type PublicationStatus string

const (
	StatusDraft     PublicationStatus = "draft"
	StatusScheduled PublicationStatus = "scheduled"
	StatusRendering PublicationStatus = "rendering"
	StatusReady     PublicationStatus = "ready"
	StatusPublished PublicationStatus = "published"
	StatusFailed    PublicationStatus = "failed"
	StatusCancelled PublicationStatus = "cancelled"
	StatusSkipped   PublicationStatus = "skipped"
)

// MediaType определяет тип медиа-вложения в Telegram.
type MediaType string

const (
	MediaTypePhoto    MediaType = "photo"
	MediaTypeDocument MediaType = "document"
	MediaTypeText     MediaType = "text"
)

// Publication представляет запись о посте в Telegram-канал.
type Publication struct {
	ID              int64             `json:"id"`
	ChannelID       int64             `json:"channel_id"`
	MessageID       *int64            `json:"message_id,omitempty"`
	PublicationType PublicationType   `json:"publication_type"`

	// Связь с футбольными сущностями
	LeagueID  *int `json:"league_id,omitempty"`
	SeasonID  *int `json:"season_id,omitempty"`
	RoundID   *int `json:"round_id,omitempty"`
	FixtureID *int `json:"fixture_id,omitempty"`

	// Контент
	Title       string          `json:"title"`
	TextContent string          `json:"text_content"`
	MediaURL    string          `json:"media_url"`
	MediaType   MediaType       `json:"media_type"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`

	// Жизненный цикл
	Status      PublicationStatus `json:"status"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	PublishedAt *time.Time        `json:"published_at,omitempty"`

	// Ошибки и повторы
	RetryCount   int       `json:"retry_count"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsTerminal возвращает true, если публикация перешла в конечное состояние.
func (p *Publication) IsTerminal() bool {
	return p.Status == StatusPublished || p.Status == StatusCancelled
}

// CanRetry возвращает true, если упавшую публикацию можно повторить.
func (p *Publication) CanRetry(maxRetries int) bool {
	return p.Status == StatusFailed && p.RetryCount < maxRetries
}
