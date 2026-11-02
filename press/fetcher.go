package press

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RSS XML структуры (совместимо со спецификацией RSS 2.0 и Media RSS)
type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title          string             `xml:"title"`
	Link           string             `xml:"link"`
	Description    string             `xml:"description"`
	PubDate        string             `xml:"pubDate"`
	MediaThumbnail []rssMediaThumb    `xml:"http://search.yahoo.com/mrss/ thumbnail"`
	MediaContent   []rssMediaContent  `xml:"http://search.yahoo.com/mrss/ content"`
	Enclosure      *rssEnclosure      `xml:"enclosure"`
}

type rssMediaThumb struct {
	URL string `xml:"url,attr"`
}

type rssMediaContent struct {
	URL    string `xml:"url,attr"`
	Medium string `xml:"medium,attr"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

// Fetcher отвечает за опрос и парсинг европейских RSS-источников
type Fetcher struct {
	httpClient *http.Client
}

// NewFetcher создает новый экземпляр сборщика новостей
func NewFetcher(timeout time.Duration) *Fetcher {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Fetcher{
		httpClient: &http.Client{Timeout: timeout},
	}
}

// FetchFeed загружает и парсит конкретную RSS-ленту
func (f *Fetcher) FetchFeed(ctx context.Context, feed RSSFeedConfig) ([]Article, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Эмулируем стандартный браузер для избежания бана по User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 FootballDataLab/2.0")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", feed.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, feed.URL)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var root rssRoot
	if err = xml.Unmarshal(bodyBytes, &root); err != nil {
		return nil, fmt.Errorf("xml unmarshal from %s: %w", feed.URL, err)
	}

	var articles []Article
	now := time.Now()

	for _, item := range root.Channel.Items {
		url := strings.TrimSpace(item.Link)
		title := CleanText(item.Title)
		lead := CleanText(item.Description)

		if url == "" || title == "" {
			continue
		}

		// Поиск наилучшей картинки
		imageURL := ""
		if len(item.MediaContent) > 0 && item.MediaContent[0].URL != "" {
			imageURL = item.MediaContent[0].URL
		} else if len(item.MediaThumbnail) > 0 && item.MediaThumbnail[0].URL != "" {
			imageURL = item.MediaThumbnail[0].URL
		} else if item.Enclosure != nil && strings.HasPrefix(item.Enclosure.Type, "image/") {
			imageURL = item.Enclosure.URL
		}

		pubTime := parseDate(item.PubDate, now)
		score := CalculateHypeScore(title, lead)

		status := "draft"
		if score < 35 {
			status = "skipped"
		}

		articles = append(articles, Article{
			Source:        feed.Source,
			LeagueID:      feed.LeagueID,
			OriginalURL:   url,
			OriginalTitle: title,
			OriginalLead:  lead,
			ImageURL:      imageURL,
			Status:        status,
			PublishedAt:   pubTime,
			CreatedAt:     now,
			UpdatedAt:     now,
			HypeScore:     score,
			SourceBadge:   fmt.Sprintf("%s %s", feed.FlagEmoji, feed.DisplayName),
		})
	}

	return articles, nil
}

// parseDate пытается распарсить дату публикации в различных форматах RSS
func parseDate(raw string, fallback time.Time) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}

	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
	}

	for _, f := range formats {
		if t, err := time.Parse(f, raw); err == nil {
			return t
		}
	}

	return fallback
}

