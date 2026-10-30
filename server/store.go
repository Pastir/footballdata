package server

import (
	"sync"
	"time"

	"github.com/Pastir/footballdata/analytics"
)

// LiveStore обеспечивает потокобезопасное in-memory хранилище для оперативных live-данных матчей.
type LiveStore struct {
	mu           sync.RWMutex
	matches      map[int64]analytics.RoundResultMatch
	leagueIndex  map[int][]int64
	lastUpdated  time.Time
}

// NewLiveStore создает новый экземпляр LiveStore.
func NewLiveStore() *LiveStore {
	return &LiveStore{
		matches:     make(map[int64]analytics.RoundResultMatch),
		leagueIndex: make(map[int][]int64),
	}
}

// SetMatches обновляет снапшот активных live-матчей.
func (s *LiveStore) SetMatches(matches []analytics.RoundResultMatch) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newMatches := make(map[int64]analytics.RoundResultMatch, len(matches))
	newIndex := make(map[int][]int64)

	for _, m := range matches {
		newMatches[m.FixtureID] = m
		newIndex[m.LeagueID] = append(newIndex[m.LeagueID], m.FixtureID)
	}

	s.matches = newMatches
	s.leagueIndex = newIndex
	s.lastUpdated = time.Now().UTC()
}

// GetLiveMatches возвращает список текущих live-матчей (опционально по лиге).
func (s *LiveStore) GetLiveMatches(leagueID int) []analytics.RoundResultMatch {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if leagueID > 0 {
		fids, ok := s.leagueIndex[leagueID]
		if !ok || len(fids) == 0 {
			return []analytics.RoundResultMatch{}
		}
		res := make([]analytics.RoundResultMatch, 0, len(fids))
		for _, fid := range fids {
			if m, found := s.matches[fid]; found {
				res = append(res, m)
			}
		}
		return res
	}

	res := make([]analytics.RoundResultMatch, 0, len(s.matches))
	for _, m := range s.matches {
		res = append(res, m)
	}
	return res
}

// GetFixtureLive возвращает live-данные конкретного матча по ID.
func (s *LiveStore) GetFixtureLive(fixtureID int64) (analytics.RoundResultMatch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.matches[fixtureID]
	return m, ok
}

// LastUpdated возвращает время последнего обновления кэша.
func (s *LiveStore) LastUpdated() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdated
}

// Count возвращает количество матчей в кэше.
func (s *LiveStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.matches)
}
