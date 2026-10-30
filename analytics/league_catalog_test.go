package analytics

import (
	"testing"
)

func TestResolveLeagueMetadata(t *testing.T) {
	tests := []struct {
		name         string
		leagueID     int
		leagueName   string
		countryName  string
		subType      string
		leagueType   string
		wantCategory string
		wantName     string
		wantTop      bool
	}{
		{
			name:         "Premier League (Domestic & Top)",
			leagueID:     8,
			leagueName:   "Premier League",
			countryName:  "England",
			wantCategory: "domestic",
			wantName:     "АПЛ",
			wantTop:      true,
		},
		{
			name:         "La Liga (Domestic & Top)",
			leagueID:     564,
			leagueName:   "La Liga",
			countryName:  "Spain",
			wantCategory: "domestic",
			wantName:     "Ла Лига",
			wantTop:      true,
		},
		{
			name:         "Champions League (Eurocups & Top)",
			leagueID:     2,
			leagueName:   "UEFA Champions League",
			countryName:  "",
			wantCategory: "eurocups",
			wantName:     "Лига чемпионов",
			wantTop:      true,
		},
		{
			name:         "Championship (Domestic)",
			leagueID:     9,
			leagueName:   "Championship",
			countryName:  "England",
			wantCategory: "domestic",
			wantName:     "Чемпионшип",
			wantTop:      false,
		},
		{
			name:         "Generic International Cup (Eurocup)",
			leagueID:     9999,
			leagueName:   "UEFA Conference League",
			countryName:  "",
			subType:      "cup_international",
			leagueType:   "cup",
			wantCategory: "eurocups",
			wantName:     "UEFA Conference League",
			wantTop:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := resolveLeagueMetadata(
				tt.leagueID, tt.leagueName, "PL", "logo.png",
				1, tt.countryName, "flag.png",
				100, "2026/2027", 10, "10", 11, "11",
				tt.subType, tt.leagueType, true,
			)

			if item.Category != tt.wantCategory {
				t.Errorf("Category: got %q, want %q", item.Category, tt.wantCategory)
			}
			if item.DisplayName != tt.wantName {
				t.Errorf("DisplayName: got %q, want %q", item.DisplayName, tt.wantName)
			}
			if item.IsTop != tt.wantTop {
				t.Errorf("IsTop: got %v, want %v", item.IsTop, tt.wantTop)
			}
		})
	}
}

func TestCategoryOmissionWhenEmpty(t *testing.T) {
	// Симуляция: в активных лигах только турниры из категории "domestic",
	// а турниров из категории "eurocups" нет.
	items := []ActiveLeagueItem{
		{LeagueID: 8, Category: "domestic"},
		{LeagueID: 564, Category: "domestic"},
	}

	catCounts := make(map[string]int)
	for _, l := range items {
		catCounts[l.Category]++
	}

	allCats := []LeagueCategory{
		{Key: "domestic", Title: "Европейские чемпионаты", Icon: "🌍"},
		{Key: "eurocups", Title: "Еврокубки", Icon: "⭐"},
	}

	var activeCategories []LeagueCategory
	for _, c := range allCats {
		cnt := catCounts[c.Key]
		if cnt > 0 {
			c.Count = cnt
			activeCategories = append(activeCategories, c)
		}
	}

	if len(activeCategories) != 1 {
		t.Fatalf("expected exactly 1 active category, got %d", len(activeCategories))
	}
	if activeCategories[0].Key != "domestic" {
		t.Errorf("expected active category 'domestic', got %q", activeCategories[0].Key)
	}

	// Проверяем, что пустая категория "eurocups" строго отброшена
	for _, c := range activeCategories {
		if c.Key == "eurocups" {
			t.Errorf("empty category 'eurocups' must not be present in activeCategories")
		}
	}
}
