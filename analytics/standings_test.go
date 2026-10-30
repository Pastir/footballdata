package analytics

import (
	"testing"
)

func TestZoneFromDeveloperName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"CHAMPIONS_LEAGUE", "ucl"},
		{"UEFA_CHAMPIONS_LEAGUE", "ucl"},
		{"EUROPA_LEAGUE", "uel"},
		{"UEFA_CONFERENCE_LEAGUE", "uecl"},
		{"RELEGATION", "relegation"},
		{"POSSIBLE_RELEGATION_PLAY_OFF", "relegation"},
		{"", ""},
		{"SOME_OTHER_RULE", ""},
	}
	for _, c := range cases {
		got := ZoneFromDeveloperName(c.input)
		if got != c.expected {
			t.Errorf("ZoneFromDeveloperName(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestGetTrendLogic(t *testing.T) {
	// Trend = prev_position - curr_position
	// Arsenal: prev=2, curr=1 → +1
	// ManCity: prev=1, curr=2 → -1
	// Brighton: prev=10, curr=4 → +6
	cases := []struct {
		prev, curr int
		expected   int
	}{
		{2, 1, 1},
		{1, 2, -1},
		{10, 4, 6},
		{5, 5, 0},
	}
	for _, c := range cases {
		got := c.prev - c.curr
		if got != c.expected {
			t.Errorf("trend prev=%d curr=%d: got %d, want %d", c.prev, c.curr, got, c.expected)
		}
	}
}

func TestGetKeyStats(t *testing.T) {
	rows := []Standing{
		{ID: 1, Position: 1, TeamName: "Arsenal", Points: 12, GF: 14, GA: 2, GD: 12},
		{ID: 2, Position: 2, TeamName: "Chelsea", Points: 9, GF: 10, GA: 4, GD: 6},
		{ID: 3, Position: 3, TeamName: "Liverpool", Points: 7, GF: 6, GA: 5, GD: 1},
	}
	details := map[int64]StandingDetails{
		1: {Played: 4, Won: 4, Drawn: 0, Lost: 0, GF: 14, GA: 2, GD: 12},
		2: {Played: 4, Won: 3, Drawn: 0, Lost: 1, GF: 10, GA: 4, GD: 6},
		3: {Played: 4, Won: 2, Drawn: 1, Lost: 1, GF: 6, GA: 5, GD: 1},
	}

	// 1. Тест с передачей details (обратная совместимость)
	statsWithDetails := GetKeyStats(rows, details)
	if len(statsWithDetails) != 3 {
		t.Fatalf("expected 3 stat cards, got %d", len(statsWithDetails))
	}
	if statsWithDetails[0].Label != "Most Goals" || statsWithDetails[0].Team != "Arsenal" {
		t.Errorf("Most Goals: got %+v, want Arsenal", statsWithDetails[0])
	}
	if statsWithDetails[1].Label != "Best Defence" || statsWithDetails[1].Team != "Arsenal" {
		t.Errorf("Best Defence: got %+v, want Arsenal", statsWithDetails[1])
	}
	if statsWithDetails[2].Label != "Best Goal Diff" || statsWithDetails[2].Team != "Arsenal" {
		t.Errorf("Best GD: got %+v, want Arsenal", statsWithDetails[2])
	}

	// 2. Тест прямого использования полей Standing (без details)
	statsDirect := GetKeyStats(rows)
	if len(statsDirect) != 3 {
		t.Fatalf("expected 3 stat cards directly from rows, got %d", len(statsDirect))
	}
	if statsDirect[0].Team != "Arsenal" || statsDirect[1].Team != "Arsenal" || statsDirect[2].Team != "Arsenal" {
		t.Errorf("expected Arsenal for all 3 cards directly from Standing, got %+v", statsDirect)
	}
}

func TestBuildInsight(t *testing.T) {
	rows := []Standing{
		{ID: 1, Position: 1, TeamName: "Arsenal", Points: 12, GF: 14, GA: 2},
		{ID: 2, Position: 2, TeamName: "Chelsea", Points: 9, GF: 7, GA: 5},
	}
	details := map[int64]StandingDetails{
		1: {GF: 14, GA: 2},
		2: {GF: 7, GA: 5},
	}

	// С details
	insight := BuildInsight(rows, details)
	if insight == "" {
		t.Error("BuildInsight with details returned empty string")
	}

	// Напрямую из Standing
	insightDirect := BuildInsight(rows)
	if insightDirect == "" {
		t.Error("BuildInsight directly from Standing returned empty string")
	}
	if insightDirect != insight {
		t.Errorf("BuildInsight mismatch: %q vs %q", insightDirect, insight)
	}
	t.Logf("Insight: %s", insightDirect)
}

func TestFormPoints(t *testing.T) {
	cases := []struct {
		form           []string
		expectedPoints int
		expectedGames  int
	}{
		{[]string{"W", "W", "D", "L", "W"}, 10, 5},
		{[]string{}, 0, 0},
		{[]string{"L", "L", "L"}, 0, 3},
		{[]string{"D", "D"}, 2, 2},
	}
	for _, c := range cases {
		pts, games := formPoints(c.form)
		if pts != c.expectedPoints || games != c.expectedGames {
			t.Errorf("formPoints(%v) = (%d,%d), want (%d,%d)",
				c.form, pts, games, c.expectedPoints, c.expectedGames)
		}
	}
}

func TestRankMatchesOfRound(t *testing.T) {
	standings := []Standing{
		{TeamID: 1, Position: 1, Points: 30, Played: 10},
		{TeamID: 2, Position: 2, Points: 28, Played: 10},
		{TeamID: 3, Position: 10, Points: 12, Played: 10},
		{TeamID: 4, Position: 11, Points: 11, Played: 10},
	}

	matches := []MatchCandidate{
		// Матч топ-команд
		{
			FixtureID: 1, TeamHomeID: 1, TeamAwayID: 2,
			TeamHomeName: "City", TeamAwayName: "United",
			PositionHome: 1, PositionAway: 2,
			PointsHome: 30, PointsAway: 28,
			PlayedHome: 10, PlayedAway: 10,
			FormHome: []string{"W", "W", "W", "W", "W"},
			FormAway: []string{"W", "W", "W", "W", "D"},
		},
		// Матч середняков
		{
			FixtureID: 2, TeamHomeID: 3, TeamAwayID: 4,
			TeamHomeName: "Brighton", TeamAwayName: "Wolves",
			PositionHome: 10, PositionAway: 11,
			PointsHome: 12, PointsAway: 11,
			PlayedHome: 10, PlayedAway: 10,
			FormHome: []string{"L", "D", "L", "D", "L"},
			FormAway: []string{"L", "L", "D", "L", "L"},
		},
	}

	ctx := RoundContext{
		TeamCount:        20,
		TotalTeamMatches: 38,
		Standings:        standings,
	}

	results, err := RankMatchesOfRound(matches, ctx)
	if err != nil {
		t.Fatalf("RankMatchesOfRound error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Матч #1 vs #2 должен быть первым
	if results[0].FixtureID != 1 {
		t.Errorf("expected fixtureID=1 as top match, got %d", results[0].FixtureID)
	}
	t.Logf("Match 1: %s vs %s final=%.2f", results[0].TeamHome, results[0].TeamAway, results[0].FinalScore)
	t.Logf("Match 2: %s vs %s final=%.2f", results[1].TeamHome, results[1].TeamAway, results[1].FinalScore)
}

func TestBuildMatchCandidates(t *testing.T) {
	posH, posA := 1, 2
	ptsH, ptsA := 30, 28
	matchH, matchA := 10, 10

	matches := []NextRoundMatch{
		{
			FixtureID:          100,
			TeamHomeID:         10,
			TeamAwayID:         20,
			TeamHomeName:       "City",
			TeamAwayName:       "United",
			FormHome:           []string{"W", "W", "D"},
			FormAway:           []string{"D", "W", "W"},
			PositionHome:       &posH,
			PositionAway:       &posA,
			PointsHome:         &ptsH,
			PointsAway:         &ptsA,
			OverallMatchesHome: &matchH,
			MatchesAway:        &matchA,
		},
	}

	candidates := BuildMatchCandidates(matches)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	c := candidates[0]
	if c.FixtureID != 100 {
		t.Errorf("FixtureID: got %d, want 100", c.FixtureID)
	}
	if len(c.FormHome) != 3 {
		t.Errorf("FormHome len: got %d, want 3", len(c.FormHome))
	}
	if c.PlayedHome != 10 {
		t.Errorf("PlayedHome: got %d, want 10", c.PlayedHome)
	}
}
