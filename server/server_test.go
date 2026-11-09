package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Pastir/footballdata/analytics"
)

func TestLiveStore_SetAndGet(t *testing.T) {
	store := NewLiveStore()

	homeID := int64(10)
	awayID := int64(20)
	stShort := "2nd"

	testMatches := []analytics.RoundResultMatch{
		{
			LeagueID:     8,
			FixtureID:    1001,
			TeamHomeID:   homeID,
			TeamHomeName: "Arsenal",
			TeamAwayID:   awayID,
			TeamAwayName: "Chelsea",
			StateShort:   &stShort,
		},
		{
			LeagueID:     564,
			FixtureID:    2001,
			TeamHomeID:   30,
			TeamHomeName: "Real Madrid",
			TeamAwayID:   40,
			TeamAwayName: "Barcelona",
			StateShort:   &stShort,
		},
	}

	store.SetMatches(testMatches)

	if store.Count() != 2 {
		t.Fatalf("expected count 2, got %d", store.Count())
	}

	// Проверяем получение по лиге
	plMatches := store.GetLiveMatches(8)
	if len(plMatches) != 1 {
		t.Fatalf("expected 1 PL match, got %d", len(plMatches))
	}
	if plMatches[0].FixtureID != 1001 {
		t.Errorf("expected fixture 1001, got %d", plMatches[0].FixtureID)
	}

	// Проверяем получение всех
	all := store.GetLiveMatches(0)
	if len(all) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(all))
	}

	// Проверяем одиночный матч
	m, found := store.GetFixtureLive(1001)
	if !found {
		t.Fatalf("fixture 1001 not found in store")
	}
	if m.TeamHomeName != "Arsenal" {
		t.Errorf("expected Arsenal, got %s", m.TeamHomeName)
	}

	// Несуществующий матч
	_, foundMissing := store.GetFixtureLive(9999)
	if foundMissing {
		t.Errorf("expected fixture 9999 to not be found")
	}
}

func TestLiveStore_Concurrency(t *testing.T) {
	store := NewLiveStore()
	var wg sync.WaitGroup

	st := "1st"
	// Конкурентные писатели и читатели
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			m := []analytics.RoundResultMatch{
				{
					LeagueID:   8,
					FixtureID:  int64(idx),
					StateShort: &st,
				},
			}
			store.SetMatches(m)
		}(i)

		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.GetLiveMatches(8)
			_ = store.Count()
		}()
	}

	wg.Wait()
}

func TestHandleHealth(t *testing.T) {
	store := NewLiveStore()
	poller := NewLivePoller(nil, store, 25*time.Second, []int{8}, nil)
	handlers := NewServerHandlers(nil, store, poller, "", nil)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler := withCORS(handlers.HandleHealth)
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	// Проверяем CORS
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS header *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health json: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %v", body["status"])
	}
	if body["poll_interval"] != "25s" {
		t.Errorf("expected poll_interval 25s, got %v", body["poll_interval"])
	}
}

func TestHandleLiveMatches_FromStore(t *testing.T) {
	store := NewLiveStore()
	st := "1st"
	store.SetMatches([]analytics.RoundResultMatch{
		{
			LeagueID:     8,
			FixtureID:    555,
			TeamHomeName: "Liverpool",
			TeamAwayName: "Everton",
			StateShort:   &st,
		},
	})

	handlers := NewServerHandlers(nil, store, nil, "", nil)
	req := httptest.NewRequest("GET", "/api/live?league_id=8", nil)
	w := httptest.NewRecorder()

	handler := withCORS(handlers.HandleLiveMatches)
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	var body struct {
		Total   int                          `json:"total"`
		Matches []analytics.RoundResultMatch `json:"matches"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	if body.Total != 1 || len(body.Matches) != 1 {
		t.Fatalf("expected 1 match, got total=%d len=%d", body.Total, len(body.Matches))
	}
	if body.Matches[0].FixtureID != 555 {
		t.Errorf("expected fixture 555, got %d", body.Matches[0].FixtureID)
	}
}

func TestNormalizeLocale(t *testing.T) {
	if got := normalizeLocale("EN"); got != "en" {
		t.Errorf("expected en, got %s", got)
	}
	if got := normalizeLocale("uk"); got != "ua" {
		t.Errorf("expected ua, got %s", got)
	}
	if got := normalizeLocale("malicious_random_key"); got != "ru" {
		t.Errorf("expected ru fallback, got %s", got)
	}
}
