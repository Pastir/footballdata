package prediction

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

// DBFixtureScoreProvider реализует FixtureScoreProvider на базе PostgreSQL.
type DBFixtureScoreProvider struct {
	pool *pgxpool.Pool
}

// NewDBFixtureScoreProvider создаёт новый экземпляр провайдера счёта.
func NewDBFixtureScoreProvider(pool *pgxpool.Pool) *DBFixtureScoreProvider {
	return &DBFixtureScoreProvider{pool: pool}
}

// GetFinishedScore извлекает финальный и перерывный счёт матча из provider_fixtures_scores.
func (p *DBFixtureScoreProvider) GetFinishedScore(ctx context.Context, fixtureID int) (string, string, bool, error) {
	if p.pool == nil {
		return "", "", false, fmt.Errorf("database pool is nil")
	}

	query := `
SELECT pf.state_id,
       (SELECT goals FROM provider_fixtures_scores WHERE fixture_id = pf.id AND type_id = 1525 AND participant = 'home' LIMIT 1) AS ft_home,
       (SELECT goals FROM provider_fixtures_scores WHERE fixture_id = pf.id AND type_id = 1525 AND participant = 'away' LIMIT 1) AS ft_away,
       (SELECT goals FROM provider_fixtures_scores WHERE fixture_id = pf.id AND type_id = 1 AND participant = 'home' LIMIT 1) AS ht_home,
       (SELECT goals FROM provider_fixtures_scores WHERE fixture_id = pf.id AND type_id = 1 AND participant = 'away' LIMIT 1) AS ht_away
FROM provider_fixtures pf
WHERE pf.id = $1`

	var (
		stateID int
		ftHome  *int
		ftAway  *int
		htHome  *int
		htAway  *int
	)

	err := p.pool.QueryRow(ctx, query, fixtureID).Scan(&stateID, &ftHome, &ftAway, &htHome, &htAway)
	if err != nil {
		return "", "", false, fmt.Errorf("query fixture score %d: %w", fixtureID, err)
	}

	// Завершённые статусы: 5 (FT), 7 (AET), 8 (FT_PEN), 14 (WO), 17 (AWARDED)
	isFinished := (stateID == 5 || stateID == 7 || stateID == 8 || stateID == 14 || stateID == 17) && ftHome != nil && ftAway != nil
	if !isFinished {
		return "", "", false, nil
	}

	ftScore := fmt.Sprintf("%d:%d", *ftHome, *ftAway)
	htScore := ""
	if htHome != nil && htAway != nil {
		htScore = fmt.Sprintf("%d:%d", *htHome, *htAway)
	}

	return ftScore, htScore, true, nil
}
