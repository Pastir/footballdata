package copywriter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/copywriter"
)

func TestGenerateRoundResultsHTMLPost(t *testing.T) {
	c0, c3 := 0, 3

	cfg := copywriter.RoundResultsPostConfig{
		LeagueName:   "Premier League",
		RoundName:    "6-й тур",
		LeagueID:     8,
		SeasonID:     28083,
		RoundID:      407869,
		BotUsername:  "football_data_lab_bot",
		BotShortName: "data_lab",
	}

	matches := []analytics.RoundResultMatch{
		{
			FixtureID:    19722148,
			FixtureStart: time.Date(2026, time.October, 10, 15, 0, 0, 0, time.UTC),
			TeamHomeName: "Liverpool",
			TeamAwayName: "Man City",
			Scores: analytics.MatchScores{
				HomeCurrent: &c0,
				AwayCurrent: &c3,
				HomeHT:      &c0,
				AwayHT:      &c0,
			},
			IsRival: true,
		},
	}

	summary := analytics.RoundSummary{
		TotalMatches:     1,
		FinishedMatches:  1,
		TotalGoals:       3,
		AvgGoals:         3.0,
		AwayWins:         1,
		TopScoringMatch:  "Liverpool — Man City (3 голов)",
		TopUpsetMatch:    "Liverpool — Man City (победа за кэф 3.20)",
		TotalYellowCards: 4,
	}

	post := copywriter.GenerateRoundResultsHTMLPost(cfg, matches, summary)

	// 1. Проверяем заголовок турнира
	if !strings.Contains(post, "PREMIER LEAGUE") || !strings.Contains(post, "РЕЗУЛЬТАТЫ") {
		t.Errorf("post header missing title: %s", post)
	}

	// 2. Проверяем гиперссылку на счёте
	expectedLink := "https://t.me/football_data_lab_bot/data_lab?startapp=8_matches"
	if !strings.Contains(post, expectedLink) {
		t.Errorf("post missing deep link %q: %s", expectedLink, post)
	}
	if !strings.Contains(post, "0:3 (0:0)") {
		t.Errorf("post missing score string '0:3 (0:0)': %s", post)
	}

	// 3. Проверяем дату на русском
	if !strings.Contains(post, "10 Окт. 2026, Суббота") {
		t.Errorf("post missing formatted date: %s", post)
	}

	// 4. Проверяем экспресс-статистику
	if !strings.Contains(post, "ЭКСПРЕСС-СТАТИСТИКА ТУРА") {
		t.Errorf("post missing express stats header: %s", post)
	}
	if !strings.Contains(post, "<b>Забито голов:</b> 3") {
		t.Errorf("post missing total goals: %s", post)
	}

	// 5. Проверяем подсказку
	if !strings.Contains(post, "Нажмите на счёт матча") {
		t.Errorf("post missing hint: %s", post)
	}
}
