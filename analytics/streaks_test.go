package analytics

import (
	"testing"
	"time"
)

func TestCalculateSingleStreak(t *testing.T) {
	typesMap := map[string]StreakTypeMeta{
		StreakTypeTeamWin: {
			Code:      StreakTypeTeamWin,
			ShortName: "Победная серия",
			LongName:  "Победная серия во всех матчах",
			Icon:      "🔥",
		},
		StreakTypeTeamUnbeaten: {
			Code:      StreakTypeTeamUnbeaten,
			ShortName: "Беспроигрышная серия",
			LongName:  "Матчей без поражений",
			Icon:      "🛡️",
		},
		StreakTypeTeamScoring: {
			Code:      StreakTypeTeamScoring,
			ShortName: "Голевая серия",
			LongName:  "Матчей подряд с забитыми голами",
			Icon:      "⚽",
		},
		StreakTypeTeamCleanSheet: {
			Code:      StreakTypeTeamCleanSheet,
			ShortName: "Сухая серия",
			LongName:  "Матчей подряд на ноль",
			Icon:      "🧤",
		},
	}

	scopesMap := map[string]StreakScopeMeta{
		StreakScopeLeague: {
			Code:      StreakScopeLeague,
			ShortName: "Лига",
			LongName:  "Чемпионат",
		},
	}

	now := time.Now()
	// Серия матчей для команды ID 10:
	// 1: 01.10 - Win (2:0, clean sheet)
	// 2: 05.10 - Loss (0:1) -> прерывание
	// 3: 10.10 - Win (3:1)
	// 4: 15.10 - Win (2:0, clean sheet)
	// 5: 20.10 - Win (1:0, clean sheet)
	matches := []rawTeamMatchRow{
		{
			fixtureID:     101,
			leagueID:      8,
			seasonID:      2025,
			startingAt:    now.AddDate(0, 0, -20),
			homeID:        10,
			homeName:      "Арсенал",
			awayID:        20,
			awayName:      "Челси",
			teamIsHome:    true,
			teamGoals:     2,
			opponentGoals: 0,
			result:        "W",
		},
		{
			fixtureID:     102,
			leagueID:      8,
			seasonID:      2025,
			startingAt:    now.AddDate(0, 0, -15),
			homeID:        30,
			homeName:      "Ливерпуль",
			awayID:        10,
			awayName:      "Арсенал",
			teamIsHome:    false,
			teamGoals:     0,
			opponentGoals: 1,
			result:        "L",
		},
		{
			fixtureID:     103,
			leagueID:      8,
			seasonID:      2025,
			startingAt:    now.AddDate(0, 0, -10),
			homeID:        10,
			homeName:      "Арсенал",
			awayID:        40,
			awayName:      "Фулхэм",
			teamIsHome:    true,
			teamGoals:     3,
			opponentGoals: 1,
			result:        "W",
		},
		{
			fixtureID:     104,
			leagueID:      8,
			seasonID:      2025,
			startingAt:    now.AddDate(0, 0, -5),
			homeID:        50,
			homeName:      "Эвертон",
			awayID:        10,
			awayName:      "Арсенал",
			teamIsHome:    false,
			teamGoals:     2,
			opponentGoals: 0,
			result:        "W",
		},
		{
			fixtureID:     105,
			leagueID:      8,
			seasonID:      2025,
			startingAt:    now,
			homeID:        10,
			homeName:      "Арсенал",
			awayID:        60,
			awayName:      "Брентфорд",
			teamIsHome:    true,
			teamGoals:     1,
			opponentGoals: 0,
			result:        "W",
		},
	}

	t.Run("Winning streak calculation", func(t *testing.T) {
		rec := calculateSingleStreak(
			matches,
			10, "Арсенал", "logo.png",
			StreakTypeTeamWin, StreakScopeLeague,
			typesMap, scopesMap,
			func(m rawTeamMatchRow) bool {
				return m.result == "W"
			},
		)

		if rec == nil {
			t.Fatalf("expected winning streak, got nil")
		}

		if rec.Length != 3 {
			t.Errorf("expected length 3, got %d", rec.Length)
		}
		if !rec.IsActive {
			t.Errorf("expected is_active true")
		}
		if *rec.StartFixtureID != 103 {
			t.Errorf("expected start fixture 103, got %d", *rec.StartFixtureID)
		}
		if *rec.EndFixtureID != 105 {
			t.Errorf("expected end fixture 105, got %d", *rec.EndFixtureID)
		}
		if len(rec.FixturesHistory) != 3 {
			t.Errorf("expected 3 history items, got %d", len(rec.FixturesHistory))
		}

		// Проверка метрик: 3+2+1 = 6 голов забито, 1+0+0 = 1 пропущен
		scored, _ := rec.Metrics["goals_scored"].(int)
		if scored != 6 {
			t.Errorf("expected 6 goals scored, got %d", scored)
		}
		conceded, _ := rec.Metrics["goals_conceded"].(int)
		if conceded != 1 {
			t.Errorf("expected 1 goal conceded, got %d", conceded)
		}
		cleanSheets, _ := rec.Metrics["clean_sheets"].(int)
		if cleanSheets != 2 {
			t.Errorf("expected 2 clean sheets, got %d", cleanSheets)
		}
	})

	t.Run("Clean sheet streak calculation", func(t *testing.T) {
		rec := calculateSingleStreak(
			matches,
			10, "Арсенал", "logo.png",
			StreakTypeTeamCleanSheet, StreakScopeLeague,
			typesMap, scopesMap,
			func(m rawTeamMatchRow) bool {
				return m.opponentGoals == 0
			},
		)

		if rec == nil {
			t.Fatalf("expected clean sheet streak, got nil")
		}

		// Последние два матча: 104 (2:0) и 105 (1:0), матч 103 был 3:1 (пропущен гол)
		if rec.Length != 2 {
			t.Errorf("expected clean sheet streak length 2, got %d", rec.Length)
		}
		if *rec.StartFixtureID != 104 {
			t.Errorf("expected start fixture 104, got %d", *rec.StartFixtureID)
		}
	})

	t.Run("Scoring streak calculation", func(t *testing.T) {
		rec := calculateSingleStreak(
			matches,
			10, "Арсенал", "logo.png",
			StreakTypeTeamScoring, StreakScopeLeague,
			typesMap, scopesMap,
			func(m rawTeamMatchRow) bool {
				return m.teamGoals > 0
			},
		)

		if rec == nil {
			t.Fatalf("expected scoring streak, got nil")
		}

		// Матч 102 был 0:1 (не забили). Значит последние 3 матча забивали (103, 104, 105)
		if rec.Length != 3 {
			t.Errorf("expected scoring streak length 3, got %d", rec.Length)
		}
	})
}

