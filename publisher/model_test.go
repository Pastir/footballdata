package publisher_test

import (
	"testing"
	"time"

	"github.com/Pastir/footballdata/publisher"
)

func TestPublication_IsTerminal(t *testing.T) {
	tests := []struct {
		status   publisher.PublicationStatus
		terminal bool
	}{
		{publisher.StatusDraft, false},
		{publisher.StatusScheduled, false},
		{publisher.StatusRendering, false},
		{publisher.StatusReady, false},
		{publisher.StatusFailed, false},
		{publisher.StatusPublished, true},
		{publisher.StatusCancelled, true},
	}

	for _, tt := range tests {
		pub := publisher.Publication{Status: tt.status}
		if got := pub.IsTerminal(); got != tt.terminal {
			t.Errorf("IsTerminal() for status %s = %v, expected %v", tt.status, got, tt.terminal)
		}
	}
}

func TestPublication_CanRetry(t *testing.T) {
	pub := publisher.Publication{
		Status:     publisher.StatusFailed,
		RetryCount: 1,
	}

	if !pub.CanRetry(3) {
		t.Errorf("expected CanRetry(3) to be true for RetryCount 1")
	}

	pub.RetryCount = 3
	if pub.CanRetry(3) {
		t.Errorf("expected CanRetry(3) to be false for RetryCount 3")
	}

	pub.Status = publisher.StatusPublished
	pub.RetryCount = 0
	if pub.CanRetry(3) {
		t.Errorf("expected CanRetry to be false for StatusPublished")
	}
}

func TestStorage_InterfaceCompliance(t *testing.T) {
	var _ publisher.Storage = (*publisher.PGStorage)(nil)
}

func TestPublication_CreationDefaults(t *testing.T) {
	now := time.Now()
	leagueID := 8
	roundID := 29

	pub := publisher.Publication{
		ChannelID:       -1001234567890,
		PublicationType: publisher.PubTypeRoundFixtures,
		LeagueID:        &leagueID,
		RoundID:         &roundID,
		Title:           "Premier League — Тур 29",
		MediaURL:        "/var/data/fixtures_29.png",
		MediaType:       publisher.MediaTypePhoto,
		Status:          publisher.StatusDraft,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if pub.PublicationType != "round_fixtures" {
		t.Errorf("unexpected publication type: %s", pub.PublicationType)
	}
	if pub.ChannelID != -1001234567890 {
		t.Errorf("unexpected channel ID: %d", pub.ChannelID)
	}
}
