package copywriter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/copywriter"
)

func TestGenerateFactHTMLPost(t *testing.T) {
	rec := analytics.StreakRecord{
		ID:             42,
		LeagueID:       8,
		LeagueName:     "Premier League",
		EntityName:     "Манчестер Сити",
		CoachName:      "Пеп Гвардиола",
		StreakType:     analytics.StreakTypeTeamWin,
		StreakTypeName: "Победы",
		StreakTypeIcon: "🔥",
		Scope:          analytics.StreakScopeLeague,
		ScopeName:      "Чемпионат",
		Length:         6,
		IsActive:       true,
		IsRecordClub:   true,
		PrevClubRecord: 5,
		FixturesHistory: []analytics.StreakMatchHistoryItem{
			{
				FixtureID:     101,
				Date:          time.Now().AddDate(0, 0, -10),
				DateStr:       "10.10",
				OpponentName:  "Челси",
				TeamGoals:     2,
				OpponentGoals: 0,
				IsHome:        true,
				Result:        "W",
			},
			{
				FixtureID:     102,
				Date:          time.Now().AddDate(0, 0, -5),
				DateStr:       "15.10",
				OpponentName:  "Ливерпуль",
				TeamGoals:     3,
				OpponentGoals: 1,
				IsHome:        false,
				Result:        "W",
			},
		},
		Metrics: map[string]interface{}{
			"goals_scored":   18,
			"goals_conceded": 3,
			"clean_sheets":   4,
		},
	}

	cfg := copywriter.FactPostConfig{
		LeagueName:   "Premier League",
		LeagueID:     8,
		BotUsername:  "football_data_lab_bot",
		BotShortName: "FDL",
	}

	post := copywriter.GenerateFactHTMLPost(rec, cfg)

	if !strings.Contains(post, "МАНЧЕСТЕР СИТИ: 6 ПОБЕД ПОДРЯД") {
		t.Errorf("expected team and streak length in title, got: %s", post)
	}

	if !strings.Contains(post, "НОВЫЙ КЛУБНЫЙ РЕКОРД") {
		t.Errorf("expected club record banner, got: %s", post)
	}

	if !strings.Contains(post, "Пеп Гвардиола") {
		t.Errorf("expected coach name, got: %s", post)
	}

	if !strings.Contains(post, "vs Челси") {
		t.Errorf("expected match history item, got: %s", post)
	}

	if !strings.Contains(post, "Забито голов: <b>18</b>") {
		t.Errorf("expected goals metric, got: %s", post)
	}

	deepLink := copywriter.BuildFactMiniAppURL(cfg, rec)
	expectedLink := "https://t.me/football_data_lab_bot/FDL?startapp=streak_42"
	if deepLink != expectedLink {
		t.Errorf("expected deep link %s, got %s", expectedLink, deepLink)
	}
}
