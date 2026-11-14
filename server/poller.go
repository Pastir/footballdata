package server

import (
	"context"
	"fmt"
	"time"

	"github.com/Pastir/footballdata/analytics"
	handler "github.com/Pastir/footballdata/api/handlers"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

type matchSnapshot struct {
	StateShort string
	Score      string
}

// LivePoller реализует настраиваемый фоновый опросчик текущих in-play матчей.
type LivePoller struct {
	pool         *pgxpool.Pool
	store        *LiveStore
	pollInterval time.Duration
	log          *zap.Logger
	prevMatches  map[int64]matchSnapshot
	initialized  bool
}

// NewLivePoller создает новый опросчик с настраиваемым интервалом (1 мин по умолчанию).
func NewLivePoller(pool *pgxpool.Pool, store *LiveStore, interval time.Duration, _ []int, log *zap.Logger) *LivePoller {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if log == nil {
		l, _ := zap.NewDevelopment()
		log = l
	}

	return &LivePoller{
		pool:         pool,
		store:        store,
		pollInterval: interval,
		log:          log,
		prevMatches:  make(map[int64]matchSnapshot),
	}
}

// PollInterval возвращает установленный интервал опроса.
func (p *LivePoller) PollInterval() time.Duration {
	return p.pollInterval
}

// hasActiveMatchWindow проверяет в локальной БД, есть ли прямо сейчас идущие матчи
// или матчи, стартующие в ближайшие 15 минут в активных лигах.
func (p *LivePoller) hasActiveMatchWindow(ctx context.Context) bool {
	if p.pool == nil {
		return false
	}
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM provider_fixtures pf
			JOIN provider_leagues pl ON pl.id = pf.league_id
			WHERE pl.active = true
			  AND (
			      (pf.starting_at BETWEEN ((NOW() AT TIME ZONE 'UTC') - INTERVAL '150 minutes')
			                          AND ((NOW() AT TIME ZONE 'UTC') + INTERVAL '15 minutes')
			       AND COALESCE(pf.state_id, 1) NOT IN (5, 6, 7, 10, 11, 12, 13, 14, 15, 17))
			      OR (pf.state_id IN (2, 3, 4, 9, 21, 22, 23, 25)
			          AND pf.starting_at >= (NOW() AT TIME ZONE 'UTC') - INTERVAL '6 hours')
			  )
		)`
	var active bool
	if err := p.pool.QueryRow(ctx, query).Scan(&active); err != nil {
		p.log.Warn("failed to check active match window", zap.Error(err))
		return false
	}
	return active
}

// PollOnce выполняет одиночный цикл опроса live-данных:
// при наличии активного окна матчей запрашивает Sportmonks livescores,
// а при изменении счёта или статуса матча в лиге — запрашивает live-таблицу лиги.
func (p *LivePoller) PollOnce(ctx context.Context) ([]analytics.RoundResultMatch, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("poller database pool is nil")
	}

	if p.hasActiveMatchWindow(ctx) && len(db.Prov.Endpoints) > 0 {
		var liveTask config.TaskData
		liveTask.Init()
		liveTask.Args.TaskName = "UpsertFixturesLive"
		handler.EndPoint(liveTask)
	}

	matches, err := analytics.GetLiveFixtures(ctx, p.pool, 0, "ru")
	if err != nil {
		p.log.Error("live poller query error", zap.Error(err))
		return nil, err
	}

	p.store.SetMatches(matches)

	changedLeagues := make(map[int]int)
	newlyFinished := false
	inPlayCount := 0

	for _, m := range matches {
		curState := ""
		if m.StateShort != nil {
			curState = *m.StateShort
		}
		if curState != "" && curState != "NS" && curState != "FT" {
			inPlayCount++
		}
		curScore := m.Scores.FullScoreString()

		if p.initialized {
			prev, existed := p.prevMatches[m.FixtureID]
			if !existed || prev.StateShort != curState || prev.Score != curScore {
				if m.LeagueID > 0 && m.RoundID > 0 {
					changedLeagues[m.LeagueID] = m.RoundID
				}
				if (curState == "FT" || curState == "AET" || curState == "FTP") && prev.StateShort != curState {
					newlyFinished = true
				}
			}
		}
		p.prevMatches[m.FixtureID] = matchSnapshot{
			StateShort: curState,
			Score:      curScore,
		}
	}
	p.initialized = true

	if len(changedLeagues) > 0 && len(db.Prov.Endpoints) > 0 {
		for leagueID, roundID := range changedLeagues {
			p.log.Info("live match changed, updating live standings",
				zap.Int("league_id", leagueID),
				zap.Int("round_id", roundID),
			)
			var stTask config.TaskData
			stTask.Init()
			stTask.Args.TaskName = "upsertStandingsLive"
			stTask.Args.LeagueId = leagueID
			stTask.Args.RoundId = roundID
			handler.EndPoint(stTask)
		}
	}

	if newlyFinished && len(db.Prov.Endpoints) > 0 {
		today := time.Now().UTC().Format("2006-01-02")
		var xgTask config.TaskData
		xgTask.Init()
		xgTask.Args.TaskName = "upsertExpectedFixtures"
		xgTask.Args.StartDate = today
		xgTask.Args.EndDate = today
		handler.EndPoint(xgTask)
	}

	if inPlayCount > 0 {
		p.log.Info("live matches updated",
			zap.Int("in_play", inPlayCount),
			zap.Int("total_today", len(matches)),
			zap.Duration("interval", p.pollInterval),
		)
	}

	return matches, nil
}

// Run запускает фоновый опрос live-данных с настраиваемой периодичностью до отмены контекста.
func (p *LivePoller) Run(ctx context.Context) error {
	p.log.Info("live poller daemon started",
		zap.Duration("interval", p.pollInterval),
	)

	// Выполняем первичный опрос сразу при старте
	_, _ = p.PollOnce(ctx)

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.log.Info("live poller gracefully stopped")
			return ctx.Err()

		case <-ticker.C:
			_, _ = p.PollOnce(ctx)
		}
	}
}
