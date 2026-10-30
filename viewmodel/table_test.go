package viewmodel

import (
	"testing"

	"github.com/Pastir/footballdata/analytics"
)

func TestBuildTop5TableViewModel(t *testing.T) {
	info := analytics.LeagueSeasonInfo{
		LeagueID:   8,
		LeagueName: "Premier League",
		SeasonName: "2024/2025",
	}

	// Создаем 20 команд
	rows := make([]analytics.Standing, 20)
	for i := 0; i < 20; i++ {
		rows[i] = analytics.Standing{
			Position: i + 1,
			TeamID:   int64(100 + i),
			TeamName: "Team",
			Played:   8,
			Won:      5,
			Drawn:    2,
			Lost:     1,
			GF:       15,
			GA:       5,
			GD:       10,
			Points:   20 - i,
			Form:     []string{"W", "W", "D", "W", "L"},
		}
	}

	vm := BuildTop5TableViewModel(info, "8-й тур", rows, "Тестовый инсайт")

	if len(vm.TopTeams) != 5 {
		t.Fatalf("expected 5 top teams, got %d", len(vm.TopTeams))
	}
	if vm.TopTeams[0].Position != 1 {
		t.Errorf("expected 1st team at pos 1, got %d", vm.TopTeams[0].Position)
	}
	if vm.TopTeams[4].Position != 5 {
		t.Errorf("expected 5th team at pos 5, got %d", vm.TopTeams[4].Position)
	}

	if len(vm.RelegationTeams) != 3 {
		t.Fatalf("expected 3 relegation teams, got %d", len(vm.RelegationTeams))
	}
	if vm.RelegationTeams[0].Position != 18 {
		t.Errorf("expected relegation start at 18, got %d", vm.RelegationTeams[0].Position)
	}
	if vm.RelegationTeams[2].Position != 20 {
		t.Errorf("expected relegation end at 20, got %d", vm.RelegationTeams[2].Position)
	}
	if vm.RoundName != "8-й тур" {
		t.Errorf("expected round '8-й тур', got %q", vm.RoundName)
	}
}
