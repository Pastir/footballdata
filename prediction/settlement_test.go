package prediction_test

import (
	"testing"

	"github.com/Pastir/footballdata/prediction"
)

func TestParseScore(t *testing.T) {
	tests := []struct {
		score   string
		home    int
		away    int
		wantErr bool
	}{
		{"2:1", 2, 1, false},
		{"0:0", 0, 0, false},
		{"3-2", 3, 2, false},
		{" 4 : 2 ", 4, 2, false},
		{"invalid", 0, 0, true},
		{"1:a", 0, 0, true},
	}

	for _, tt := range tests {
		h, a, err := prediction.ParseScore(tt.score)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseScore(%q) err = %v, wantErr %v", tt.score, err, tt.wantErr)
		}
		if !tt.wantErr && (h != tt.home || a != tt.away) {
			t.Errorf("ParseScore(%q) = (%d, %d), expected (%d, %d)", tt.score, h, a, tt.home, tt.away)
		}
	}
}

func TestEvaluateItemOutcome_1X2(t *testing.T) {
	// Home Win
	status, _, err := prediction.EvaluateItemOutcome(prediction.MarketKey1X2, "1", "2:1")
	if err != nil || status != prediction.StatusWon {
		t.Errorf("expected Won for 1 with 2:1, got %v (err: %v)", status, err)
	}

	status, _, _ = prediction.EvaluateItemOutcome(prediction.MarketKey1X2, "1", "1:1")
	if status != prediction.StatusLost {
		t.Errorf("expected Lost for 1 with 1:1, got %v", status)
	}

	// Draw
	status, _, _ = prediction.EvaluateItemOutcome(prediction.MarketKey1X2, "X", "1:1")
	if status != prediction.StatusWon {
		t.Errorf("expected Won for X with 1:1, got %v", status)
	}

	// Away Win
	status, _, _ = prediction.EvaluateItemOutcome(prediction.MarketKey1X2, "2", "0:3")
	if status != prediction.StatusWon {
		t.Errorf("expected Won for 2 with 0:3, got %v", status)
	}
}

func TestEvaluateItemOutcome_DoubleChance(t *testing.T) {
	// 1X
	status, _, _ := prediction.EvaluateItemOutcome(prediction.MarketKeyDoubleChance, "1X", "2:1")
	if status != prediction.StatusWon {
		t.Errorf("expected Won for 1X with 2:1, got %v", status)
	}
	status, _, _ = prediction.EvaluateItemOutcome(prediction.MarketKeyDoubleChance, "1X", "1:1")
	if status != prediction.StatusWon {
		t.Errorf("expected Won for 1X with 1:1, got %v", status)
	}
	status, _, _ = prediction.EvaluateItemOutcome(prediction.MarketKeyDoubleChance, "1X", "0:1")
	if status != prediction.StatusLost {
		t.Errorf("expected Lost for 1X with 0:1, got %v", status)
	}
}

func TestEvaluateItemOutcome_OverUnder(t *testing.T) {
	// Over 2.5
	status, _, _ := prediction.EvaluateItemOutcome(prediction.MarketKeyOverUnder, "Over 2.5", "2:1")
	if status != prediction.StatusWon {
		t.Errorf("expected Won for Over 2.5 with 2:1 (3 goals), got %v", status)
	}
	status, _, _ = prediction.EvaluateItemOutcome(prediction.MarketKeyOverUnder, "Over 2.5", "1:1")
	if status != prediction.StatusLost {
		t.Errorf("expected Lost for Over 2.5 with 1:1 (2 goals), got %v", status)
	}

	// Under 2.5
	status, _, _ = prediction.EvaluateItemOutcome(prediction.MarketKeyOverUnder, "Under 2.5", "1:0")
	if status != prediction.StatusWon {
		t.Errorf("expected Won for Under 2.5 with 1:0, got %v", status)
	}

	// ТБ 2,5 (с запятой)
	status, _, err := prediction.EvaluateItemOutcome(prediction.MarketKeyOverUnder, "ТБ 2,5", "2:1")
	if err != nil || status != prediction.StatusWon {
		t.Errorf("expected Won for ТБ 2,5 with 2:1, got %v (err: %v)", status, err)
	}

	// ТБ2.5 (слитно)
	status, _, err = prediction.EvaluateItemOutcome(prediction.MarketKeyOverUnder, "ТБ2.5", "2:1")
	if err != nil || status != prediction.StatusWon {
		t.Errorf("expected Won for ТБ2.5 with 2:1, got %v (err: %v)", status, err)
	}
}

func TestEvaluateItemOutcome_BTTS(t *testing.T) {
	// BTTS Yes
	status, _, _ := prediction.EvaluateItemOutcome(prediction.MarketKeyBTTS, "Yes", "2:1")
	if status != prediction.StatusWon {
		t.Errorf("expected Won for BTTS Yes with 2:1, got %v", status)
	}
	status, _, _ = prediction.EvaluateItemOutcome(prediction.MarketKeyBTTS, "Yes", "2:0")
	if status != prediction.StatusLost {
		t.Errorf("expected Lost for BTTS Yes with 2:0, got %v", status)
	}

	// BTTS No
	status, _, _ = prediction.EvaluateItemOutcome(prediction.MarketKeyBTTS, "No", "2:0")
	if status != prediction.StatusWon {
		t.Errorf("expected Won for BTTS No with 2:0, got %v", status)
	}
}

func TestEvaluatePrediction_Single(t *testing.T) {
	pred := &prediction.Prediction{
		BetType:          prediction.BetTypeSingle,
		TotalOdds:        2.00,
		RecommendedStake: 1.0,
		Items: []prediction.PredictionItem{
			{Odds: 2.00, SettlementStatus: prediction.StatusWon},
		},
	}

	status, profit := prediction.EvaluatePrediction(pred)
	if status != prediction.StatusWon || profit != 1.0 {
		t.Errorf("expected Won with profit 1.0, got status %v, profit %v", status, profit)
	}

	pred.Items[0].SettlementStatus = prediction.StatusLost
	status, profit = prediction.EvaluatePrediction(pred)
	if status != prediction.StatusLost || profit != -1.0 {
		t.Errorf("expected Lost with profit -1.0, got status %v, profit %v", status, profit)
	}
}

func TestEvaluatePrediction_Accumulator(t *testing.T) {
	// Экспресс из двух событий (1.80 * 2.00 = 3.60)
	pred := &prediction.Prediction{
		BetType:          prediction.BetTypeAccumulator,
		TotalOdds:        3.60,
		RecommendedStake: 2.0,
		Items: []prediction.PredictionItem{
			{Odds: 1.80, SettlementStatus: prediction.StatusWon},
			{Odds: 2.00, SettlementStatus: prediction.StatusWon},
		},
	}

	status, profit := prediction.EvaluatePrediction(pred)
	if status != prediction.StatusWon || profit != 5.20 {
		t.Errorf("expected Won with profit 5.20, got status %v, profit %v", status, profit)
	}

	// Одно событие проиграло — весь экспресс проигран
	pred.Items[1].SettlementStatus = prediction.StatusLost
	status, profit = prediction.EvaluatePrediction(pred)
	if status != prediction.StatusLost || profit != -2.0 {
		t.Errorf("expected Lost with profit -2.0, got status %v, profit %v", status, profit)
	}
}
