package renderer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/renderer"
	"github.com/Pastir/footballdata/viewmodel"
)

func TestRenderStandingsPNG(t *testing.T) {
	templatePath := filepath.Join("..", "template", "football_table_template.html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("template not found: skipping test")
	}

	teams := []viewmodel.TeamRow{
		{Position: 1, Name: "Arsenal", Played: 28, Won: 20, Drawn: 4, Lost: 4, GF: 70, GA: 24, GD: 46, Points: 64, Form: []string{"W", "W", "W", "W", "W"}, Trend: 1, Zone: "ucl"},
		{Position: 2, Name: "Liverpool", Played: 28, Won: 19, Drawn: 7, Lost: 2, GF: 65, GA: 26, GD: 39, Points: 64, Form: []string{"W", "W", "D", "W", "W"}, Trend: -1, Zone: "ucl"},
		{Position: 3, Name: "Manchester City", Played: 28, Won: 19, Drawn: 6, Lost: 3, GF: 63, GA: 28, GD: 35, Points: 63, Form: []string{"W", "D", "W", "W", "D"}, Trend: 0, Zone: "ucl"},
		{Position: 4, Name: "Aston Villa", Played: 29, Won: 17, Drawn: 5, Lost: 7, GF: 60, GA: 42, GD: 18, Points: 56, Form: []string{"L", "W", "W", "W", "D"}, Trend: 0, Zone: "ucl"},
		{Position: 5, Name: "Tottenham Hotspur", Played: 28, Won: 16, Drawn: 5, Lost: 7, GF: 59, GA: 42, GD: 17, Points: 53, Form: []string{"W", "L", "W", "L", "W"}, Trend: 0, Zone: "uel"},
		{Position: 6, Name: "Manchester United", Played: 28, Won: 15, Drawn: 2, Lost: 11, GF: 39, GA: 39, GD: 0, Points: 47, Form: []string{"W", "W", "L", "L", "W"}, Trend: 0, Zone: "uecl"},
		{Position: 7, Name: "West Ham United", Played: 29, Won: 12, Drawn: 8, Lost: 9, GF: 46, GA: 50, GD: -4, Points: 44, Form: []string{"L", "W", "W", "D", "D"}, Trend: 1, Zone: ""},
		{Position: 8, Name: "Brighton & Hove", Played: 28, Won: 11, Drawn: 9, Lost: 8, GF: 50, GA: 44, GD: 6, Points: 42, Form: []string{"D", "L", "W", "L", "W"}, Trend: -1, Zone: ""},
		{Position: 9, Name: "Wolverhampton", Played: 28, Won: 12, Drawn: 5, Lost: 11, GF: 42, GA: 44, GD: -2, Points: 41, Form: []string{"W", "L", "W", "W", "L"}, Trend: 0, Zone: ""},
		{Position: 10, Name: "Newcastle United", Played: 28, Won: 12, Drawn: 4, Lost: 12, GF: 59, GA: 48, GD: 11, Points: 40, Form: []string{"D", "W", "L", "W", "L"}, Trend: 0, Zone: ""},
		{Position: 11, Name: "Chelsea", Played: 27, Won: 11, Drawn: 6, Lost: 10, GF: 47, GA: 45, GD: 2, Points: 39, Form: []string{"W", "D", "D", "D", "W"}, Trend: 0, Zone: ""},
		{Position: 12, Name: "Fulham", Played: 29, Won: 11, Drawn: 5, Lost: 13, GF: 43, GA: 44, GD: -1, Points: 38, Form: []string{"L", "W", "W", "L", "W"}, Trend: 0, Zone: ""},
		{Position: 13, Name: "AFC Bournemouth", Played: 28, Won: 9, Drawn: 8, Lost: 11, GF: 41, GA: 49, GD: -8, Points: 35, Form: []string{"L", "L", "W", "D", "W"}, Trend: 0, Zone: ""},
		{Position: 14, Name: "Crystal Palace", Played: 28, Won: 7, Drawn: 8, Lost: 13, GF: 33, GA: 48, GD: -15, Points: 29, Form: []string{"L", "D", "W", "L", "D"}, Trend: 1, Zone: ""},
		{Position: 15, Name: "Brentford", Played: 29, Won: 7, Drawn: 5, Lost: 17, GF: 41, GA: 54, GD: -13, Points: 26, Form: []string{"L", "L", "L", "D", "L"}, Trend: -1, Zone: ""},
		{Position: 16, Name: "Everton", Played: 28, Won: 8, Drawn: 7, Lost: 13, GF: 29, GA: 39, GD: -10, Points: 25, Form: []string{"D", "D", "D", "L", "L"}, Trend: 0, Zone: ""},
		{Position: 17, Name: "Nottingham Forest", Played: 29, Won: 6, Drawn: 7, Lost: 16, GF: 35, GA: 51, GD: -16, Points: 21, Form: []string{"L", "W", "L", "L", "D"}, Trend: 0, Zone: ""},
		{Position: 18, Name: "Luton Town", Played: 29, Won: 5, Drawn: 7, Lost: 17, GF: 42, GA: 60, GD: -18, Points: 22, Form: []string{"L", "L", "L", "L", "D"}, Trend: 0, Zone: "relegation"},
		{Position: 19, Name: "Burnley", Played: 29, Won: 4, Drawn: 5, Lost: 20, GF: 29, GA: 63, GD: -34, Points: 17, Form: []string{"L", "L", "L", "D", "W"}, Trend: 0, Zone: "relegation"},
		{Position: 20, Name: "Sheffield United", Played: 28, Won: 3, Drawn: 5, Lost: 20, GF: 24, GA: 74, GD: -50, Points: 14, Form: []string{"L", "L", "L", "L", "D"}, Trend: 0, Zone: "relegation"},
	}

	stats := []analytics.StatCard{
		{Label: "Top Attack", Team: "Arsenal", Value: "70 GF"},
		{Label: "Best Defense", Team: "Arsenal", Value: "24 GA"},
		{Label: "Best Goal Diff", Team: "Arsenal", Value: "+46 GD"},
		{Label: "Best Form", Team: "Arsenal", Value: "15 PTS"},
	}

	vm := viewmodel.TableViewModel{
		LeagueID:   8,
		LeagueName: "Premier League",
		Season:     "2023/2024",
		Teams:      teams,
		Stats:      stats,
		UpdatedAt:  "18 Mar 2026, 17:00 UTC",
	}

	tmpOut := filepath.Join("..", "template", "standings_preview.png")
	err := renderer.RenderStandingsPNG(vm, templatePath, tmpOut)
	if err != nil {
		t.Fatalf("RenderStandingsPNG failed: %v", err)
	}

	info, err := os.Stat(tmpOut)
	if err != nil {
		t.Fatalf("failed to stat output file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("output file is empty")
	}
}

func TestRenderTop5StandingsPNG(t *testing.T) {
	templatePath := filepath.Join("..", "template", "top5_standings_template.html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("template not found: skipping test")
	}

	topTeams := []viewmodel.TeamRow{
		{Position: 1, Name: "Arsenal", Played: 8, Won: 6, Drawn: 1, Lost: 1, GF: 18, GA: 8, GD: 10, Points: 19, Form: []string{"W", "W", "W", "D", "W"}},
		{Position: 2, Name: "Manchester City", Played: 8, Won: 5, Drawn: 3, Lost: 0, GF: 19, GA: 5, GD: 14, Points: 18, Form: []string{"W", "W", "D", "D", "W"}},
		{Position: 3, Name: "Liverpool", Played: 8, Won: 5, Drawn: 3, Lost: 0, GF: 16, GA: 4, GD: 12, Points: 18, Form: []string{"W", "D", "W", "W", "D"}},
		{Position: 4, Name: "Chelsea", Played: 8, Won: 5, Drawn: 2, Lost: 1, GF: 15, GA: 6, GD: 9, Points: 17, Form: []string{"W", "W", "L", "W", "W"}},
		{Position: 5, Name: "Aston Villa", Played: 8, Won: 5, Drawn: 1, Lost: 2, GF: 14, GA: 7, GD: 7, Points: 16, Form: []string{"W", "W", "D", "L", "W"}},
	}

	relTeams := []viewmodel.TeamRow{
		{Position: 18, Name: "Wolverhampton", Played: 8, GD: -8, Points: 5},
		{Position: 19, Name: "Everton", Played: 8, GD: -10, Points: 4},
		{Position: 20, Name: "Southampton", Played: 8, GD: -14, Points: 2},
	}

	vm := viewmodel.Top5TableViewModel{
		LeagueID:        8,
		LeagueName:      "Premier League",
		Season:          "2024/2025",
		RoundName:       "8-й тур",
		TopTeams:        topTeams,
		RelegationTeams: relTeams,
		UpdatedAt:       "22 Oct 2026, 20:00 UTC",
	}

	tmpOut := filepath.Join("..", "template", "top5_preview.png")
	defer os.Remove(tmpOut)

	err := renderer.RenderTop5StandingsPNG(vm, templatePath, tmpOut)
	if err != nil {
		t.Fatalf("RenderTop5StandingsPNG failed: %v", err)
	}

	info, err := os.Stat(tmpOut)
	if err != nil {
		t.Fatalf("failed to stat top5_preview.png: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("top5_preview.png is empty")
	}
}

