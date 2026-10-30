package analytics

import (
	"testing"
	"time"
)

func TestBuildSmartExpress_ValidCandidates(t *testing.T) {
	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)

	matches := []ExpressCandidateMatch{
		{
			FixtureID:       101,
			HomeTeamName:    "Arsenal",
			AwayTeamName:    "Everton",
			StartingAt:      now,
			OddsHome:        1.45,
			OddsDraw:        4.50,
			OddsAway:        7.00,
			HomeFormRank:    2,
			AwayFormRank:    16,
			HomeRecentGoals: 2.1,
			AwayRecentGoals: 0.9,
		},
		{
			FixtureID:       102,
			HomeTeamName:    "Aston Villa",
			AwayTeamName:    "Manchester City",
			StartingAt:      now.Add(2 * time.Hour),
			OddsHome:        4.20,
			OddsDraw:        3.90,
			OddsAway:        1.70,
			HomeFormRank:    7,
			AwayFormRank:    1,
			HomeRecentGoals: 1.8,
			AwayRecentGoals: 2.4,
		},
		{
			FixtureID:       103,
			HomeTeamName:    "Chelsea",
			AwayTeamName:    "Tottenham Hotspur",
			StartingAt:      now.Add(4 * time.Hour),
			OddsHome:        2.30,
			OddsDraw:        3.50,
			OddsAway:        2.80,
			HomeFormRank:    4,
			AwayFormRank:    5,
			HomeRecentGoals: 1.9,
			AwayRecentGoals: 1.7,
		},
		{
			FixtureID:       104,
			HomeTeamName:    "Liverpool",
			AwayTeamName:    "Brighton",
			StartingAt:      now.Add(24 * time.Hour),
			OddsHome:        1.55,
			OddsDraw:        4.20,
			OddsAway:        5.50,
			HomeFormRank:    3,
			AwayFormRank:    8,
			HomeRecentGoals: 2.0,
			AwayRecentGoals: 1.5,
		},
	}

	plan, err := BuildSmartExpress(matches)
	if err != nil {
		t.Fatalf("unexpected error building express: %v", err)
	}

	if len(plan.Selections) != 3 {
		t.Fatalf("expected 3 selections, got %d", len(plan.Selections))
	}

	if plan.TotalOdds < 3.0 || plan.TotalOdds > 6.0 {
		t.Errorf("total odds out of realistic range: %.2f", plan.TotalOdds)
	}

	// Проверяем, что все 3 матча разные
	seen := make(map[int]bool)
	for _, sel := range plan.Selections {
		if seen[sel.Match.FixtureID] {
			t.Errorf("duplicate fixture in express: %d", sel.Match.FixtureID)
		}
		seen[sel.Match.FixtureID] = true
	}
}

func TestBuildSmartExpress_InsufficientMatches(t *testing.T) {
	matches := []ExpressCandidateMatch{
		{FixtureID: 1, OddsHome: 1.5, OddsDraw: 3.5, OddsAway: 6.0},
		{FixtureID: 2, OddsHome: 1.8, OddsDraw: 3.4, OddsAway: 4.5},
	}

	_, err := BuildSmartExpress(matches)
	if err == nil {
		t.Errorf("expected error for less than 3 matches, got nil")
	}
}
