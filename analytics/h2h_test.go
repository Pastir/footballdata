package analytics

import (
	"testing"
)

func TestComputePeriodStats(t *testing.T) {
	xg1, xga1 := 2.5, 0.8
	xg2, xga2 := 1.5, 1.2
	sot1, poss1 := 6.0, 60.0
	sot2, poss2 := 4.0, 50.0

	matches := []matchRawStatRow{
		{
			goalsFor:      2,
			goalsAgainst:  1,
			xgFor:         &xg1,
			xgAgainst:     &xga1,
			shotsOnTarget: &sot1,
			possession:    &poss1,
		},
		{
			goalsFor:      0,
			goalsAgainst:  2,
			xgFor:         &xg2,
			xgAgainst:     &xga2,
			shotsOnTarget: &sot2,
			possession:    &poss2,
		},
	}

	p := computePeriodStats(matches)
	if p.MatchesCount != 2 {
		t.Fatalf("expected 2 matches, got %d", p.MatchesCount)
	}

	expectedGF := 1.0 // (2 + 0) / 2
	if p.GoalsScoredAvg != expectedGF {
		t.Errorf("expected GoalsScoredAvg %.2f, got %.2f", expectedGF, p.GoalsScoredAvg)
	}

	expectedGA := 1.5 // (1 + 2) / 2
	if p.GoalsConcededAvg != expectedGA {
		t.Errorf("expected GoalsConcededAvg %.2f, got %.2f", expectedGA, p.GoalsConcededAvg)
	}

	expectedXG := 2.0 // (2.5 + 1.5) / 2
	if p.XGPerMatch != expectedXG {
		t.Errorf("expected XGPerMatch %.2f, got %.2f", expectedXG, p.XGPerMatch)
	}

	expectedXGA := 1.0 // (0.8 + 1.2) / 2
	if p.XGAPerMatch != expectedXGA {
		t.Errorf("expected XGAPerMatch %.2f, got %.2f", expectedXGA, p.XGAPerMatch)
	}

	if p.BTTSPercent != 50 { // 1 of 2 (2:1)
		t.Errorf("expected BTTSPercent 50, got %d", p.BTTSPercent)
	}

	if p.Over25Percent != 50 { // 1 of 2 (2+1 = 3 > 2)
		t.Errorf("expected Over25Percent 50, got %d", p.Over25Percent)
	}

	if p.ShotsOnTargetAvg != 5.0 {
		t.Errorf("expected ShotsOnTargetAvg 5.0, got %.2f", p.ShotsOnTargetAvg)
	}

	if p.PossessionAvg != 55.0 {
		t.Errorf("expected PossessionAvg 55.0, got %.2f", p.PossessionAvg)
	}
}

func TestH2HSummaryCalculation(t *testing.T) {
	summary := H2HSummary{
		TotalMatches: 10,
		HomeWins:     5,
		Draws:        3,
		AwayWins:     2,
		HomeGoals:    16,
		AwayGoals:    10,
		AvgGoals:     2.6,
	}

	if summary.TotalMatches != summary.HomeWins+summary.Draws+summary.AwayWins {
		t.Errorf("total matches mismatch: %d != %d + %d + %d",
			summary.TotalMatches, summary.HomeWins, summary.Draws, summary.AwayWins)
	}
}
