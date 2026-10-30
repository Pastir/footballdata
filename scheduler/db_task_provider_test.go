package scheduler_test

import (
	"context"
	"testing"
	"time"

	"github.com/Pastir/footballdata/scheduler"
)

func TestNewDBTaskProvider_DefaultLeagues(t *testing.T) {
	storage := newMockPublisherStorage()
	provider := scheduler.NewDBTaskProvider(nil, storage, "mock_token", -100123456789, nil)
	if provider == nil {
		t.Fatalf("expected non-nil provider")
	}

	// Проверяем вызов на nil pool (безопасность от паник)
	tasks, err := provider.GetDueTasks(context.Background(), time.Now())
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks with nil pool, got %d", len(tasks))
	}
	_ = err

	statusList, err := provider.CheckRoundStatusAll(context.Background(), time.Now())
	if err == nil && len(statusList) > 0 {
		t.Errorf("expected error or empty list on nil pool")
	}
}
