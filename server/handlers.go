package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Pastir/footballdata/analytics"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

// ServerHandlers объединяет зависимости для HTTP обработчиков.
type ServerHandlers struct {
	pool      *pgxpool.Pool
	store     *LiveStore
	poller    *LivePoller
	staticDir string
	log       *zap.Logger
}

// NewServerHandlers создает экземпляр ServerHandlers.
func NewServerHandlers(pool *pgxpool.Pool, store *LiveStore, poller *LivePoller, staticDir string, log *zap.Logger) *ServerHandlers {
	if log == nil {
		l, _ := zap.NewDevelopment()
		log = l
	}
	return &ServerHandlers{
		pool:      pool,
		store:     store,
		poller:    poller,
		staticDir: staticDir,
		log:       log,
	}
}

// withCORS добавляет стандартные CORS заголовки для интеграции с Telegram Mini App.
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// writeJSON отправляет форматированный JSON ответ клиенту.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// HandleHealth возвращает статус доступности API сервера и опросчика.
func (h *ServerHandlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	pollIntervalStr := "unknown"
	if h.poller != nil {
		pollIntervalStr = h.poller.PollInterval().String()
	}

	liveCount := 0
	lastUpdate := "never"
	if h.store != nil {
		liveCount = h.store.Count()
		if !h.store.LastUpdated().IsZero() {
			lastUpdate = h.store.LastUpdated().Format(time.RFC3339)
		}
	}

	resp := map[string]interface{}{
		"status":        "ok",
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"poll_interval": pollIntervalStr,
		"live_matches":  liveCount,
		"last_updated":  lastUpdate,
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleLeagues отдает каталог лиг.
func (h *ServerHandlers) HandleLeagues(w http.ResponseWriter, r *http.Request) {
	// Сначала проверяем файл active_leagues.json в staticDir
	if h.staticDir != "" {
		filePath := filepath.Join(h.staticDir, "active_leagues.json")
		if data, err := os.ReadFile(filePath); err == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}
	}

	// Fallback на прямое чтение из БД
	catalog, err := analytics.GetActiveLeagues(r.Context(), h.pool)
	if err != nil {
		h.log.Error("failed to load leagues catalog", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, catalog)
}

// HandleLeagueArchive отдает список всех сезонов и туров лиги для интерактивного селектора в Mini App.
func (h *ServerHandlers) HandleLeagueArchive(w http.ResponseWriter, r *http.Request) {
	// Ожидается путь: /api/leagues/{id}/archive
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid url format, expected /api/leagues/{id}/archive"})
		return
	}

	leagueID, err := strconv.Atoi(parts[2])
	if err != nil || leagueID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid league id"})
		return
	}

	archive, err := analytics.GetLeagueArchive(r.Context(), h.pool, leagueID)
	if err != nil {
		h.log.Error("archive query failed", zap.Int("league_id", leagueID), zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, archive)
}

// HandleRoundOnDemand генерирует или отдает полные данные тура On-Demand.
func (h *ServerHandlers) HandleRoundOnDemand(w http.ResponseWriter, r *http.Request) {
	// Ожидается путь: /api/rounds/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid url format, expected /api/rounds/{round_id}"})
		return
	}

	roundID, err := strconv.Atoi(parts[2])
	if err != nil || roundID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid round id"})
		return
	}

	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "ru"
	}

	matches, info, err := analytics.GetRoundResultsByRoundID(r.Context(), h.pool, roundID, locale)
	if err != nil {
		h.log.Error("GetRoundResultsByRoundID failed", zap.Int("round_id", roundID), zap.Error(err))
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("round %d not found: %v", roundID, err)})
		return
	}

	standings, _ := analytics.GetCurrentStandings(r.Context(), h.pool, info)
	summary := analytics.ComputeRoundSummary(matches)

	payload := analytics.RoundExportPayload{
		LeagueID:      info.LeagueID,
		LeagueName:    info.LeagueName,
		SeasonID:      info.SeasonID,
		SeasonName:    info.SeasonName,
		RoundID:       info.CurrRoundID,
		RoundName:     info.CurrRoundName,
		PrevRoundID:   info.PrevRoundID,
		NextRoundID:   info.NextRoundID,
		Summary:       summary,
		Matches:       matches,
		Standings:     standings,
	}

	writeJSON(w, http.StatusOK, payload)
}

// HandleLiveMatches возвращает список текущих live / in-play матчей.
func (h *ServerHandlers) HandleLiveMatches(w http.ResponseWriter, r *http.Request) {
	leagueID := 0
	if lStr := r.URL.Query().Get("league_id"); lStr != "" {
		leagueID, _ = strconv.Atoi(lStr)
	}

	// Сначала проверяем LiveStore (кэш в памяти)
	if h.store != nil && h.store.Count() > 0 {
		matches := h.store.GetLiveMatches(leagueID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"updated_at": h.store.LastUpdated().Format(time.RFC3339),
			"total":      len(matches),
			"matches":    matches,
		})
		return
	}

	// Fallback в базу, если кэш пуст
	matches, err := analytics.GetLiveFixtures(r.Context(), h.pool, leagueID, "ru")
	if err != nil {
		h.log.Error("GetLiveFixtures failed", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"updated_at": time.Now().UTC().Format(time.RFC3339),
		"total":      len(matches),
		"matches":    matches,
	})
}

// HandleFixtureLive возвращает оперативный срез live-данных для конкретного матча.
func (h *ServerHandlers) HandleFixtureLive(w http.ResponseWriter, r *http.Request) {
	// Ожидается путь: /api/fixtures/{id}/live
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid url format, expected /api/fixtures/{id}/live"})
		return
	}

	fixtureID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || fixtureID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid fixture id"})
		return
	}

	// Сначала проверяем LiveStore
	if h.store != nil {
		if m, ok := h.store.GetFixtureLive(fixtureID); ok {
			writeJSON(w, http.StatusOK, m)
			return
		}
	}

	// Fallback в базу
	m, err := analytics.GetFixtureLiveDetails(r.Context(), h.pool, fixtureID, "ru")
	if err != nil {
		h.log.Error("GetFixtureLiveDetails failed", zap.Int64("fixture_id", fixtureID), zap.Error(err))
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("fixture %d not found: %v", fixtureID, err)})
		return
	}

	writeJSON(w, http.StatusOK, m)
}
