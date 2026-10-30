package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// RoundClosingStatus описывает текущий статус завершения тура для принятия решения о публикации.
type RoundClosingStatus struct {
	LeagueID          int        `json:"league_id"`
	LeagueName        string     `json:"league_name"`
	SeasonID          int        `json:"season_id"`
	SeasonName        string     `json:"season_name"`
	RoundID           int        `json:"round_id"`
	RoundName         string     `json:"round_name"`
	StartingAt        time.Time  `json:"starting_at"`
	EndingAt          time.Time  `json:"ending_at"`
	TotalMatches      int        `json:"total_matches"`
	FinishedMatches   int        `json:"finished_matches"`
	PostponedMatches  int        `json:"postponed_matches"`
	InPlayMatches     int        `json:"in_play_matches"`
	UpcomingMatches   int        `json:"upcoming_matches"`
	LastMatchKickoff  *time.Time `json:"last_match_kickoff,omitempty"`
	IsReadyForClosing bool       `json:"is_ready_for_closing"`
	HasPostponements  bool       `json:"has_postponements"`
	Reason            string     `json:"reason"`
}

// EvaluateRoundClosing вычисляет готовность тура к подведению итогов и публикации в Telegram.
// Логика учитывает:
// 1. Недопустимость закрытия во время идущих матчей (InPlayMatches > 0).
// 2. Штатный сценарий: все матчи сыграны (или перенесены), и с момента свистка последнего матча прошёл буфер (~2ч 15мин от стартового свистка).
// 3. Fallback-сценарий (переносы): утро вторника (начиная с 08:30 UTC после ending_at). Если есть сыгранные матчи, тур закрывается с пометкой о переносах.
func EvaluateRoundClosing(status *RoundClosingStatus, now time.Time) {
	if status == nil {
		return
	}

	if status.TotalMatches == 0 {
		status.IsReadyForClosing = false
		status.Reason = "no matches found in round"
		return
	}

	// 1. Если прямо сейчас идёт матч — закрывать категорически нельзя
	if status.InPlayMatches > 0 {
		status.IsReadyForClosing = false
		status.Reason = fmt.Sprintf("%d match(es) currently in play", status.InPlayMatches)
		return
	}

	// 2. Если нет ни одного сыгранного матча — тур ещё не начинался
	if status.FinishedMatches == 0 {
		status.IsReadyForClosing = false
		status.Reason = "no matches finished yet"
		return
	}

	// Флаг наличия перенесённых матчей
	status.HasPostponements = status.PostponedMatches > 0

	// 3. Штатный сценарий: все доступные матчи завершены (Finished + Postponed == Total)
	if status.FinishedMatches+status.PostponedMatches == status.TotalMatches {
		// Проверяем временной буфер от стартового свистка последнего матча:
		// 105 минут основного времени + 15 минут перерыв/интервью = 2 часа 00 минут
		bufferDuration := 2 * time.Hour
		if status.LastMatchKickoff != nil && now.Before(status.LastMatchKickoff.Add(bufferDuration)) {
			status.IsReadyForClosing = false
			status.Reason = fmt.Sprintf("waiting match conclusion buffer until %s", status.LastMatchKickoff.Add(bufferDuration).Format(time.RFC3339))
			return
		}

		status.IsReadyForClosing = true
		if status.HasPostponements {
			status.Reason = fmt.Sprintf("all available matches completed (%d finished, %d postponed)", status.FinishedMatches, status.PostponedMatches)
		} else {
			status.Reason = fmt.Sprintf("clean finish: all %d matches completed", status.TotalMatches)
		}
		return
	}

	// 4. Fallback-сценарий (перенесённые матчи, не начавшиеся вовремя, или ожидающие переноса):
	// В топ-лигах ending_at обычно приходится на воскресенье или понедельник.
	// Начиная с утра вторника (08:30 UTC следующего дня после ending_at), если тур завис из-за несыгранных матчей,
	// мы принудительно закрываем его по сыгранным матчам, чтобы лента и Mini App не блокировались.
	deadline := time.Date(
		status.EndingAt.Year(),
		status.EndingAt.Month(),
		status.EndingAt.Day(),
		8, 30, 0, 0,
		time.UTC,
	).AddDate(0, 0, 1) // утро вторника после ending_at понедельника

	if now.After(deadline) {
		status.IsReadyForClosing = true
		status.HasPostponements = true
		status.Reason = fmt.Sprintf("fallback deadline reached (%s): %d finished, %d pending/postponed",
			deadline.Format("2006-01-02 15:04 UTC"), status.FinishedMatches, status.UpcomingMatches+status.PostponedMatches)
		return
	}

	// 5. Тур ещё в процессе ожидания оставшихся матчей
	status.IsReadyForClosing = false
	status.Reason = fmt.Sprintf("waiting remaining matches: %d finished, %d upcoming, %d postponed (deadline: %s)",
		status.FinishedMatches, status.UpcomingMatches, status.PostponedMatches, deadline.Format("2006-01-02 15:04 UTC"))
}

