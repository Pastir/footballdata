package server

import (
	"context"
	"fmt"
	"time"

	"github.com/Pastir/footballdata/analytics"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

// LivePoller реализует настраиваемый фоновый опросчик текущих in-play матчей.
type LivePoller struct {
	pool         *pgxpool.Pool
	store        *LiveStore
	pollInterval time.Duration
	log          *zap.Logger
}

// NewLivePoller создает новый опросчик с настраиваемым интервалом (15-30 сек по умолчанию).
func NewLivePoller(pool *pgxpool.Pool, store *LiveStore, interval time.Duration, _ []int, log *zap.Logger) *LivePoller {
	if interval <= 0 {
		interval = 20 * time.Second
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
	}
}

// PollInterval возвращает установленный интервал опроса.
func (p *LivePoller) PollInterval() time.Duration {
	return p.pollInterval
}

// PollOnce выполняет одиночный цикл опроса live-данных из базы.
func (p *LivePoller) PollOnce(ctx context.Context) ([]analytics.RoundResultMatch, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("poller database pool is nil")
	}

	matches, err := analytics.GetLiveFixtures(ctx, p.pool, 0, "ru")
	if err != nil {
		p.log.Error("live poller query error", zap.Error(err))
		return nil, err
	}

	p.store.SetMatches(matches)

	inPlayCount := 0
	for _, m := range matches {
		if m.StateShort != nil && *m.StateShort != "NS" && *m.StateShort != "FT" {
			inPlayCount++
		}
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
