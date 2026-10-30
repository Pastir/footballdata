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

type OverCorners struct {
	Id               uint64  `ch:"id"`
	StartingAt       string  `ch:"starting_at"`
	ParticipantId    uint32  `ch:"participant_id"`
	SeasonId         uint32  `ch:"season_id"`
	LeagueId         uint32  `ch:"league_id"`
	Location         string  `ch:"location"`
	MatchPlayed      uint64  `ch:"match_played"`
	MatchPlayedHome  uint64  `ch:"match_played_home"`
	MatchPlayedAway  uint64  `ch:"match_played_away"`
	Ov75All          uint64  `ch:"ov75_all"`
	Ov75Home         uint64  `ch:"ov75_home"`
	Ov75Away         uint64  `ch:"ov75_away"`
	PercentOv75All   float64 `ch:"percent_ov75_all"`
	PercentOv75Home  float64 `ch:"percent_ov75_home"`
	PercentOv75Away  float64 `ch:"percent_ov75_away"`
	Ov85All          uint64  `ch:"ov85_all"`
	Ov85Home         uint64  `ch:"ov85_home"`
	Ov85Away         uint64  `ch:"ov85_away"`
	PercentOv85All   float64 `ch:"percent_ov85_all"`
	PercentOv85Home  float64 `ch:"percent_ov85_home"`
	PercentOv85Away  float64 `ch:"percent_ov85_away"`
	Ov95All          uint64  `ch:"ov95_all"`
	Ov95Home         uint64  `ch:"ov95_home"`
	Ov95Away         uint64  `ch:"ov95_away"`
	PercentOv95All   float64 `ch:"percent_ov95_all"`
	PercentOv95Home  float64 `ch:"percent_ov95_home"`
	PercentOv95Away  float64 `ch:"percent_ov95_away"`
	Ov105All         uint64  `ch:"ov105_all"`
	Ov105Home        uint64  `ch:"ov105_home"`
	Ov105Away        uint64  `ch:"ov105_away"`
	PercentOv105All  float64 `ch:"percent_ov105_all"`
	PercentOv105Home float64 `ch:"percent_ov105_home"`
	PercentOv105Away float64 `ch:"percent_ov105_away"`
	Ov115All         uint64  `ch:"ov115_all"`
	Ov115Home        uint64  `ch:"ov115_home"`
	Ov115Away        uint64  `ch:"ov115_away"`
	PercentOv115All  float64 `ch:"percent_ov115_all"`
	PercentOv115Home float64 `ch:"percent_ov115_home"`
	PercentOv115Away float64 `ch:"percent_ov115_away"`
	Ov125All         uint64  `ch:"ov125_all"`
	Ov125Home        uint64  `ch:"ov125_home"`
	Ov125Away        uint64  `ch:"ov125_away"`
	PercentOv125All  float64 `ch:"percent_ov125_all"`
	PercentOv125Home float64 `ch:"percent_ov125_home"`
	PercentOv125Away float64 `ch:"percent_ov125_away"`
	Ov135All         uint64  `ch:"ov135_all"`
	Ov135Home        uint64  `ch:"ov135_home"`
	Ov135Away        uint64  `ch:"ov135_away"`
	PercentOv135All  float64 `ch:"percent_ov135_all"`
	PercentOv135Home float64 `ch:"percent_ov135_home"`
	PercentOv135Away float64 `ch:"percent_ov135_away"`
}

