package copywriter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Pastir/footballdata/copywriter"
	"github.com/Pastir/footballdata/viewmodel"
)

func TestBuildRoundResultsFactPack(t *testing.T) {
	vm := viewmodel.RoundResultsViewModel{
		LeagueName: "Premier League",
		Season:     "2023/2024",
		RoundTitle: "ТУР 28",
		Stats: viewmodel.RoundStatsSummary{
			TotalMatches: 10,
			TotalGoals:   28,
			AvgGoals:     "2.8",
			TopMatch:     "Man City 2:2 Liverpool",
		},
		LeftCol: []viewmodel.MatchScoreItem{
			{
				HomeTeamName: "Arsenal",
				AwayTeamName: "Chelsea",
				HomeScore:    3,
				AwayScore:    1,
				XGHome:       "2.10",
				XGAway:       "0.90",
			},
		},
	}

	pack := copywriter.BuildRoundResultsFactPack(vm)
	if pack.League != "Premier League" {
		t.Errorf("expected league Premier League, got %s", pack.League)
	}
	if pack.Round != "ТУР 28" {
		t.Errorf("expected round ТУР 28, got %s", pack.Round)
	}
	if len(pack.KeyMatches) != 1 {
		t.Errorf("expected 1 key match, got %d", len(pack.KeyMatches))
	}
	if pack.KeyMatches[0].Score != "3:1" {
		t.Errorf("expected score 3:1, got %s", pack.KeyMatches[0].Score)
	}
}

func TestGenerateFallbackPost(t *testing.T) {
	pack := copywriter.FactPack{
		PostType: "round_results",
		League:   "Premier League",
		Round:    "ТУР 28",
		Stats: map[string]interface{}{
			"total_goals": 28,
			"avg_goals":   "2.8",
			"red_cards":   2,
		},
		KeyMatches: []copywriter.KeyMatchFact{
			{
				HomeTeam: "Arsenal",
				AwayTeam: "Chelsea",
				Score:    "3:1",
				XG:       "2.10 - 0.90",
			},
		},
	}

	post := copywriter.GenerateFallbackPost(pack)
	if !strings.Contains(post, "Premier League") {
		t.Errorf("expected post to contain Premier League")
	}
	if !strings.Contains(post, "3:1") {
		t.Errorf("expected post to contain score 3:1")
	}
	if err := copywriter.ValidatePost(post, true); err != nil {
		t.Errorf("fallback post should be valid: %v", err)
	}
}

func TestGenerateFallbackPost_RoundFixtures(t *testing.T) {
	pack := copywriter.FactPack{
		PostType: "round_fixtures",
		League:   "Premier League",
		Round:    "ТУР 29",
		Fixtures: []copywriter.FixtureFact{
			{
				HomeTeam: "Liverpool",
				AwayTeam: "Manchester City",
				Time:     "15:30",
				IsDerby:  true,
				OddsHome: "2.75",
				OddsDraw: "3.60",
				OddsAway: "2.30",
			},
			{
				HomeTeam: "Arsenal",
				AwayTeam: "Chelsea",
				Time:     "18:00",
				IsDerby:  false,
				OddsHome: "1.95",
				OddsDraw: "3.40",
				OddsAway: "3.90",
			},
		},
	}

	post := copywriter.GenerateFallbackPost(pack)
	t.Logf("\n%s", post)
	expectedDerbyLine := "• 15:30 <b>Liverpool — Manchester City</b> ⚔\n            2.75 · 3.60 · 2.30\n"
	expectedNonDerbyLine := "• 18:00 <b>Arsenal — Chelsea</b>\n            1.95 · 3.40 · 3.90\n"

	if !strings.Contains(post, expectedDerbyLine) {
		t.Errorf("expected post to contain:\n%q\ngot:\n%s", expectedDerbyLine, post)
	}
	if !strings.Contains(post, expectedNonDerbyLine) {
		t.Errorf("expected post to contain:\n%q\ngot:\n%s", expectedNonDerbyLine, post)
	}
	if err := copywriter.ValidatePost(post, true); err != nil {
		t.Errorf("fallback post should be valid: %v", err)
	}
}

func TestValidatePost(t *testing.T) {
	if err := copywriter.ValidatePost("", true); err == nil {
		t.Errorf("expected error for empty post")
	}

	shortPost := "<b>Краткий пост</b>"
	if err := copywriter.ValidatePost(shortPost, true); err != nil {
		t.Errorf("expected short post to be valid: %v", err)
	}

	tooLongCaption := strings.Repeat("A", copywriter.MaxCaptionLength+10)
	if err := copywriter.ValidatePost(tooLongCaption, true); err == nil {
		t.Errorf("expected error for too long caption")
	}
}

func TestLLMClient_FallbackWhenNoKey(t *testing.T) {
	client := copywriter.NewLLMClient(copywriter.Config{
		APIKey: "", // Пустой ключ
	})

	pack := copywriter.FactPack{
		PostType: "round_fixtures",
		League:   "Premier League",
		Round:    "ТУР 29",
	}

	post, err := client.GeneratePost(context.Background(), pack)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(post, "Premier League") {
		t.Errorf("expected fallback post content")
	}
}

func TestLLMClient_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "🔥 <b>Огненный тур в АПЛ!</b> Ливерпуль вырывает победу."
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := copywriter.NewLLMClient(copywriter.Config{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
	})

	pack := copywriter.FactPack{
		PostType: "round_results",
		League:   "Premier League",
		Round:    "ТУР 28",
	}

	post, err := client.GeneratePost(context.Background(), pack)
	if err != nil {
		t.Fatalf("GeneratePost failed: %v", err)
	}

	if !strings.Contains(post, "Огненный тур") {
		t.Errorf("expected mock server response, got: %s", post)
	}
}