// GetRoundClosingStatus загружает данные о матчах активного тура лиги и вычисляет готовность к закрытию.
func GetRoundClosingStatus(ctx context.Context, pool *pgxpool.Pool, leagueID int, now time.Time) (*RoundClosingStatus, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	// Выбираем текущий или последний сыгранный тур текущего сезона лиги
	queryRound := `
SELECT pl.id AS league_id,
       pl.name AS league_name,
       ps.id AS season_id,
       ps.name AS season_name,
       pr.id AS round_id,
       pr.name AS round_name,
       pr.starting_at,
       pr.ending_at
FROM provider_leagues pl
JOIN provider_seasons ps ON ps.league_id = pl.id AND ps.is_current = true
JOIN provider_rounds pr ON pr.season_id = ps.id
WHERE pl.id = $1
  AND (pr.is_current = true OR pr.ending_at <= $2::date)
ORDER BY pr.name::int DESC
LIMIT 1`

	var status RoundClosingStatus
	err := pool.QueryRow(ctx, queryRound, leagueID, now.Format("2006-01-02")).Scan(
		&status.LeagueID,
		&status.LeagueName,
		&status.SeasonID,
		&status.SeasonName,
		&status.RoundID,
		&status.RoundName,
		&status.StartingAt,
		&status.EndingAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query active round for league %d: %w", leagueID, err)
	}

	// Агрегируем статусы матчей раунда
	queryFixtures := `
SELECT COUNT(f.id) AS total_count,
       COUNT(f.id) FILTER (WHERE f.state_id IN (5, 7, 8, 14, 17)) AS finished_count,
       COUNT(f.id) FILTER (WHERE f.state_id IN (10, 11, 12, 15)) AS postponed_count,
       COUNT(f.id) FILTER (WHERE f.state_id IN (2, 3, 4, 6, 9, 16, 18, 21, 22, 23, 25)) AS in_play_count,
       COUNT(f.id) FILTER (WHERE f.state_id IN (1, 13, 26)) AS upcoming_count,
       MAX(f.starting_at) FILTER (WHERE f.state_id IN (5, 7, 8, 14, 17)) AS last_match_kickoff
FROM provider_fixtures f
WHERE f.round_id = $1`

	err = pool.QueryRow(ctx, queryFixtures, status.RoundID).Scan(
		&status.TotalMatches,
		&status.FinishedMatches,
		&status.PostponedMatches,
		&status.InPlayMatches,
		&status.UpcomingMatches,
		&status.LastMatchKickoff,
	)
	if err != nil {
		return nil, fmt.Errorf("query fixtures for round %d: %w", status.RoundID, err)
	}

	// Вычисляем готовность
	EvaluateRoundClosing(&status, now)

	return &status, nil
}
