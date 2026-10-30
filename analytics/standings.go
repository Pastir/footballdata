package analytics

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
)

// GetCurrentStandings возвращает полную турнирную таблицу тура со всей статистикой (W/D/L/GF/GA/GD),
// формой последних 5 матчей, динамикой (trend) и правилами зон в рамках единого SQL-запроса.
func GetCurrentStandings(ctx context.Context, pool *pgxpool.Pool, info *LeagueSeasonInfo) ([]Standing, error) {
	query := `WITH next_round AS (
    SELECT *
    FROM get_neighbor_rounds($1)
),
form AS (
    SELECT
        f.participant_id,
        f.form,
        f.rounds AS form_rounds,
        nr.last_round_id,
        nr.prev_round_id,
        nr.league_id,
        nr.season_id
    FROM next_round AS nr
    CROSS JOIN LATERAL get_form_5last_rounds(nr.season_id, nr.next_round_starting_at) AS f
)
SELECT ps.id,
       ps.position,
       COALESCE(ps_prev.position - ps.position, 0) AS trend,
       ps.participant_id,
       COALESCE(
           NULLIF(pt.locale->$4->>'short', ''),
           NULLIF(pt.locale->$4->>'long', ''),
           pt.name
       ) AS name,
       COALESCE(pt.image_path, '') AS image_path,
       COALESCE(ps.standing_rule_id, 0) AS standing_rule_id,
       COALESCE(psr.name, '') AS standings_position_rules_name,
       COALESCE(psr.developer_name, '') AS standings_position_rules_dev_name,
       form.form,
       COALESCE((details.details -> 'OVERALL_MATCHES' ->> 'value')::int, 0)         AS played,
       COALESCE((details.details -> 'OVERALL_WINS' ->> 'value')::int, 0)            AS wins,
       COALESCE((details.details -> 'OVERALL_DRAWS' ->> 'value')::int, 0)           AS draws,
       COALESCE((details.details -> 'OVERALL_LOST' ->> 'value')::int, 0)            AS lost,
       COALESCE((details.details -> 'OVERALL_SCORED' ->> 'value')::int, 0)          AS scored,
       COALESCE((details.details -> 'OVERALL_CONCEDED' ->> 'value')::int, 0)        AS conceded,
       COALESCE((details.details -> 'OVERALL_GOAL_DIFFERENCE' ->> 'value')::int, 0) AS goal_difference,
       COALESCE((details.details -> 'TOTAL_POINTS' ->> 'value')::int, ps.points)    AS points
FROM provider_standings ps
         JOIN provider_teams pt ON pt.id = ps.participant_id
         LEFT JOIN provider_standings_position_rules AS pspr ON (pspr.id = ps.standing_rule_id)
         LEFT JOIN provider_types psr ON (psr.id = pspr.type_id AND psr.model_type = 'standing_rule')
         LEFT JOIN form ON (form.participant_id = ps.participant_id)
         LEFT JOIN provider_standings AS ps_prev 
                ON (ps_prev.participant_id = ps.participant_id 
                    AND ps_prev.season_id = form.season_id 
                    AND ps_prev.round_id = (CASE WHEN $3::int > 0 THEN $3::int ELSE form.prev_round_id END))
         LEFT JOIN LATERAL (
    SELECT jsonb_object_agg(
                   provider_types.developer_name,
                   jsonb_build_object(
                           'type_id', psd_d.type_id,
                           'value', psd_d.value
                   )
           ) AS details
    FROM provider_standings_details psd_d
             LEFT JOIN provider_types ON (psd_d.type_id = provider_types.id)
    WHERE type_id IN (129, 130, 131, 132, 133, 134, 179, 187)
      AND psd_d.standing_id = ps.id
    ) AS details ON TRUE
WHERE ps.league_id = form.league_id
  AND ps.season_id = form.season_id
  AND ps.round_id = (CASE WHEN $2::int > 0 THEN $2::int ELSE form.last_round_id END)
ORDER BY ps.position;`

	locale := "ru"
	if info != nil && info.Locale != "" {
		locale = info.Locale
	}

	rows, err := pool.Query(ctx, query,
		info.SeasonID,
		info.CurrRoundID,
		info.PrevRoundID,
		locale,
	)
	if err != nil {
		return nil, fmt.Errorf("GetCurrentStandings query: %w", err)
	}
	defer rows.Close()

	var result []Standing
	for rows.Next() {
		var r Standing
		if err = rows.Scan(
			&r.ID, &r.Position, &r.Trend, &r.TeamID, &r.TeamName,
			&r.TeamLogoURL, &r.StandingRuleID, &r.Rule, &r.RuleDevName,
			&r.Form, &r.Played, &r.Won, &r.Drawn, &r.Lost,
			&r.GF, &r.GA, &r.GD, &r.Points,
		); err != nil {
			return nil, fmt.Errorf("GetCurrentStandings scan: %w", err)
		}
		result = append(result, r)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("GetCurrentStandings rows: %w", err)
	}
	return result, nil
}

