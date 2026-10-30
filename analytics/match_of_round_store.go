package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// MatchOfRoundRecord — запись матча тура, сохранённая в analytics_match_of_round.
type MatchOfRoundRecord struct {
	FixtureID    int64
	TeamHomeName string
	TeamAwayName string

	StrengthScore   float64
	FormScore       float64
	BaseScore       float64
	ImportanceScore float64
	ImportanceBonus float64
	FinalScore      float64

	Reasons []string

	ComputedAt time.Time
}

// GetOrComputeMatchOfRound возвращает матч тура из кэша.
// Если кэша нет или он инвалидирован — запускает алгоритм, сохраняет и возвращает результат.
func GetOrComputeMatchOfRound(
	ctx context.Context,
	pool *pgxpool.Pool,
	info *LeagueSeasonInfo,
	matches []MatchCandidate,
) (MatchOfRoundResult, error) {
	// 1. Пробуем взять из кэша
	cached, found, err := GetCachedMatchOfRound(ctx, pool, info.LeagueID, info.CurrRoundID)
	if err != nil {
		return MatchOfRoundResult{}, fmt.Errorf("GetOrComputeMatchOfRound cache lookup: %w", err)
	}
	if found {
		return cached, nil
	}

	// 2. Вычисляем
	ranked, err := MatchOfRound(ctx, pool, info, matches)
	if err != nil {
		return MatchOfRoundResult{}, fmt.Errorf("GetOrComputeMatchOfRound compute: %w", err)
	}
	if len(ranked) == 0 {
		return MatchOfRoundResult{}, fmt.Errorf("no ranked matches for league %d round %d",
			info.LeagueID, info.CurrRoundID)
	}

	best := ranked[0]

	// 3. Сохраняем асинхронно (ошибка не критична для текущего ответа)
	if saveErr := SaveMatchOfRound(ctx, pool, info, best); saveErr != nil {
		// Логируем, но не прерываем выполнение — кэш опционален
		_ = saveErr
	}

	return best, nil
}

// GetCachedMatchOfRound возвращает сохранённый результат матча тура.
// found=false означает: кэш не найден или инвалидирован.
func GetCachedMatchOfRound(
	ctx context.Context,
	pool *pgxpool.Pool,
	leagueID, roundID int,
) (result MatchOfRoundResult, found bool, err error) {
	query := `
		SELECT fixture_id,
		       team_home_name,
		       team_away_name,
		       strength_score,
		       form_score,
		       base_score,
		       importance_score,
		       importance_bonus,
		       final_score,
		       COALESCE(reasons, '{}')
		FROM analytics_match_of_round
		WHERE league_id      = $1
		  AND round_id       = $2
		  AND invalidated_at IS NULL
		LIMIT 1`

	var reasons []string
	row := pool.QueryRow(ctx, query, leagueID, roundID)
	if err = row.Scan(
		&result.FixtureID,
		&result.TeamHome,
		&result.TeamAway,
		&result.StrengthScore,
		&result.FormScore,
		&result.BaseScore,
		&result.ImportanceScore,
		&result.ImportanceBonus,
		&result.FinalScore,
		&reasons,
	); err != nil {
		// pgx возвращает pgx.ErrNoRows — не критично
		return MatchOfRoundResult{}, false, nil
	}
	result.Reasons = reasons
	return result, true, nil
}

// SaveMatchOfRound сохраняет результат выбора матча тура.
// При повторном вызове для той же пары league_id+round_id обновляет запись (UPSERT).
func SaveMatchOfRound(
	ctx context.Context,
	pool *pgxpool.Pool,
	info *LeagueSeasonInfo,
	result MatchOfRoundResult,
) error {
	query := `
		INSERT INTO analytics_match_of_round (
		    league_id, season_id, round_id,
		    fixture_id, team_home_name, team_away_name,
		    strength_score, form_score, base_score,
		    importance_score, importance_bonus, final_score,
		    reasons,
		    team_count, total_team_matches,
		    computed_at
		) VALUES (
		    $1, $2, $3,
		    $4, $5, $6,
		    $7, $8, $9,
		    $10, $11, $12,
		    $13,
		    $14, $15,
		    NOW()
		)
		ON CONFLICT (league_id, season_id, round_id)
		DO UPDATE SET
		    fixture_id        = EXCLUDED.fixture_id,
		    team_home_name    = EXCLUDED.team_home_name,
		    team_away_name    = EXCLUDED.team_away_name,
		    strength_score    = EXCLUDED.strength_score,
		    form_score        = EXCLUDED.form_score,
		    base_score        = EXCLUDED.base_score,
		    importance_score  = EXCLUDED.importance_score,
		    importance_bonus  = EXCLUDED.importance_bonus,
		    final_score       = EXCLUDED.final_score,
		    reasons           = EXCLUDED.reasons,
		    team_count        = EXCLUDED.team_count,
		    total_team_matches = EXCLUDED.total_team_matches,
		    computed_at       = NOW(),
		    invalidated_at    = NULL`

	_, err := pool.Exec(ctx, query,
		info.LeagueID, info.SeasonID, info.CurrRoundID,
		result.FixtureID, result.TeamHome, result.TeamAway,
		result.StrengthScore, result.FormScore, result.BaseScore,
		result.ImportanceScore, result.ImportanceBonus, result.FinalScore,
		result.Reasons,
		info.TeamCount, info.TotalTeamMatches,
	)
	if err != nil {
		return fmt.Errorf("SaveMatchOfRound: %w", err)
	}
	return nil
}

// InvalidateMatchOfRound помечает кэш как устаревший.
// Вызывается при обновлении standings коллектором или принудительно через CLI.
func InvalidateMatchOfRound(
	ctx context.Context,
	pool *pgxpool.Pool,
	leagueID, roundID int,
) error {
	query := `
		UPDATE analytics_match_of_round
		SET invalidated_at = NOW()
		WHERE league_id = $1
		  AND round_id  = $2
		  AND invalidated_at IS NULL`

	_, err := pool.Exec(ctx, query, leagueID, roundID)
	if err != nil {
		return fmt.Errorf("InvalidateMatchOfRound: %w", err)
	}
	return nil
}
