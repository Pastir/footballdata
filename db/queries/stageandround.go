package queries

import (
	"context"
	"github.com/Pastir/footballdata/db"
	"os"
)

// GetMatchNextRound - возвращает матчи двух ближайших туров
// Если это поиск и генерация типа "trend", возвращает матчи которые is_visible not true
func GetMatchNextRound(leagueID int, seasonId int, types string) []db.ListMatches {

	var listMatches []db.ListMatches
	var WhereString string

	if types == "trend" {
		WhereString = " AND NOT is_visible"
	}
	query := `SELECT mcm.id
                     ,mcm.league_id
                     ,mcm.season_id
                     ,mcr.id AS round_id
                     ,mcm.stage_id
                     ,mcm.home_team_id
                     ,mcm.away_team_id
                     ,CASE WHEN mt_h.shortname ISNULL THEN mt_h.longname ELSE mt_h.shortname END AS home_team_name
                     ,CASE WHEN mt_a.shortname ISNULL THEN mt_a.longname ELSE mt_a.shortname END AS away_team_name
                     ,jsonb_build_array(mc_tl.altname_1, mc_tl.altname_2, mc_tl.altname_3)       AS league_name
              FROM matchcenter_match AS mcm LEFT JOIN (
                                                        SELECT id
                                                               ,season_id
                                                        FROM matchcenter_rounds
                                                        WHERE start_date > NOW()
                                                          AND season_id = $1
                                                        ORDER BY start_date
                                                        LIMIT 2
                                                      ) AS mcr ON(mcm.round_id = mcr.id AND mcm.season_id = mcr.season_id)
                                            LEFT JOIN matchcenter_team AS mct_h ON (mcm.home_team_id = mct_h.id)
                                            LEFT JOIN matchcenter_translation mt_h ON (mct_h.uid = mt_h.uid)
                                            LEFT JOIN matchcenter_team AS mct_a ON (mcm.away_team_id = mct_a.id)
                                            LEFT JOIN matchcenter_translation mt_a ON (mct_a.uid = mt_a.uid)
                                            LEFT JOIN matchcenter_league AS mc_l ON (mcm.league_id = mc_l.id)
                                            LEFT JOIN matchcenter_translation AS mc_tl ON (mc_l.uid = mc_tl.uid)
              WHERE mcm.season_id = mcr.season_id AND mcm.round_id = mcr.id` + WhereString

	//Выполняем запрос
	rows, err := db.DBx.PGxPool.Query(context.Background(), query, seasonId)
	if err != nil {
		db.PGxConnPanic("Init MatchNextRound", "GetMatchNextRound", err)
		os.Exit(1)
	}

	for rows.Next() {
		var lm db.ListMatches

		err = rows.Scan(&lm.MatchId, &lm.LeagueId, &lm.SeasonId, &lm.RoundId, &lm.StageId, &lm.HomeTeamId,
			&lm.AwayTeamId, &lm.HomeTeamName, &lm.AwayTeamName, &lm.LeagueName)
		if err != nil {
			db.PGxConnPanic("Init MatchNextRound", "GetMatchNextRound", err)
			os.Exit(1)
		}

		listMatches = append(listMatches, lm)
	}

	return listMatches
}
