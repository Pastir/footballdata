package viewmodel_test

import (
	"testing"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/viewmodel"
)

func TestFixtureMatchItem_HasOdds(t *testing.T) {
	tests := []struct {
		name     string
		item     viewmodel.FixtureMatchItem
		expected bool
	}{
		{
			name: "All odds valid",
			item: viewmodel.FixtureMatchItem{
				OddsHome: "1.85",
				OddsDraw: "3.60",
				OddsAway: "4.20",
			},
			expected: true,
		},
		{
			name: "Empty odds",
			item: viewmodel.FixtureMatchItem{
				OddsHome: "",
				OddsDraw: "",
				OddsAway: "",
			},
			expected: false,
		},
		{
			name: "Missing Home odds",
			item: viewmodel.FixtureMatchItem{
				OddsHome: "",
				OddsDraw: "3.60",
				OddsAway: "4.20",
			},
			expected: false,
		},
		{
			name: "Missing Away odds",
			item: viewmodel.FixtureMatchItem{
				OddsHome: "1.85",
				OddsDraw: "3.60",
				OddsAway: "",
			},
			expected: false,
		},
		{
			name: "Zero odds",
			item: viewmodel.FixtureMatchItem{
				OddsHome: "0",
				OddsDraw: "0",
				OddsAway: "0",
			},
			expected: false,
		},
		{
			name: "Dash odds placeholder",
			item: viewmodel.FixtureMatchItem{
				OddsHome: "-",
				OddsDraw: "-",
				OddsAway: "-",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.HasOdds(); got != tt.expected {
				t.Errorf("HasOdds() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestMatchScoreItem_HasOdds(t *testing.T) {
	tests := []struct {
		name     string
		item     viewmodel.MatchScoreItem
		expected bool
	}{
		{
			name: "Closing odds valid",
			item: viewmodel.MatchScoreItem{
				ClosingOddsHome: "2.10",
				ClosingOddsDraw: "3.40",
				ClosingOddsAway: "3.30",
			},
			expected: true,
		},
		{
			name: "Empty closing odds",
			item: viewmodel.MatchScoreItem{
				ClosingOddsHome: "",
				ClosingOddsAway: "",
			},
			expected: false,
		},
		{
			name: "Zero closing odds",
			item: viewmodel.MatchScoreItem{
				ClosingOddsHome: "0.00",
				ClosingOddsAway: "0.00",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.HasOdds(); got != tt.expected {
				t.Errorf("HasOdds() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestBuildRoundFixturesViewModel_FeaturedMatch(t *testing.T) {
	info := analytics.LeagueSeasonInfo{
		LeagueID:   8,
		LeagueName: "Premier League",
		SeasonName: "2026/2027",
	}

	pos1 := 1
	pos2 := 2
	pos7 := 7
	pos8 := 8

	matches := []analytics.NextRoundMatch{
		{
			FixtureID:    101,
			RoundName:    "8",
			TeamHomeName: "Arsenal",
			TeamAwayName: "Chelsea",
			PositionHome: &pos1,
			PositionAway: &pos2,
			OddsHome:     "2.00",
			OddsDraw:     "3.50",
			OddsAway:     "3.80",
		},
		{
			FixtureID:    102,
			RoundName:    "8",
			TeamHomeName: "Everton",
			TeamAwayName: "Fulham",
			PositionHome: &pos7,
			PositionAway: &pos8,
			OddsHome:     "2.50",
			OddsDraw:     "3.20",
			OddsAway:     "2.90",
		},
	}

	// 1. Without explicit featured ID, matches[0] (Arsenal vs Chelsea, sum=3) should be chosen over sum=15
	vmDefault := viewmodel.BuildRoundFixturesViewModel(info, matches, "MSK")
	if vmDefault.FeaturedMatch == nil {
		t.Fatal("expected FeaturedMatch to be non-nil")
	}
	if vmDefault.FeaturedMatch.ID != 101 {
		t.Errorf("expected default FeaturedMatch to be 101, got %d", vmDefault.FeaturedMatch.ID)
	}
	if vmDefault.FeaturedMatch.ProbHome == "" || vmDefault.FeaturedMatch.ProbDraw == "" || vmDefault.FeaturedMatch.ProbAway == "" {
		t.Errorf("expected probabilities to be populated, got Home=%s Draw=%s Away=%s",
			vmDefault.FeaturedMatch.ProbHome, vmDefault.FeaturedMatch.ProbDraw, vmDefault.FeaturedMatch.ProbAway)
	}

	// 2. With explicit featured ID (102), match 102 should be selected despite worse table positions
	vmExplicit := viewmodel.BuildRoundFixturesViewModel(info, matches, "MSK", 102)
	if vmExplicit.FeaturedMatch == nil {
		t.Fatal("expected FeaturedMatch to be non-nil")
	}
	if vmExplicit.FeaturedMatch.ID != 102 {
		t.Errorf("expected explicit FeaturedMatch to be 102, got %d", vmExplicit.FeaturedMatch.ID)
	}
}