func TestDetectTeamStreaksFromMatches(t *testing.T) {
	typesMap := map[string]StreakTypeMeta{
		StreakTypeTeamWin:        {Code: StreakTypeTeamWin, ShortName: "Победы"},
		StreakTypeTeamUnbeaten:   {Code: StreakTypeTeamUnbeaten, ShortName: "Без поражений"},
		StreakTypeTeamNoWin:      {Code: StreakTypeTeamNoWin, ShortName: "Без побед"},
		StreakTypeTeamScoring:    {Code: StreakTypeTeamScoring, ShortName: "Забитые"},
		StreakTypeTeamCleanSheet: {Code: StreakTypeTeamCleanSheet, ShortName: "Сухие"},
	}
	scopesMap := map[string]StreakScopeMeta{
		StreakScopeLeague: {Code: StreakScopeLeague, ShortName: "Лига"},
		StreakScopeAll:    {Code: StreakScopeAll, ShortName: "Все турниры"},
		StreakScopeHome:   {Code: StreakScopeHome, ShortName: "Дома"},
		StreakScopeAway:   {Code: StreakScopeAway, ShortName: "В гостях"},
	}

	now := time.Now()
	// 4 победы подряд дома!
	matches := []rawTeamMatchRow{
		{fixtureID: 1, homeID: 1, awayID: 2, teamIsHome: true, result: "W", teamGoals: 2, opponentGoals: 0, startingAt: now.AddDate(0, 0, -4)},
		{fixtureID: 2, homeID: 3, awayID: 1, teamIsHome: false, result: "L", teamGoals: 0, opponentGoals: 1, startingAt: now.AddDate(0, 0, -3)},
		{fixtureID: 3, homeID: 1, awayID: 4, teamIsHome: true, result: "W", teamGoals: 3, opponentGoals: 1, startingAt: now.AddDate(0, 0, -2)},
		{fixtureID: 4, homeID: 1, awayID: 5, teamIsHome: true, result: "W", teamGoals: 1, opponentGoals: 0, startingAt: now.AddDate(0, 0, -1)},
	}

	filter := StreaksFilter{
		Scope:     StreakScopeHome,
		MinLength: 3,
	}

	records := detectTeamStreaksFromMatches(matches, filter, typesMap, scopesMap)
	if len(records) == 0 {
		t.Fatalf("expected at least one home streak")
	}

	var foundHomeWin bool
	for _, r := range records {
		if r.StreakType == StreakTypeTeamWin && r.Scope == StreakScopeHome {
			foundHomeWin = true
			if r.Length != 3 {
				t.Errorf("expected 3 home wins, got %d", r.Length)
			}
		}
	}

	if !foundHomeWin {
		t.Errorf("home win streak was not found")
	}
}
