package prediction_test

import (
	"context"
	"testing"

	"github.com/Pastir/footballdata/prediction"
)

type mockScoreProvider struct {
	scores map[int]string
}

func (m *mockScoreProvider) GetFinishedScore(_ context.Context, fixtureID int) (string, string, bool, error) {
	score, ok := m.scores[fixtureID]
	if !ok {
		return "", "", false, nil
	}
	return score, "1:0", true, nil
}

type mockStorage struct {
	pending []prediction.Prediction
	settled map[int64]prediction.BetStatus
	profits map[int64]float64
}

func (m *mockStorage) EnsureSchema(_ context.Context) error { return nil }
func (m *mockStorage) CreatePrediction(_ context.Context, _ *prediction.Prediction) (int64, error) {
	return 1, nil
}
func (m *mockStorage) GetPredictionByID(_ context.Context, id int64) (*prediction.Prediction, error) {
	for _, p := range m.pending {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, nil
}
func (m *mockStorage) ListPendingPredictions(_ context.Context) ([]prediction.Prediction, error) {
	return m.pending, nil
}
func (m *mockStorage) SettleItem(_ context.Context, _ int64, _ prediction.BetStatus, _, _, _ string) error {
	return nil
}
func (m *mockStorage) SettlePrediction(_ context.Context, id int64, status prediction.BetStatus, profitUnits float64) error {
	m.settled[id] = status
	m.profits[id] = profitUnits
	return nil
}
func (m *mockStorage) GetPerformanceReport(_ context.Context) (*prediction.PerformanceReport, error) {
	return &prediction.PerformanceReport{}, nil
}

func TestSettlementWorker_ProcessPending(t *testing.T) {
	storage := &mockStorage{
		pending: []prediction.Prediction{
			{
				ID:               101,
				BetType:          prediction.BetTypeSingle,
				RecommendedStake: 1.0,
				Status:           prediction.StatusPending,
				Items: []prediction.PredictionItem{
					{
						ID:               1,
						FixtureID:        501,
						MarketKey:        prediction.MarketKey1X2,
						Selection:        "1",
						Odds:             2.10,
						SettlementStatus: prediction.StatusPending,
					},
				},
			},
		},
		settled: make(map[int64]prediction.BetStatus),
		profits: make(map[int64]float64),
	}

	scores := &mockScoreProvider{
		scores: map[int]string{
			501: "3:1", // Home Win
		},
	}

	worker := prediction.NewSettlementWorker(storage, scores)
	count, err := worker.ProcessPending(context.Background())
	if err != nil {
		t.Fatalf("ProcessPending failed: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 settled prediction, got %d", count)
	}

	if storage.settled[101] != prediction.StatusWon {
		t.Errorf("expected prediction 101 to be Won, got %v", storage.settled[101])
	}

	if storage.profits[101] != 1.10 {
		t.Errorf("expected profit 1.10, got %v", storage.profits[101])
	}
}
