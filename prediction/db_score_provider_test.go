package prediction_test

import (
	"context"
	"testing"

	"github.com/Pastir/footballdata/prediction"
)

func TestDBFixtureScoreProvider_NilPool(t *testing.T) {
	provider := prediction.NewDBFixtureScoreProvider(nil)
	if provider == nil {
		t.Fatalf("expected non-nil provider")
	}

	_, _, _, err := provider.GetFinishedScore(context.Background(), 12345)
	if err == nil {
		t.Errorf("expected error with nil pool, got nil")
	}
}