func HandlerOverCorners() {
	var taskData config.TaskData
	// Get only current seasons
	taskData.SetCurrentSeason()
	queries.GetSeasons(taskData)
	seasonIDs := queries.GetStrSeasonsIds(queries.GetSeasons(taskData))

	var oc []OverCorners

	//var seasonIDs []int
	chCtx := clickhouse.Context(context.Background())
	err := ch.CHgo.Connect.Select(
		chCtx,
		&oc,
		`SELECT id
       , toString(starting_at) AS starting_at
       , participant_id
       , season_id
       , league_id
       , location
       , match_played
       , match_played_home
       , match_played_away
       , ov75_all
       , ov75_home
       , ov75_away
       , round(ov75_all / match_played * 100, 2) AS percent_ov75_all
       , round(ov75_home / match_played_home * 100, 2) AS percent_ov75_home
       , round(ov75_away / match_played_away * 100, 2) AS percent_ov75_away
       , ov85_all
       , ov85_home
       , ov85_away
       , round(ov85_all / match_played * 100, 2) AS percent_ov85_all
       , round(ov85_home / match_played_home * 100, 2) AS percent_ov85_home
       , round(ov85_away / match_played_away * 100, 2) AS percent_ov85_away
       , ov95_all
       , ov95_home
       , ov95_away
       , round(ov95_all / match_played * 100, 2) AS percent_ov95_all
       , round(ov95_home / match_played_home * 100, 2) AS percent_ov95_home
       , round(ov95_away / match_played_away * 100, 2) AS percent_ov95_away
       , ov105_all
       , ov105_home
       , ov105_away
       , round(ov105_all / match_played * 100, 2) AS percent_ov105_all
       , round(ov105_home / match_played_home * 100, 2) AS percent_ov105_home
       , round(ov105_away / match_played_away * 100, 2) AS percent_ov105_away
       , ov115_all
       , ov115_home
       , ov115_away
       , round(ov115_all / match_played * 100, 2) AS percent_ov115_all
       , round(ov115_home / match_played_home * 100, 2) AS percent_ov115_home
       , round(ov115_away / match_played_away * 100, 2) AS percent_ov115_away
       , ov125_all
       , ov125_home
       , ov125_away
       , round(ov125_all / match_played * 100, 2) AS percent_ov125_all
       , round(ov125_home / match_played_home * 100, 2) AS percent_ov125_home
       , round(ov125_away / match_played_away * 100, 2) AS percent_ov125_away
       , ov135_all
       , ov135_home
       , ov135_away
       , round(ov135_all / match_played * 100, 2) AS percent_ov135_all
       , round(ov135_home / match_played_home * 100, 2) AS percent_ov135_home
       , round(ov135_away / match_played_away * 100, 2) AS percent_ov135_away

FROM (
       SELECT
           id
           , starting_at
           , participant_id
           , location
           , season_id
           , league_id
           , count(*) OVER (PARTITION BY participant_id, season_id ORDER BY starting_at ASC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS number_played_asc
           , count(*) OVER (PARTITION BY participant_id, season_id) AS match_played
           , countIf(id, location = 'home') OVER (PARTITION BY participant_id, season_id) AS match_played_home
           , countIf(id, location = 'away') OVER (PARTITION BY participant_id, season_id) AS match_played_away
           , sum(value) OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS corners_scored_all
           , sumIf(value, location = 'home') OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS corners_scored_home
           , sumIf(value, location = 'away') OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS corners_scored_away
           , countIf(id, location = 'home') OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS number_played_home
           , countIf(id, location = 'away') OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS number_played_away
           , round(avg(value) OVER (PARTITION BY participant_id, season_id), 2) AS avg_corners_per_match
           , round(avgIf(value, location = 'home') OVER (PARTITION BY participant_id, season_id), 2) AS avg_corners_per_match_home
           , round(avgIf(value, location = 'away') OVER (PARTITION BY participant_id, season_id), 2) AS avg_corners_per_match_away
           , countIf(id, value > 7.5) OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov75_all
           , countIf(id, location='home' and value > 7.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov75_home
           , countIf(id, location='away' and value > 7.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov75_away
           , countIf(id, value > 8.5) OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov85_all
           , countIf(id, location='home' and value > 8.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov85_home
           , countIf(id, location='away' and value > 8.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov85_away
           , countIf(id, value > 9.5) OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov95_all
           , countIf(id, location='home' and value > 9.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov95_home
           , countIf(id, location='away' and value > 9.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov95_away
           , countIf(id, value > 10.5) OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov105_all
           , countIf(id, location='home' and value > 10.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov105_home
           , countIf(id, location='away' and value > 10.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov105_away
           , countIf(id, value > 11.5) OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov115_all
           , countIf(id, location='home' and value > 11.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov115_home
           , countIf(id, location='away' and value > 11.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov115_away
           , countIf(id, value > 12.5) OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov125_all
           , countIf(id, location='home' and value > 12.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov125_home
           , countIf(id, location='away' and value > 12.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov125_away
           , countIf(id, value > 13.5) OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov135_all
           , countIf(id, location='home' and value > 13.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov135_home
           , countIf(id, location='away' and value > 13.5)  OVER (PARTITION BY participant_id, season_id ORDER BY starting_at DESC) AS ov135_away
       FROM fixtures_statistics
       WHERE type_id = 34  AND season_id = 19734) AS it WHERE number_played_asc = 1`, seasonIDs)
	if err != nil {
		logs.ErrorLog(taskData.Name(), "HandlerOverCorners", err)
		return
	}

	query := `INSERT INTO statistics_bk (participant_id, season_id, league_id, match_played, location, type_id, value, percent) VALUES `
	conflictQuery := ` ON CONFLICT (participant_id, season_id, type_id) DO UPDATE SET  match_played = EXCLUDED.match_played
                                                      ,location   = EXCLUDED.location
                                                      ,type_id    = EXCLUDED.type_id
                                                      ,value      = EXCLUDED.value
                                                      ,percent    = EXCLUDED.percent
                                                      ,updated_at = NOW()`
	insertQuery := ""

	for _, v := range oc {
		insertQuery += fmt.Sprintf("(%d, %d, %d, %d, '%s', %d, %d, %f),",
			v.ParticipantId,
			v.SeasonId,
			v.LeagueId,
			v.MatchPlayed,
			v.Location,
			987,
			v.Ov75All,
			v.PercentOv75All)

		insertQuery += fmt.Sprintf("(%d, %d, %d, %d, '%s', %d, %d, %f),",
			v.ParticipantId,
			v.SeasonId,
			v.LeagueId,
			v.MatchPlayed,
			v.Location,
			988,
			v.Ov85All,
			v.PercentOv85All)

		insertQuery += fmt.Sprintf("(%d, %d, %d, %d, '%s', %d, %d, %f),",
			v.ParticipantId,
			v.SeasonId,
			v.LeagueId,
			v.MatchPlayed,
			v.Location,
			989,
			v.Ov95All,
			v.PercentOv95All)

		insertQuery += fmt.Sprintf("(%d, %d, %d, %d, '%s', %d, %d, %f),",
			v.ParticipantId,
			v.SeasonId,
			v.LeagueId,
			v.MatchPlayed,
			v.Location,
			990,
			v.Ov105All,
			v.PercentOv105All)

		insertQuery += fmt.Sprintf("(%d, %d, %d, %d, '%s', %d, %d, %f),",
			v.ParticipantId,
			v.SeasonId,
			v.LeagueId,
			v.MatchPlayed,
			v.Location,
			991,
			v.Ov115All,
			v.PercentOv115All)

		insertQuery += fmt.Sprintf("(%d, %d, %d, %d, '%s', %d, %d, %f),",
			v.ParticipantId,
			v.SeasonId,
			v.LeagueId,
			v.MatchPlayed,
			v.Location,
			992,
			v.Ov125All,
			v.PercentOv125All)

		insertQuery += fmt.Sprintf("(%d, %d, %d, %d, '%s', %d, %d, %f),",
			v.ParticipantId,
			v.SeasonId,
			v.LeagueId,
			v.MatchPlayed,
			v.Location,
			993,
			v.Ov135All,
			v.PercentOv135All)
	}

	query = query + strings.TrimRight(insertQuery, ",") + conflictQuery

	_, err = db.DBx.PGxPool.Exec(context.Background(), query)
	if err != nil {
		db.PGxConnError(taskData.Name(), "HandlerOverCorners", err.(*pgconn.PgError))
	}

	return
}
