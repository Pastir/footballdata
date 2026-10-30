package analytics

import (
	"testing"
)

func TestPlayerLeaderItem(t *testing.T) {
	item := PlayerLeaderItem{
		Rank:         1,
		PlayerID:     28931574,
		PlayerName:   "João Pedro",
		PlayerImage:  "https://cdn.sportmonks.com/images/soccer/players/14/28931574.png",
		TeamID:       18,
		TeamName:     "Челси",
		TeamCode:     "CHE",
		TeamLogo:     "https://cdn.sportmonks.com/images/soccer/teams/18/18.png",
		Matches:      3,
		PrimaryVal:   4,
		SecondaryVal: 0,
		MetricLabel:  "голов",
	}

	if item.Rank != 1 {
		t.Errorf("expected Rank=1, got %d", item.Rank)
	}
	if item.PrimaryVal != 4 {
		t.Errorf("expected PrimaryVal=4, got %d", item.PrimaryVal)
	}
	if item.TeamCode != "CHE" {
		t.Errorf("expected TeamCode=CHE, got %s", item.TeamCode)
	}
}

func TestSeasonLeadersDataStructures(t *testing.T) {
	data := SeasonLeadersData{
		LeagueID:   8,
		LeagueName: "АПЛ",
		SeasonID:   28083,
		SeasonName: "2026/2027",
		Locale:     "ru",
		TopScorers: []PlayerLeaderItem{
			{Rank: 1, PlayerName: "João Pedro", PrimaryVal: 4, TeamName: "Челси"},
			{Rank: 2, PlayerName: "Bruno Fernandes", PrimaryVal: 3, TeamName: "Манчестер Юнайтед"},
		},
		TopAssists: []PlayerLeaderItem{
			{Rank: 1, PlayerName: "Cole Palmer", PrimaryVal: 2, TeamName: "Челси"},
		},
		CleanSheets: []PlayerLeaderItem{
			{Rank: 1, PlayerName: "B. Verbruggen", PrimaryVal: 5, TeamName: "Брайтон"},
		},
	}

	if len(data.TopScorers) != 2 {
		t.Errorf("expected 2 scorers, got %d", len(data.TopScorers))
	}
	if data.TopScorers[0].PrimaryVal < data.TopScorers[1].PrimaryVal {
		t.Errorf("expected scorers ordered by goals descending")
	}
	if len(data.CleanSheets) != 1 || data.CleanSheets[0].PrimaryVal != 5 {
		t.Errorf("expected CleanSheets[0].PrimaryVal=5, got %v", data.CleanSheets)
	}
}
