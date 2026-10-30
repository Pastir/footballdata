package analytics

import (
	"testing"
)

func TestParseMatchScores(t *testing.T) {
	raw := []byte(`{"home": {"CURRENT": 3, "1ST_HALF": 1, "participant_id": 10}, "away": {"CURRENT": 2, "1ST_HALF": 0, "participant_id": 20}}`)
	scores := parseMatchScores(raw)

	if scores.HomeCurrent == nil || *scores.HomeCurrent != 3 {
		t.Errorf("expected HomeCurrent=3, got %v", scores.HomeCurrent)
	}
	if scores.AwayCurrent == nil || *scores.AwayCurrent != 2 {
		t.Errorf("expected AwayCurrent=2, got %v", scores.AwayCurrent)
	}
	if scores.HomeHT == nil || *scores.HomeHT != 1 {
		t.Errorf("expected HomeHT=1, got %v", scores.HomeHT)
	}
	if scores.AwayHT == nil || *scores.AwayHT != 0 {
		t.Errorf("expected AwayHT=0, got %v", scores.AwayHT)
	}
	if scores.FullScoreString() != "3:2" {
		t.Errorf("FullScoreString: got %q, want 3:2", scores.FullScoreString())
	}
	if scores.HTScoreString() != "(1:0)" {
		t.Errorf("HTScoreString: got %q, want (1:0)", scores.HTScoreString())
	}
}

func TestComputeRoundSummary(t *testing.T) {
	c3, c2, c1, c0 := 3, 2, 1, 0

	matches := []RoundResultMatch{
		// Матч 1: Хозяева победили, ОЗ, ТБ 2.5
		{
			FixtureID:    1,
			TeamHomeName: "Arsenal",
			TeamAwayName: "Chelsea",
			Scores: MatchScores{
				HomeCurrent: &c3,
				AwayCurrent: &c2,
				HomeHT:      &c1,
				AwayHT:      &c0,
			},
			Odds: MatchOdds{
				HomeOdds: "1.80",
				DrawOdds: "3.50",
				AwayOdds: "4.50",
			},
			Events: []MatchEvent{
				{Type: "yellowcard", Minute: 25, Player: "Rice"},
				{Type: "goal", Minute: 92, Player: "Saka"}, // Драма 90+
			},
		},
		// Матч 2: Ничья
		{
			FixtureID:    2,
			TeamHomeName: "Liverpool",
			TeamAwayName: "City",
			Scores: MatchScores{
				HomeCurrent: &c1,
				AwayCurrent: &c1,
			},
			Odds: MatchOdds{
				HomeOdds: "2.50",
				DrawOdds: "3.20",
				AwayOdds: "2.80",
			},
			Events: []MatchEvent{
				{Type: "yellowcard", Minute: 40},
				{Type: "redcard", Minute: 80},
			},
		},
		// Матч 3: Андердог победил в гостях (сенсация)
		{
			FixtureID:    3,
			TeamHomeName: "United",
			TeamAwayName: "Brentford",
			Scores: MatchScores{
				HomeCurrent: &c0,
				AwayCurrent: &c1,
			},
			Odds: MatchOdds{
				HomeOdds: "1.40", // Фаворит проиграл
				DrawOdds: "4.50",
				AwayOdds: "7.50",
			},
		},
	}

	summary := ComputeRoundSummary(matches)

	if summary.TotalMatches != 3 {
		t.Errorf("TotalMatches: got %d, want 3", summary.TotalMatches)
	}
	if summary.FinishedMatches != 3 {
		t.Errorf("FinishedMatches: got %d, want 3", summary.FinishedMatches)
	}
	// Всего голов: 5 + 2 + 1 = 8
	if summary.TotalGoals != 8 {
		t.Errorf("TotalGoals: got %d, want 8", summary.TotalGoals)
	}
	if summary.AvgGoals != 2.7 {
		t.Errorf("AvgGoals: got %.1f, want 2.7", summary.AvgGoals)
	}
	if summary.HomeWins != 1 || summary.Draws != 1 || summary.AwayWins != 1 {
		t.Errorf("1-X-2: got %d-%d-%d, want 1-1-1", summary.HomeWins, summary.Draws, summary.AwayWins)
	}
	if summary.BTTSCount != 2 {
		t.Errorf("BTTSCount: got %d, want 2", summary.BTTSCount)
	}
	if summary.Over25Count != 1 {
		t.Errorf("Over25Count: got %d, want 1", summary.Over25Count)
	}
	if summary.FavoritesWon != 1 {
		t.Errorf("FavoritesWon: got %d, want 1", summary.FavoritesWon)
	}
	if summary.FavoritesFailed != 2 {
		t.Errorf("FavoritesFailed: got %d, want 2", summary.FavoritesFailed)
	}
	if summary.TopUpsetOdds != 7.50 {
		t.Errorf("TopUpsetOdds: got %.2f, want 7.50", summary.TopUpsetOdds)
	}
	if summary.TotalYellowCards != 2 {
		t.Errorf("TotalYellowCards: got %d, want 2", summary.TotalYellowCards)
	}
	if summary.TotalRedCards != 1 {
		t.Errorf("TotalRedCards: got %d, want 1", summary.TotalRedCards)
	}
	if summary.LateDramaEvent == "" {
		t.Errorf("expected LateDramaEvent, got empty string")
	}
}

func TestRoundResultMatchXG(t *testing.T) {
	// Case 1: Both xG are present
	hXG, aXG := 2.45, 0.82
	m1 := RoundResultMatch{
		HomeXG: &hXG,
		AwayXG: &aXG,
	}
	if got := m1.XGString(); got != "2.45 – 0.82" {
		t.Errorf("expected '2.45 – 0.82', got %q", got)
	}

	// Case 2: One or both xG are missing (nil)
	m2 := RoundResultMatch{
		HomeXG: &hXG,
		AwayXG: nil,
	}
	if got := m2.XGString(); got != "" {
		t.Errorf("expected empty string for missing away xG, got %q", got)
	}

	m3 := RoundResultMatch{}
	if got := m3.XGString(); got != "" {
		t.Errorf("expected empty string when both xG are nil, got %q", got)
	}
}

