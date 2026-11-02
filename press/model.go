package press

import "time"

// Article описывает единицу европейской спортивной новости
type Article struct {
	ID            int64     `json:"id"`
	Source        string    `json:"source"`
	LeagueID      *int      `json:"league_id,omitempty"`
	OriginalURL   string    `json:"original_url"`
	OriginalTitle string    `json:"original_title"`
	OriginalLead  string    `json:"original_lead"`
	ImageURL      string    `json:"image_url"`
	SummaryRU     string    `json:"summary_ru"`
	Status        string    `json:"status"` // "draft", "ready", "published", "skipped"
	TGMessageID   *int64    `json:"tg_message_id,omitempty"`
	PublishedAt   time.Time `json:"published_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Внутренний скоринг важности (не сохраняется в БД напрямую)
	HypeScore   int    `json:"hype_score"`
	SourceBadge string `json:"source_badge"`
}

// RSSFeedConfig описывает источник новостей для сбора
type RSSFeedConfig struct {
	Source      string
	DisplayName string
	FlagEmoji   string
	URL         string
	LeagueID    *int
}

// DefaultFeeds возвращает список проверенных европейских футбольных RSS-лент
func DefaultFeeds() []RSSFeedConfig {
	eplID := 8
	laligaID := 564

	return []RSSFeedConfig{
		{
			Source:      "bbc",
			DisplayName: "BBC Sport",
			FlagEmoji:   "🇬🇧",
			URL:         "https://feeds.bbci.co.uk/sport/football/rss.xml",
			LeagueID:    &eplID,
		},
		{
			Source:      "marca_laliga",
			DisplayName: "MARCA",
			FlagEmoji:   "🇪🇸",
			URL:         "https://e00-marca.uecdn.es/rss/futbol/primera-division.xml",
			LeagueID:    &laligaID,
		},
		{
			Source:      "marca_fichajes",
			DisplayName: "MARCA Fichajes",
			FlagEmoji:   "🔥",
			URL:         "https://e00-marca.uecdn.es/rss/futbol/mercado-fichajes.xml",
			LeagueID:    &laligaID,
		},
		{
			Source:      "theguardian",
			DisplayName: "The Guardian",
			FlagEmoji:   "🇬🇧",
			URL:         "https://www.theguardian.com/football/rss",
			LeagueID:    &eplID,
		},
	}
}

