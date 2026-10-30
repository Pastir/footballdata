package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// H2HMatchItem описывает один очный матч между соперниками.
type H2HMatchItem struct {
	FixtureID    int64      `json:"fixture_id"`
	StartingAt   time.Time  `json:"starting_at"`
	DateStr      string     `json:"date_str"`
	LeagueName   string     `json:"league_name"`
	HomeTeamID   int64      `json:"home_team_id"`
	HomeTeamName string     `json:"home_team_name"`
	HomeTeamLogo string     `json:"home_team_logo"`
	AwayTeamID   int64      `json:"away_team_id"`
	AwayTeamName string     `json:"away_team_name"`
	AwayTeamLogo string     `json:"away_team_logo"`
	HomeScore    int        `json:"home_score"`
	AwayScore    int        `json:"away_score"`
	HomeXG       *float64   `json:"home_xg,omitempty"`
	AwayXG       *float64   `json:"away_xg,omitempty"`
}

// H2HSummary содержит агрегированную статистику личных встреч относительно текущего хозяина поля.
type H2HSummary struct {
	TotalMatches int     `json:"total_matches"`
	HomeWins     int     `json:"home_wins"` // победы команды, являющейся домашней в предстоящей игре
	Draws        int     `json:"draws"`
	AwayWins     int     `json:"away_wins"` // победы команды, являющейся гостевой в предстоящей игре
	HomeGoals    int     `json:"home_goals"`
	AwayGoals    int     `json:"away_goals"`
	AvgGoals     float64 `json:"avg_goals"`
}

// H2HData объединяет сводку и список недавних очных матчей.
type H2HData struct {
	Summary H2HSummary     `json:"summary"`
	Matches []H2HMatchItem `json:"matches"`
}

// GetH2HMatches извлекает из БД до limit последних очных матчей между командами homeTeamID и awayTeamID.
func GetH2HMatches(ctx context.Context, pool *pgxpool.Pool, homeTeamID, awayTeamID int64, locale string, limit int) (H2HData, error) {
	var result H2HData
	if pool == nil {
		return result, fmt.Errorf("database pool is not initialized")
	}
	if homeTeamID == 0 || awayTeamID == 0 {
		return result, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if locale == "" {
		locale = "ru"
	}

	query := `
		SELECT pf.id AS fixture_id,
		       pf.starting_at,
		       pl.name AS league_name,
		       pth.id AS home_id,
		       COALESCE(
		           NULLIF(pth.locale->$3->>'short', ''),
		           NULLIF(pth.locale->$3->>'long', ''),
		           pth.name
		       ) AS home_name,
		       COALESCE(pth.image_path, '') AS home_logo,
		       pta.id AS away_id,
		       COALESCE(
		           NULLIF(pta.locale->$3->>'short', ''),
		           NULLIF(pta.locale->$3->>'long', ''),
		           pta.name
		       ) AS away_name,
		       COALESCE(pta.image_path, '') AS away_logo,
		       COALESCE(sh.goals, 0) AS home_score,
		       COALESCE(sa.goals, 0) AS away_score,
		       xgh.value AS home_xg,
		       xga.value AS away_xg
		FROM provider_fixtures pf
		JOIN provider_leagues pl ON pl.id = pf.league_id
		JOIN provider_fixtures_participants fph ON fph.fixture_id = pf.id AND fph.location = 'home'
		JOIN provider_teams pth ON pth.id = fph.participant_id
		JOIN provider_fixtures_participants fpa ON fpa.fixture_id = pf.id AND fpa.location = 'away'
		JOIN provider_teams pta ON pta.id = fpa.participant_id
		LEFT JOIN provider_fixtures_scores sh ON sh.fixture_id = pf.id AND sh.participant_id = pth.id AND sh.type_id = 1525
		LEFT JOIN provider_fixtures_scores sa ON sa.fixture_id = pf.id AND sa.participant_id = pta.id AND sa.type_id = 1525
		LEFT JOIN provider_expected_fixtures xgh ON xgh.fixture_id = pf.id AND xgh.participant_id = pth.id AND xgh.type_id = 5304
		LEFT JOIN provider_expected_fixtures xga ON xga.fixture_id = pf.id AND xga.participant_id = pta.id AND xga.type_id = 5304
		WHERE ((fph.participant_id = $1 AND fpa.participant_id = $2) OR (fph.participant_id = $2 AND fpa.participant_id = $1))
		  AND pf.starting_at < NOW()
		  AND sh.goals IS NOT NULL
		ORDER BY pf.starting_at DESC
		LIMIT $4;
	`

	rows, err := pool.Query(ctx, query, homeTeamID, awayTeamID, locale, limit)
	if err != nil {
		return result, fmt.Errorf("GetH2HMatches query: %w", err)
	}
	defer rows.Close()

	var matches []H2HMatchItem
	var totalGoals int
	var homeWins, draws, awayWins int
	var homeGoals, awayGoals int

	monthsRu := map[time.Month]string{
		time.January: "янв", time.February: "фев", time.March: "мар",
		time.April: "апр", time.May: "мая", time.June: "июн",
		time.July: "июл", time.August: "авг", time.September: "сен",
		time.October: "окт", time.November: "ноя", time.December: "дек",
	}

	for rows.Next() {
		var m H2HMatchItem
		if err := rows.Scan(
			&m.FixtureID,
			&m.StartingAt,
			&m.LeagueName,
			&m.HomeTeamID,
			&m.HomeTeamName,
			&m.HomeTeamLogo,
			&m.AwayTeamID,
			&m.AwayTeamName,
			&m.AwayTeamLogo,
			&m.HomeScore,
			&m.AwayScore,
			&m.HomeXG,
			&m.AwayXG,
		); err != nil {
			return result, fmt.Errorf("GetH2HMatches scan: %w", err)
		}

		m.DateStr = fmt.Sprintf("%d %s %d", m.StartingAt.Day(), monthsRu[m.StartingAt.Month()], m.StartingAt.Year())
		matches = append(matches, m)

		totalGoals += m.HomeScore + m.AwayScore

		// Рассчитываем победы/голы относительно homeTeamID предстоящего матча
		var teamHomeGoalsInPast, teamAwayGoalsInPast int
		if m.HomeTeamID == homeTeamID {
			teamHomeGoalsInPast = m.HomeScore
			teamAwayGoalsInPast = m.AwayScore
		} else {
			teamHomeGoalsInPast = m.AwayScore
			teamAwayGoalsInPast = m.HomeScore
		}

		homeGoals += teamHomeGoalsInPast
		awayGoals += teamAwayGoalsInPast

		if teamHomeGoalsInPast > teamAwayGoalsInPast {
			homeWins++
		} else if teamAwayGoalsInPast > teamHomeGoalsInPast {
			awayWins++
		} else {
			draws++
		}
	}

	result.Matches = matches
	result.Summary.TotalMatches = len(matches)
	result.Summary.HomeWins = homeWins
	result.Summary.Draws = draws
	result.Summary.AwayWins = awayWins
	result.Summary.HomeGoals = homeGoals
	result.Summary.AwayGoals = awayGoals
	if len(matches) > 0 {
		result.Summary.AvgGoals = float64(totalGoals) / float64(len(matches))
	}

	return result, nil
}
