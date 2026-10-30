package prediction_test

import (
	"testing"

	"github.com/Pastir/footballdata/prediction"
)

func TestCalculatePerformance(t *testing.T) {
	preds := []prediction.Prediction{
		{
			Status:           prediction.StatusWon,
			TotalOdds:        2.00,
			RecommendedStake: 1.0,
			ProfitUnits:      1.0,
		},
		{
			Status:           prediction.StatusWon,
			TotalOdds:        1.80,
			RecommendedStake: 1.0,
			ProfitUnits:      0.80,
		},
		{
			Status:           prediction.StatusLost,
			TotalOdds:        2.50,
			RecommendedStake: 1.0,
			ProfitUnits:      -1.0,
		},
		{
			Status:           prediction.StatusRefund,
			TotalOdds:        1.90,
			RecommendedStake: 1.0,
			ProfitUnits:      0.0,
		},
	}

	report := prediction.CalculatePerformance(preds)

	if report.TotalBets != 4 {
		t.Errorf("expected TotalBets 4, got %d", report.TotalBets)
	}
	if report.WonBets != 2 {
		t.Errorf("expected WonBets 2, got %d", report.WonBets)
	}
	if report.LostBets != 1 {
		t.Errorf("expected LostBets 1, got %d", report.LostBets)
	}
	if report.RefundBets != 1 {
		t.Errorf("expected RefundBets 1, got %d", report.RefundBets)
	}

	// WinRate = 2 / (2 + 1) * 100 = 66.67%
	if report.WinRate != 66.67 {
		t.Errorf("expected WinRate 66.67, got %v", report.WinRate)
	}

	// TotalStaked = 4.0u
	if report.TotalStaked != 4.0 {
		t.Errorf("expected TotalStaked 4.0, got %v", report.TotalStaked)
	}

	// TotalProfit = 1.0 + 0.8 - 1.0 = 0.80u
	if report.TotalProfit != 0.80 {
		t.Errorf("expected TotalProfit 0.80, got %v", report.TotalProfit)
	}

	// ROI = 0.80 / 4.0 * 100 = 20.00%
	if report.ROI != 20.00 {
		t.Errorf("expected ROI 20.00, got %v", report.ROI)
	}

	// AvgOdds = (2.00 + 1.80) / 2 = 1.90
	if report.AvgOdds != 1.90 {
		t.Errorf("expected AvgOdds 1.90, got %v", report.AvgOdds)
	}

	// ProfitFactor = 1.80 / 1.0 = 1.80
	if report.ProfitFactor != 1.80 {
		t.Errorf("expected ProfitFactor 1.80, got %v", report.ProfitFactor)
	}
}
