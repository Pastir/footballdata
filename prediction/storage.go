package prediction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Storage определяет интерфейс работы с базой данных прогнозов.
type Storage interface {
	EnsureSchema(ctx context.Context) error
	CreatePrediction(ctx context.Context, p *Prediction) (int64, error)
	GetPredictionByID(ctx context.Context, id int64) (*Prediction, error)
	ListPendingPredictions(ctx context.Context) ([]Prediction, error)
	SettleItem(ctx context.Context, itemID int64, status BetStatus, ftScore, htScore, details string) error
	SettlePrediction(ctx context.Context, id int64, status BetStatus, profitUnits float64) error
	GetPerformanceReport(ctx context.Context) (*PerformanceReport, error)
}

// PGStorage реализует Storage поверх PostgreSQL.
type PGStorage struct {
	pool *pgxpool.Pool
}

// NewPGStorage создаёт новый репозиторий прогнозов.
func NewPGStorage(pool *pgxpool.Pool) *PGStorage {
	return &PGStorage{pool: pool}
}

// EnsureSchema инициализирует таблицы predictions и prediction_items.
func (s *PGStorage) EnsureSchema(ctx context.Context) error {
	const query = `
CREATE TABLE IF NOT EXISTS predictions (
    id BIGSERIAL PRIMARY KEY,
    bet_type VARCHAR(30) NOT NULL,
    title VARCHAR(255) NOT NULL,
    analysis_text TEXT,
    total_odds NUMERIC(8, 3) NOT NULL,
    confidence NUMERIC(4, 2),
    value_edge NUMERIC(5, 2),
    recommended_stake NUMERIC(5, 2) DEFAULT 1.0,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    profit_units NUMERIC(8, 3) DEFAULT 0.000,
    settled_at TIMESTAMPTZ,
    publication_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_predictions_status ON predictions(status);
CREATE INDEX IF NOT EXISTS idx_predictions_created ON predictions(created_at);

CREATE TABLE IF NOT EXISTS prediction_items (
    id BIGSERIAL PRIMARY KEY,
    prediction_id BIGINT NOT NULL REFERENCES predictions(id) ON DELETE CASCADE,
    fixture_id INT NOT NULL,
    league_id INT NOT NULL,
    home_team_id INT NOT NULL,
    away_team_id INT NOT NULL,
    match_starting_at TIMESTAMPTZ NOT NULL,
    market_name VARCHAR(100) NOT NULL,
    market_key VARCHAR(50) NOT NULL,
    selection VARCHAR(100) NOT NULL,
    odds NUMERIC(6, 3) NOT NULL,
    bookmaker_name VARCHAR(50) DEFAULT 'Pinnacle',
    actual_score_ft VARCHAR(20),
    actual_score_ht VARCHAR(20),
    settlement_status VARCHAR(20) DEFAULT 'pending',
    settlement_details TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pred_items_fixture ON prediction_items(fixture_id);
CREATE INDEX IF NOT EXISTS idx_pred_items_status ON prediction_items(settlement_status);
`
	_, err := s.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ensure predictions schema: %w", err)
	}
	return nil
}

