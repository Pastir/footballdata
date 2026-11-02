package press_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Pastir/footballdata/press"
)

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/">
  <channel>
    <title>Sample Sports Feed</title>
    <item>
      <title><![CDATA[Pep Guardiola signs contract extension at Manchester City]]></title>
      <description><![CDATA[The manager has agreed terms on a new two-year deal at the Etihad.]]></description>
      <link>https://example.com/pep-contract</link>
      <pubDate>Mon, 02 Nov 2026 14:00:00 GMT</pubDate>
      <media:content url="https://example.com/pep.jpg" medium="image"/>
    </item>
    <item>
      <title><![CDATA[Arsenal Women cruise to win]]></title>
      <description><![CDATA[A routine victory in the league.]]></description>
      <link>https://example.com/women-match</link>
      <pubDate>Mon, 02 Nov 2026 15:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

func TestFetcher_FetchFeed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer ts.Close()

	fetcher := press.NewFetcher(5 * time.Second)
	feed := press.RSSFeedConfig{
		Source:      "test_feed",
		DisplayName: "Test Feed",
		FlagEmoji:   "⚽",
		URL:         ts.URL,
	}

	articles, err := fetcher.FetchFeed(context.Background(), feed)
	if err != nil {
		t.Fatalf("FetchFeed failed: %v", err)
	}

	if len(articles) != 2 {
		t.Fatalf("expected 2 articles, got %d", len(articles))
	}

	// 1st article (Pep contract) should have status 'draft' and high score
	pepArticle := articles[0]
	if pepArticle.OriginalURL != "https://example.com/pep-contract" {
		t.Errorf("wrong url: %s", pepArticle.OriginalURL)
	}
	if pepArticle.ImageURL != "https://example.com/pep.jpg" {
		t.Errorf("wrong image url: %s", pepArticle.ImageURL)
	}
	if pepArticle.Status != "draft" {
		t.Errorf("expected draft status, got %s", pepArticle.Status)
	}
	if pepArticle.HypeScore < 50 {
		t.Errorf("expected high hype score, got %d", pepArticle.HypeScore)
	}

	// 2nd article (women's team) should have status 'skipped' and hype score 0
	womenArticle := articles[1]
	if womenArticle.Status != "skipped" {
		t.Errorf("expected skipped status for noise, got %s", womenArticle.Status)
	}
	if womenArticle.HypeScore != 0 {
		t.Errorf("expected 0 hype score for noise, got %d", womenArticle.HypeScore)
	}
}

