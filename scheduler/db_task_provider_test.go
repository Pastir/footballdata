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

func TestStorage_RecordSkipped(t *testing.T) {
	storage := newMockPublisherStorage()
	channelID := int64(-100999999)
	leagueID := 8
	roundID := 407865

	// Сначала пост не опубликован
	already, err := storage.IsAlreadyPublished(context.Background(), channelID, "round_results", leagueID, roundID)
	if err != nil || already {
		t.Fatalf("expected false, got %v (err: %v)", already, err)
	}

	// Фиксируем пропуск устаревшего раунда
	err = storage.RecordSkipped(context.Background(), channelID, "round_results", leagueID, roundID, "stale round")
	if err != nil {
		t.Fatalf("RecordSkipped failed: %v", err)
	}

	// После фиксации пропуска IsAlreadyPublished должен вернуть true (защита от спама)
	already, err = storage.IsAlreadyPublished(context.Background(), channelID, "round_results", leagueID, roundID)
	if err != nil || !already {
		t.Fatalf("expected true after skip, got %v (err: %v)", already, err)
	}
}
