package statistics

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/db/ch"
	"github.com/Pastir/footballdata/db/queries"
	"github.com/Pastir/footballdata/logs"
	"github.com/jackc/pgconn"
	"strings"
)

type Stats struct {
	ParticipantId   uint32  `ch:"participant_id"`
	TypeId          uint8   `ch:"type_id"`
	SeasonId        uint32  `ch:"season_id"`
	LeagueId        uint32  `ch:"league_id"`
	PlayedAll       uint64  `ch:"played_all"`
	PlayedHome      uint64  `ch:"played_home"`
	PlayedAway      uint64  `ch:"played_away"`
	HomeOwn         uint64  `ch:"home_own"`
	AvgHomeOwn      float64 `ch:"avg_home_own"`
	AwayOwn         uint64  `ch:"away_own"`
	AvgAwayOwn      float64 `ch:"avg_away_own"`
	OwnAll          uint64  `ch:"own_all"`
	AvgAllOwn       float64 `ch:"avg_all_own"`
	HomeOpponent    uint64  `ch:"home_opponent"`
	AvgHomeOpponent float64 `ch:"avg_home_opponent"`
	AwayOpponent    uint64  `ch:"away_opponent"`
	AvgAwayOpponent float64 `ch:"avg_away_opponent"`
	OpponentAll     uint64  `ch:"opponent_all"`
	AvgAllOpponent  float64 `ch:"avg_all_opponent"`
}

func StatsCorners() {
	var taskData config.TaskData
	// Get only current seasons
	taskData.SetCurrentSeason()
	queries.GetSeasons(taskData)
	seasonIDs := queries.GetStrSeasonsIds(queries.GetSeasons(taskData))

	//var seasonIDs []int
	chCtx := clickhouse.Context(context.Background())
	rows, err := ch.CHgo.Connect.Query(
		chCtx,
		`SELECT participant_id_away AS participant_id,
                      34 AS type_id,
                      ap.season_id,
                      ap.league_id,
                      played_away + played_home AS played_all,
                      played_home,
                      played_away,
                      corners_home_own AS home_own,
                      avg_corners_home_own AS avg_home_own,
                      corners_away_own AS away_own,
                      avg_corners_away_own AS avg_away_own,
                      corners_away_own + corners_home_own AS own_all,
                      round((corners_away_own + corners_home_own)/played_all, 2) AS avg_all_own,
                      corners_home_opponent AS home_opponent,
                      avg_corners_home_opponent AS avg_home_opponent,
                      corners_away_opponent AS away_opponent,
                      avg_corners_away_opponent AS avg_away_opponent,
                      corners_home_opponent + corners_away_opponent AS opponent_all,
                      round((corners_home_opponent + corners_away_opponent)/played_all, 2) AS avg_all_opponent
               FROM (
                      SELECT participant_id_away,
                             league_id,
                             season_id,
                             count(*) AS played_away,
                             sum(corners_away) AS corners_away_own,
                             round(avg(corners_away), 2) AS avg_corners_away_own,
                             sum(corners_home) AS corners_away_opponent,
                             round(avg(corners_home), 2) AS avg_corners_away_opponent,
                             corners_away_own + corners_away_opponent AS corners_away_all,
                             round(avg_corners_away_own + avg_corners_away_opponent, 2) AS avg_corners_away_all
                      FROM fixtures
                      WHERE fixtures.participant_id_away != 0  AND season_id IN ($1)
                      GROUP BY participant_id_away, league_id, season_id) AS ap
                   JOIN (
                      SELECT participant_id_home,
                             league_id,
                             season_id,
                             count(*) AS played_home,
                             sum(corners_home) AS corners_home_own,
                             round(avg(corners_home), 2) AS avg_corners_home_own,
                             sum(corners_away) AS corners_home_opponent,
                             round(avg(corners_away), 2) AS avg_corners_home_opponent,
                             corners_home_own + corners_home_opponent AS corners_home_all,
                             round(avg_corners_home_own + avg_corners_home_opponent, 2) AS avg_corners_home_all
                      FROM fixtures
                      WHERE  participant_id_home != 0 AND season_id IN ($1)
                      GROUP BY participant_id_home, league_id, season_id) AS hp ON (ap.participant_id_away = hp.participant_id_home
                                                                                        AND ap.league_id = hp.league_id
                                                                                        AND ap.season_id = hp.season_id)`,
		seasonIDs,
	)
	if err != nil {
		logs.ErrorLog(taskData.Name(), "StatsCorners", err)
		return
	}

	var insertQuery string
	// prepare batch data insert
	for rows.Next() {
		var s Stats
		err = rows.Scan(&s.ParticipantId,
			&s.TypeId,
			&s.SeasonId,
			&s.LeagueId,
			&s.PlayedAll,
			&s.PlayedHome,
			&s.PlayedAway,
			&s.HomeOwn,
			&s.AvgHomeOwn,
			&s.AwayOwn,
			&s.AvgAwayOwn,
			&s.OwnAll,
			&s.AvgAllOwn,
			&s.HomeOpponent,
			&s.AvgHomeOpponent,
			&s.AwayOpponent,
			&s.AvgAwayOpponent,
			&s.OpponentAll,
			&s.AvgAllOpponent,
		)
		if err != nil {
			logs.ErrorLog(taskData.Name(), "StatsCorners", err)
			return
		}
		rows.Close()

		insertQuery += fmt.Sprintf("(%d, %d, %d, %d, %d, %d, %d, %d, %f, %d, %f, %d, %f, %d, %f, %d, %f, %d, %f),",
			s.ParticipantId,
			s.TypeId,
			s.SeasonId,
			s.LeagueId,
			s.PlayedAll,
			s.PlayedHome,
			s.PlayedAway,
			s.HomeOwn,
			s.AvgHomeOwn,
			s.AwayOwn,
			s.AvgAwayOwn,
			s.OwnAll,
			s.AvgAllOwn,
			s.HomeOpponent,
			s.AvgHomeOpponent,
			s.AwayOpponent,
			s.AvgAwayOpponent,
			s.OpponentAll,
			s.AvgAllOpponent)
	}
	query := `INSERT INTO statistics_events (participant_id, type_id, season_id, league_id, played_all, played_home, 
                               played_away, home_own, avg_home_own, away_own, avg_away_own, all_own, avg_all_own, 
                               home_opponent, avg_home_opponent, away_opponent, avg_away_opponent, all_opponent, 
                               avg_all_opponent ) VALUES ` + strings.TrimRight(insertQuery, ",")

	_, err = db.DBx.PGxPool.Exec(context.Background(), query)
	if err != nil {
		db.PGxConnError(taskData.Name(), "StatsCorners", err.(*pgconn.PgError))
	}
	return
}
