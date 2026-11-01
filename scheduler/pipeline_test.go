package scheduler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Pastir/footballdata/copywriter"
	"github.com/Pastir/footballdata/publisher"
	"github.com/Pastir/footballdata/scheduler"
)

type mockPublisherStorage struct {
	publishedMap map[string]bool
	items        map[int64]*publisher.Publication
	lastID       int64
}

func newMockPublisherStorage() *mockPublisherStorage {
	return &mockPublisherStorage{
		publishedMap: make(map[string]bool),
		items:        make(map[int64]*publisher.Publication),
	}
}

func (m *mockPublisherStorage) EnsureSchema(_ context.Context) error { return nil }

func (m *mockPublisherStorage) Create(_ context.Context, pub *publisher.Publication) (int64, error) {
	m.lastID++
	pub.ID = m.lastID
	m.items[pub.ID] = pub
	return pub.ID, nil
}

func (m *mockPublisherStorage) GetByID(_ context.Context, id int64) (*publisher.Publication, error) {
	pub, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return pub, nil
}

func (m *mockPublisherStorage) IsAlreadyPublished(_ context.Context, channelID int64, pubType publisher.PublicationType, leagueID, roundID int) (bool, error) {
	key := fmt.Sprintf("%d_%s_%d_%d", channelID, pubType, leagueID, roundID)
	return m.publishedMap[key], nil
}

func (m *mockPublisherStorage) GetLastPublished(_ context.Context, _ int64, _ publisher.PublicationType, _, _ int) (*publisher.Publication, error) {
	return nil, nil
}

func (m *mockPublisherStorage) MarkPublished(_ context.Context, id int64, messageID int64) error {
	pub, ok := m.items[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	pub.Status = publisher.StatusPublished
	pub.MessageID = &messageID
	now := time.Now()
	pub.PublishedAt = &now

	leagueID := 0
	if pub.LeagueID != nil {
		leagueID = *pub.LeagueID
	}
	roundID := 0
	if pub.RoundID != nil {
		roundID = *pub.RoundID
	}
	key := fmt.Sprintf("%d_%s_%d_%d", pub.ChannelID, pub.PublicationType, leagueID, roundID)
	m.publishedMap[key] = true
	return nil
}

func (m *mockPublisherStorage) MarkFailed(_ context.Context, id int64, errMsg string) error {
	pub, ok := m.items[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	pub.Status = publisher.StatusFailed
	pub.ErrorMessage = errMsg
	pub.RetryCount++
	return nil
}

func (m *mockPublisherStorage) RecordSkipped(_ context.Context, channelID int64, pubType publisher.PublicationType, leagueID, roundID int, _ string) error {
	key := fmt.Sprintf("%d_%s_%d_%d", channelID, pubType, leagueID, roundID)
	m.publishedMap[key] = true
	return nil
}

func (m *mockPublisherStorage) UpdateStatus(_ context.Context, id int64, status publisher.PublicationStatus) error {
	pub, ok := m.items[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	pub.Status = status
	return nil
}

func (m *mockPublisherStorage) ListPendingScheduled(_ context.Context, _ time.Time, _ int) ([]publisher.Publication, error) {
	return nil, nil
}

type mockCopywriterClient struct {
	response string
}

func (m *mockCopywriterClient) GeneratePost(_ context.Context, _ copywriter.FactPack) (string, error) {
	return m.response, nil
}

func TestPipeline_IdempotencyBlock(t *testing.T) {
	storage := newMockPublisherStorage()
	storage.publishedMap["-100123_round_results_8_28"] = true

	pipe := scheduler.NewPipeline(storage, &mockCopywriterClient{response: "Test"}, 3)

	task := scheduler.PublicationTask{
		ChannelID:       -100123,
		PublicationType: publisher.PubTypeRoundResults,
		LeagueID:        8,
		RoundID:         28,
		Title:           "Duplicate test",
	}

	_, err := pipe.Execute(context.Background(), task)
	if err == nil {
		t.Errorf("expected error due to idempotency block, got nil")
	}
}

type mockTaskProvider struct {
	tasks []scheduler.PublicationTask
}

func (m *mockTaskProvider) GetDueTasks(_ context.Context, _ time.Time) ([]scheduler.PublicationTask, error) {
	return m.tasks, nil
}

func TestDaemon_Tick(t *testing.T) {
	storage := newMockPublisherStorage()
	pipe := scheduler.NewPipeline(storage, &mockCopywriterClient{response: "Post text"}, 3)

	provider := &mockTaskProvider{
		tasks: []scheduler.PublicationTask{
			{
				ChannelID:       -100999,
				PublicationType: publisher.PubTypeTable,
				LeagueID:        8,
				RoundID:         28,
				Title:           "Table post",
				FactPack: copywriter.FactPack{
					PostType: "table",
					League:   "Premier League",
				},
			},
		},
	}

	daemon := scheduler.NewDaemon(pipe, provider, storage, 1*time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Выполняем 1 тик (упадёт на telegram send т.к. фейковый токен, но зафиксирует статус failed)
	_ = daemon.Tick(ctx)

	// Проверяем, что задача попала в хранилище и перешла в статус failed
	pub, err := storage.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("expected pub with id 1, got error: %v", err)
	}
	if pub.Status != publisher.StatusFailed {
		t.Errorf("expected status failed due to fake telegram endpoint, got %s", pub.Status)
	}
	if pub.RetryCount != 1 {
		t.Errorf("expected RetryCount 1, got %d", pub.RetryCount)
	}
}
