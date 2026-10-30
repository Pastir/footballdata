package analytics

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

// TeamProfilePeriod содержит рассчитанные средние метрики команды за определенный период.
type TeamProfilePeriod struct {
	MatchesCount     int     `json:"matches_count"`
	XGPerMatch       float64 `json:"xg_per_match"`
	XGAPerMatch      float64 `json:"xga_per_match"`
	GoalsScoredAvg   float64 `json:"goals_scored_avg"`
	GoalsConcededAvg float64 `json:"goals_conceded_avg"`
	BTTSPercent      int     `json:"btts_percent"`
	Over25Percent    int     `json:"over25_percent"`
	ShotsOnTargetAvg float64 `json:"shots_on_target_avg"`
	PossessionAvg    float64 `json:"possession_avg"`
}

// TeamPreMatchStats содержит профиль команды в двух временных срезах: последние 5 матчей и весь сезон.
type TeamPreMatchStats struct {
	TeamID   int64             `json:"team_id"`
	TeamName string            `json:"team_name"`
	Last5    TeamProfilePeriod `json:"last_5"`
	Season   TeamProfilePeriod `json:"season"`
}

// PreMatchStatsComparison объединяет профили обеих команд для экрана предматчевого сравнения.
type PreMatchStatsComparison struct {
	Home TeamPreMatchStats `json:"home"`
	Away TeamPreMatchStats `json:"away"`
}

type matchRawStatRow struct {
	goalsFor      int
	goalsAgainst  int
	xgFor         *float64
	xgAgainst     *float64
	shotsOnTarget *float64
	possession    *float64
}

// GetTeamPreMatchProfile рассчитывает статистический профиль команды за текущий сезон и последние 5 матчей.
func GetTeamPreMatchProfile(ctx context.Context, pool *pgxpool.Pool, teamID int64, seasonID int) (TeamPreMatchStats, error) {
	var profile TeamPreMatchStats
	profile.TeamID = teamID
	if pool == nil {
		return profile, fmt.Errorf("database pool is not initialized")
	}
	if teamID == 0 || seasonID == 0 {
		return profile, nil
	}

	query := `
		SELECT COALESCE(s_for.goals, 0) AS goals_for,
		       COALESCE(s_against.goals, 0) AS goals_against,
		       xg_for.value AS xg_for,
		       xg_against.value AS xg_against,
		       pfs_sot.value AS shots_on_target,
		       pfs_poss.value AS possession
		FROM provider_fixtures pf
		JOIN provider_fixtures_participants pfp ON pfp.fixture_id = pf.id AND pfp.participant_id = $1
		JOIN provider_fixtures_participants pfp_opp ON pfp_opp.fixture_id = pf.id AND pfp_opp.participant_id != $1
		LEFT JOIN provider_fixtures_scores s_for ON s_for.fixture_id = pf.id AND s_for.participant_id = $1 AND s_for.type_id = 1525
		LEFT JOIN provider_fixtures_scores s_against ON s_against.fixture_id = pf.id AND s_against.participant_id = pfp_opp.participant_id AND s_against.type_id = 1525
		LEFT JOIN provider_expected_fixtures xg_for ON xg_for.fixture_id = pf.id AND xg_for.participant_id = $1 AND xg_for.type_id = 5304
		LEFT JOIN provider_expected_fixtures xg_against ON xg_against.fixture_id = pf.id AND xg_against.participant_id = pfp_opp.participant_id AND xg_against.type_id = 5304
		LEFT JOIN provider_fixtures_statistics pfs_sot ON pfs_sot.fixture_id = pf.id AND pfs_sot.participant_id = $1 AND pfs_sot.type_id = 86
		LEFT JOIN provider_fixtures_statistics pfs_poss ON pfs_poss.fixture_id = pf.id AND pfs_poss.participant_id = $1 AND pfs_poss.type_id = 45
		WHERE pf.season_id = $2
		  AND pf.starting_at < NOW()
		  AND s_for.goals IS NOT NULL
		ORDER BY pf.starting_at DESC;
	`

	rows, err := pool.Query(ctx, query, teamID, seasonID)
	if err != nil {
		return profile, fmt.Errorf("GetTeamPreMatchProfile query: %w", err)
	}
	defer rows.Close()

	var allMatches []matchRawStatRow
	for rows.Next() {
		var r matchRawStatRow
		if err := rows.Scan(
			&r.goalsFor,
			&r.goalsAgainst,
			&r.xgFor,
			&r.xgAgainst,
			&r.shotsOnTarget,
			&r.possession,
		); err != nil {
			return profile, fmt.Errorf("GetTeamPreMatchProfile scan: %w", err)
		}
		allMatches = append(allMatches, r)
	}

	profile.Season = computePeriodStats(allMatches)

	// Последние 5 матчей
	last5Count := len(allMatches)
	if last5Count > 5 {
		last5Count = 5
	}
	profile.Last5 = computePeriodStats(allMatches[:last5Count])

	return profile, nil
}

func computePeriodStats(matches []matchRawStatRow) TeamProfilePeriod {
	var p TeamProfilePeriod
	p.MatchesCount = len(matches)
	if len(matches) == 0 {
		return p
	}

	var totalGF, totalGA int
	var totalXG, totalXGA float64
	var countXG, countXGA int
	var bttsCount, over25Count int
	var totalSOT, totalPoss float64
	var countSOT, countPoss int

	for _, m := range matches {
		totalGF += m.goalsFor
		totalGA += m.goalsAgainst

		if m.goalsFor > 0 && m.goalsAgainst > 0 {
			bttsCount++
		}
		if (m.goalsFor + m.goalsAgainst) > 2 {
			over25Count++
		}

		if m.xgFor != nil {
			totalXG += *m.xgFor
			countXG++
		}
		if m.xgAgainst != nil {
			totalXGA += *m.xgAgainst
			countXGA++
		}

		if m.shotsOnTarget != nil {
			totalSOT += *m.shotsOnTarget
			countSOT++
		}
		if m.possession != nil {
			totalPoss += *m.possession
			countPoss++
		}
	}

	n := float64(len(matches))
	p.GoalsScoredAvg = float64(totalGF) / n
	p.GoalsConcededAvg = float64(totalGA) / n
	p.BTTSPercent = int((float64(bttsCount) / n) * 100.0)
	p.Over25Percent = int((float64(over25Count) / n) * 100.0)

	if countXG > 0 {
		p.XGPerMatch = totalXG / float64(countXG)
	}
	if countXGA > 0 {
		p.XGAPerMatch = totalXGA / float64(countXGA)
	}
	if countSOT > 0 {
		p.ShotsOnTargetAvg = totalSOT / float64(countSOT)
	}
	if countPoss > 0 {
		p.PossessionAvg = totalPoss / float64(countPoss)
	}

	return p
}
