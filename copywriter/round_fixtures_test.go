package copywriter

import (
	"strings"
	"testing"

	"github.com/Pastir/footballdata/viewmodel"
)

func TestGenerateRoundFixturesHTMLPost(t *testing.T) {
	cfg := RoundFixturesPostConfig{
		LeagueName:   "Premier League",
		RoundName:    "Тур 7",
		LeagueID:     8,
		SeasonID:     28083,
		RoundID:      407870,
		BotUsername:  "football_data_lab_bot",
		BotShortName: "data_lab",
	}

	vm := viewmodel.RoundFixturesViewModel{
		LeagueID:   8,
		LeagueName: "Premier League",
		RoundTitle: "Тур 7",
		DayGroups: []viewmodel.FixtureDayGroup{
			{
				DayLabel: "17 Окт. 2026, Суббота",
				Matches: []viewmodel.FixtureMatchItem{
					{
						ID:           19722153,
						HomeTeamName: "Арсенал",
						AwayTeamName: "Челси",
						KickoffTime:  "14:30",
						OddsHome:     "1.85",
						OddsDraw:     "3.60",
						OddsAway:     "4.20",
					},
					{
						ID:           19722154,
						HomeTeamName: "Астон Вилла",
						AwayTeamName: "Вулверхэмптон",
						KickoffTime:  "17:00",
						OddsHome:     "2.10",
						OddsDraw:     "3.40",
						OddsAway:     "3.50",
					},
				},
			},
		},
		FeaturedMatch: &viewmodel.FixtureMatchItem{
			HomeTeamName: "Арсенал",
			AwayTeamName: "Челси",
			ProbHome:     "52%",
			ProbDraw:     "26%",
			ProbAway:     "22%",
		},
	}

	post := GenerateRoundFixturesHTMLPost(cfg, vm)

	if !strings.Contains(post, "PREMIER LEAGUE · ТУР 7 — РАСПИСАНИЕ") {
		t.Errorf("expected header in post, got:\n%s", post)
	}

	if !strings.Contains(post, "17 Окт. 2026, Суббота") {
		t.Errorf("expected day header in post, got:\n%s", post)
	}

	if !strings.Contains(post, "[ 1.85 · 3.60 · 4.20 ]") {
		t.Errorf("expected odds in post, got:\n%s", post)
	}

	if !strings.Contains(post, "https://t.me/football_data_lab_bot/data_lab") {
		t.Errorf("expected Mini App link in post, got:\n%s", post)
	}

	if !strings.Contains(post, "МАТЧ ТУРА: Арсенал — Челси") {
		t.Errorf("expected featured match block in post, got:\n%s", post)
	}

	if !strings.Contains(post, "П1 52% | Х 26% | П2 22%") {
		t.Errorf("expected probabilities in post, got:\n%s", post)
	}

	if !strings.Contains(post, "<code>") || !strings.Contains(post, "</code>") {
		t.Errorf("expected monospace <code> tags in post, got:\n%s", post)
	}
}
