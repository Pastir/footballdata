package analytics

import (
	"strings"
	"testing"
)

func TestDetectLateDrama_Comeback(t *testing.T) {
	// Ipswich 2:1 Fulham
	// Iwobi 42' (0:1)
	// Ouattara 88' (1:1)
	// O'Shea 90+1' (2:1)
	hScore := 2
	aScore := 1
	extraMin := 1
	st := "FT"

	m := RoundResultMatch{
		FixtureID:    19722147,
		TeamHomeID:   1,
		TeamHomeName: "Ипсвич",
		TeamAwayID:   2,
		TeamAwayName: "Фулхэм",
		StateShort:   &st,
		Scores: MatchScores{
			HomeCurrent: &hScore,
			AwayCurrent: &aScore,
		},
		Events: []MatchEvent{
			{Minute: 42, TeamID: 2, Type: "goal", Result: "0-1", Player: "Iwobi"},
			{Minute: 88, TeamID: 1, Type: "goal", Result: "1-1", Player: "Ouattara"},
			{Minute: 90, ExtraMinute: &extraMin, TeamID: 1, Type: "goal", Result: "2-1", Player: "O'Shea"},
		},
	}

	info := LeagueSeasonInfo{LeagueID: 8, LeagueName: "Премьер-лига"}

	anomaly, ok := detectLateDrama(m, info)
	if !ok {
		t.Fatalf("expected comeback to be detected, got ok=false")
	}

	if anomaly.BadgeText != "Камбэк тура" {
		t.Errorf("expected BadgeText 'Камбэк тура', got '%s'", anomaly.BadgeText)
	}

	if anomaly.SeverityScore < 95.0 {
		t.Errorf("expected high severity for comeback, got %.1f", anomaly.SeverityScore)
	}

	if !strings.Contains(anomaly.FooterQuote, "2 гола за 3 минуты") {
		t.Errorf("expected quote to mention '2 гола за 3 минуты', got: %s", anomaly.FooterQuote)
	}

	if !strings.Contains(anomaly.TagText, "Камбэк за 3 минуты") {
		t.Errorf("expected tag to mention 'Камбэк за 3 минуты', got: %s", anomaly.TagText)
	}
}

func TestDetectLateDrama_ConsolationGoalIgnored(t *testing.T) {
	// Chelsea 4:3 Brighton
	// Chelsea led 4:2, Brighton scored at 90+6' (4:3) - should NOT trigger late drama
	hScore := 4
	aScore := 3
	extraMin := 6
	st := "FT"

	m := RoundResultMatch{
		FixtureID:    19722191,
		TeamHomeID:   10,
		TeamHomeName: "Челси",
		TeamAwayID:   20,
		TeamAwayName: "Брайтон",
		StateShort:   &st,
		Scores: MatchScores{
			HomeCurrent: &hScore,
			AwayCurrent: &aScore,
		},
		Events: []MatchEvent{
			{Minute: 15, TeamID: 10, Type: "goal", Result: "1-0", Player: "Palmer"},
			{Minute: 28, TeamID: 10, Type: "goal", Result: "2-0", Player: "Palmer"},
			{Minute: 31, TeamID: 20, Type: "goal", Result: "2-1", Player: "Rutter"},
			{Minute: 41, TeamID: 10, Type: "goal", Result: "3-1", Player: "Palmer"},
			{Minute: 65, TeamID: 10, Type: "goal", Result: "4-1", Player: "Palmer"},
			{Minute: 70, TeamID: 20, Type: "goal", Result: "4-2", Player: "Baleba"},
			{Minute: 90, ExtraMinute: &extraMin, TeamID: 20, Type: "goal", Result: "4-3", Player: "Ferguson"},
		},
	}

	info := LeagueSeasonInfo{LeagueID: 8, LeagueName: "Премьер-лига"}

	_, ok := detectLateDrama(m, info)
	if ok {
		t.Errorf("expected consolation goal at 90+6' to be ignored, but detectLateDrama returned true")
	}
}

func TestDetectLateDrama_LateEqualizer(t *testing.T) {
	// Bournemouth 1:1 Everton (Everton equalized at 90+1')
	hScore := 1
	aScore := 1
	extraMin := 1
	st := "FT"

	m := RoundResultMatch{
		FixtureID:    19722193,
		TeamHomeID:   30,
		TeamHomeName: "Борнмут",
		TeamAwayID:   40,
		TeamAwayName: "Эвертон",
		StateShort:   &st,
		Scores: MatchScores{
			HomeCurrent: &hScore,
			AwayCurrent: &aScore,
		},
		Events: []MatchEvent{
			{Minute: 50, TeamID: 30, Type: "goal", Result: "1-0", Player: "Semenyo"},
			{Minute: 90, ExtraMinute: &extraMin, TeamID: 40, Type: "goal", Result: "1-1", Player: "Beto"},
		},
	}

	info := LeagueSeasonInfo{LeagueID: 8, LeagueName: "Премьер-лига"}

	anomaly, ok := detectLateDrama(m, info)
	if !ok {
		t.Fatalf("expected late equalizer to be detected, got false")
	}

	if anomaly.BadgeText != "Спасение 90+" {
		t.Errorf("expected BadgeText 'Спасение 90+', got '%s'", anomaly.BadgeText)
	}
}

func TestPluralHelpers(t *testing.T) {
	if pluralMinutes(1) != "1 минуту" {
		t.Errorf("expected '1 минуту', got '%s'", pluralMinutes(1))
	}
	if pluralMinutes(2) != "2 минуты" {
		t.Errorf("expected '2 минуты', got '%s'", pluralMinutes(2))
	}
	if pluralMinutes(3) != "3 минуты" {
		t.Errorf("expected '3 минуты', got '%s'", pluralMinutes(3))
	}
	if pluralMinutes(5) != "5 минут" {
		t.Errorf("expected '5 минут', got '%s'", pluralMinutes(5))
	}

	if pluralGoals(1) != "1 гол" {
		t.Errorf("expected '1 гол', got '%s'", pluralGoals(1))
	}
	if pluralGoals(2) != "2 гола" {
		t.Errorf("expected '2 гола', got '%s'", pluralGoals(2))
	}
	if pluralGoals(5) != "5 голов" {
		t.Errorf("expected '5 голов', got '%s'", pluralGoals(5))
	}
}