// CreatePrediction сохраняет прогноз и все входящие в него исходы в одной транзакции.
func (s *PGStorage) CreatePrediction(ctx context.Context, p *Prediction) (int64, error) {
	if p.Status == "" {
		p.Status = StatusPending
	}
	if p.RecommendedStake <= 0 {
		p.RecommendedStake = 1.0
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertPred = `
INSERT INTO predictions (
    bet_type, title, analysis_text, total_odds, confidence, value_edge,
    recommended_stake, status, profit_units, settled_at, publication_id,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    NOW(), NOW()
) RETURNING id`

	var predID int64
	err = tx.QueryRow(ctx, insertPred,
		string(p.BetType), p.Title, p.AnalysisText, p.TotalOdds, p.Confidence, p.ValueEdge,
		p.RecommendedStake, string(p.Status), p.ProfitUnits, p.SettledAt, p.PublicationID,
	).Scan(&predID)
	if err != nil {
		return 0, fmt.Errorf("insert prediction: %w", err)
	}

	const insertItem = `
INSERT INTO prediction_items (
    prediction_id, fixture_id, league_id, home_team_id, away_team_id,
    match_starting_at, market_name, market_key, selection, odds,
    bookmaker_name, actual_score_ft, actual_score_ht, settlement_status,
    settlement_details, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13, $14,
    $15, NOW(), NOW()
) RETURNING id`

	for i := range p.Items {
		item := &p.Items[i]
		if item.SettlementStatus == "" {
			item.SettlementStatus = StatusPending
		}
		if item.BookmakerName == "" {
			item.BookmakerName = "Pinnacle"
		}

		var itemID int64
		err = tx.QueryRow(ctx, insertItem,
			predID, item.FixtureID, item.LeagueID, item.HomeTeamID, item.AwayTeamID,
			item.MatchStartingAt, item.MarketName, string(item.MarketKey), item.Selection, item.Odds,
			item.BookmakerName, item.ActualScoreFT, item.ActualScoreHT, string(item.SettlementStatus),
			item.SettlementDetails,
		).Scan(&itemID)
		if err != nil {
			return 0, fmt.Errorf("insert prediction item %d: %w", i, err)
		}
		item.ID = itemID
		item.PredictionID = predID
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	p.ID = predID
	return predID, nil
}

// GetPredictionByID возвращает карточку прогноза со всеми исходами.
func (s *PGStorage) GetPredictionByID(ctx context.Context, id int64) (*Prediction, error) {
	const queryPred = `
SELECT 
    id, bet_type, title, COALESCE(analysis_text, ''), total_odds,
    COALESCE(confidence, 0), COALESCE(value_edge, 0), recommended_stake,
    status, profit_units, settled_at, publication_id, created_at, updated_at
FROM predictions
WHERE id = $1`

	var (
		p       Prediction
		betType string
		status  string
	)

	err := s.pool.QueryRow(ctx, queryPred, id).Scan(
		&p.ID, &betType, &p.Title, &p.AnalysisText, &p.TotalOdds,
		&p.Confidence, &p.ValueEdge, &p.RecommendedStake,
		&status, &p.ProfitUnits, &p.SettledAt, &p.PublicationID,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get prediction %d: %w", id, err)
	}
	p.BetType = BetType(betType)
	p.Status = BetStatus(status)

	items, err := s.getItemsByPredictionID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Items = items

	return &p, nil
}

func (s *PGStorage) getItemsByPredictionID(ctx context.Context, predID int64) ([]PredictionItem, error) {
	const queryItems = `
SELECT 
    id, prediction_id, fixture_id, league_id, home_team_id, away_team_id,
    match_starting_at, market_name, market_key, selection, odds,
    COALESCE(bookmaker_name, 'Pinnacle'), COALESCE(actual_score_ft, ''),
    COALESCE(actual_score_ht, ''), settlement_status, COALESCE(settlement_details, ''),
    created_at, updated_at
FROM prediction_items
WHERE prediction_id = $1
ORDER BY id ASC`

	rows, err := s.pool.Query(ctx, queryItems, predID)
	if err != nil {
		return nil, fmt.Errorf("query prediction items: %w", err)
	}
	defer rows.Close()

	var items []PredictionItem
	for rows.Next() {
		var (
			it        PredictionItem
			marketKey string
			status    string
		)
		err = rows.Scan(
			&it.ID, &it.PredictionID, &it.FixtureID, &it.LeagueID, &it.HomeTeamID, &it.AwayTeamID,
			&it.MatchStartingAt, &it.MarketName, &marketKey, &it.Selection, &it.Odds,
			&it.BookmakerName, &it.ActualScoreFT, &it.ActualScoreHT, &status, &it.SettlementDetails,
			&it.CreatedAt, &it.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan prediction item: %w", err)
		}
		it.MarketKey = MarketKey(marketKey)
		it.SettlementStatus = BetStatus(status)
		items = append(items, it)
	}

	return items, nil
}

// ListPendingPredictions возвращает все незавершённые прогнозы для settlement-воркера.
func (s *PGStorage) ListPendingPredictions(ctx context.Context) ([]Prediction, error) {
	const query = `
SELECT id FROM predictions
WHERE status = 'pending'
ORDER BY created_at ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list pending predictions: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan pending id: %w", err)
		}
		ids = append(ids, id)
	}

	var result []Prediction
	for _, id := range ids {
		p, err := s.GetPredictionByID(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, *p)
	}

	return result, nil
}

// SettleItem обновляет статус одного исхода.
func (s *PGStorage) SettleItem(ctx context.Context, itemID int64, status BetStatus, ftScore, htScore, details string) error {
	const query = `
UPDATE prediction_items
SET settlement_status = $2,
    actual_score_ft = $3,
    actual_score_ht = $4,
    settlement_details = $5,
    updated_at = NOW()
WHERE id = $1`

	res, err := s.pool.Exec(ctx, query, itemID, string(status), ftScore, htScore, details)
	if err != nil {
		return fmt.Errorf("settle item %d: %w", itemID, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("prediction item %d not found", itemID)
	}
	return nil
}

// SettlePrediction обновляет статус и прибыль всей карточки прогноза.
func (s *PGStorage) SettlePrediction(ctx context.Context, id int64, status BetStatus, profitUnits float64) error {
	const query = `
UPDATE predictions
SET status = $2,
    profit_units = $3,
    settled_at = NOW(),
    updated_at = NOW()
WHERE id = $1`

	res, err := s.pool.Exec(ctx, query, id, string(status), profitUnits)
	if err != nil {
		return fmt.Errorf("settle prediction %d: %w", id, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("prediction %d not found", id)
	}
	return nil
}

// GetPerformanceReport выгружает все завершённые ставки и рассчитывает метрики WinRate, ROI, Yield.
func (s *PGStorage) GetPerformanceReport(ctx context.Context) (*PerformanceReport, error) {
	const query = `
SELECT 
    id, bet_type, title, COALESCE(analysis_text, ''), total_odds,
    COALESCE(confidence, 0), COALESCE(value_edge, 0), recommended_stake,
    status, profit_units, settled_at, publication_id, created_at, updated_at
FROM predictions
WHERE status IN ('won', 'lost', 'refund')
ORDER BY settled_at ASC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query settled predictions: %w", err)
	}
	defer rows.Close()

	var settled []Prediction
	for rows.Next() {
		var (
			p       Prediction
			betType string
			status  string
		)
		err = rows.Scan(
			&p.ID, &betType, &p.Title, &p.AnalysisText, &p.TotalOdds,
			&p.Confidence, &p.ValueEdge, &p.RecommendedStake,
			&status, &p.ProfitUnits, &p.SettledAt, &p.PublicationID,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan settled prediction: %w", err)
		}
		p.BetType = BetType(betType)
		p.Status = BetStatus(status)
		settled = append(settled, p)
	}

	report := CalculatePerformance(settled)
	return &report, nil
}