// ZoneFromDeveloperName маппирует developer_name из provider_standings_rules → CSS zone class.
func ZoneFromDeveloperName(developerName string) string {
	dn := strings.ToUpper(developerName)
	switch {
	case strings.Contains(dn, "CHAMPION"):
		return "ucl"
	case strings.Contains(dn, "EUROPA"):
		return "uel"
	case strings.Contains(dn, "CONFERENCE"):
		return "uecl"
	case strings.Contains(dn, "RELEGAT"):
		return "relegation"
	default:
		return ""
	}
}

// GetKeyStats возвращает 3 карточки Key Stats: Most Goals / Best Defence / Best GD.
func GetKeyStats(rows []Standing, details ...map[int64]StandingDetails) []StatCard {
	if len(rows) == 0 {
		return nil
	}

	var dMap map[int64]StandingDetails
	if len(details) > 0 && details[0] != nil {
		dMap = details[0]
	}

	findBest := func(label string, f func(Standing) int, higher bool) StatCard {
		var bestTeam string
		bestVal := 0
		first := true
		for _, r := range rows {
			v := f(r)
			if first || (higher && v > bestVal) || (!higher && v < bestVal) {
				bestVal = v
				bestTeam = r.TeamName
				first = false
			}
		}
		return StatCard{Label: label, Team: bestTeam, Value: fmt.Sprintf("%d", bestVal)}
	}

	getGF := func(r Standing) int {
		if d, ok := dMap[r.ID]; ok && d.GF != 0 {
			return d.GF
		}
		return r.GF
	}
	getGA := func(r Standing) int {
		if d, ok := dMap[r.ID]; ok && d.GA != 0 {
			return d.GA
		}
		return r.GA
	}
	getGD := func(r Standing) int {
		if d, ok := dMap[r.ID]; ok && d.GD != 0 {
			return d.GD
		}
		return r.GD
	}

	return []StatCard{
		findBest("Most Goals", getGF, true),
		findBest("Best Defence", getGA, false),
		findBest("Best Goal Diff", getGD, true),
	}
}

// BuildInsight формирует детерминированный аналитический вывод из данных таблицы.
func BuildInsight(rows []Standing, details ...map[int64]StandingDetails) string {
	if len(rows) == 0 {
		return ""
	}

	var dMap map[int64]StandingDetails
	if len(details) > 0 && details[0] != nil {
		dMap = details[0]
	}

	leader := rows[0]

	bestAttackTeam := rows[0].TeamName
	bestAttackGF := 0
	bestDefenceTeam := rows[0].TeamName
	bestDefenceGA := -1

	for _, r := range rows {
		gf := r.GF
		ga := r.GA
		if d, ok := dMap[r.ID]; ok {
			gf = d.GF
			ga = d.GA
		}

		if gf > bestAttackGF {
			bestAttackGF = gf
			bestAttackTeam = r.TeamName
		}
		if bestDefenceGA < 0 || ga < bestDefenceGA {
			bestDefenceGA = ga
			bestDefenceTeam = r.TeamName
		}
	}

	leaderGF := leader.GF
	leaderGA := leader.GA
	if d, ok := dMap[leader.ID]; ok {
		leaderGF = d.GF
		leaderGA = d.GA
	}

	if bestAttackTeam == bestDefenceTeam {
		return fmt.Sprintf(
			"%s leads the table with %d pts, dominating both attack (%d goals) and defence (%d conceded).",
			leader.TeamName, leader.Points,
			leaderGF, leaderGA,
		)
	}

	return fmt.Sprintf(
		"%s leads with %d pts. %s has the best attack (%d goals scored), %s the best defence (%d conceded).",
		leader.TeamName, leader.Points,
		bestAttackTeam, bestAttackGF,
		bestDefenceTeam, bestDefenceGA,
	)
}
