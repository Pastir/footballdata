package handler

import (
	"context"
	"fmt"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
	"github.com/valyala/fastjson"
	"strings"
)

/*
   - task upsertSeasons
*/

// UpsertSeasons - seasons data processing
func UpsertSeasons(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_seasons(id, sport_id, league_id, tie_breaker_rule_id, name, 
                             finished, pending, is_current, standing_method, starting_at, ending_at, 
                             standings_recalculated_at, games_in_current_week) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id              = EXCLUDED.sport_id
                                                      , league_id           = EXCLUDED.league_id
                                                      , tie_breaker_rule_id = EXCLUDED.tie_breaker_rule_id
                                                      , name                = EXCLUDED.name
                                                      , finished            = EXCLUDED.finished
                                                      , pending             = EXCLUDED.pending
                                                      , is_current          = EXCLUDED.is_current
                                                      , standing_method     = EXCLUDED.standing_method
                                                      , starting_at         = EXCLUDED.starting_at
                                                      , ending_at           = EXCLUDED.ending_at
                                                      , standings_recalculated_at = EXCLUDED.standings_recalculated_at
                                                      , games_in_current_week     = EXCLUDED.games_in_current_week
                                                      , updated_at          = NOW()`

	for _, v := range data {
		query += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("league_id")),
			db.CastInterfaceString(v.GetInt("tie_breaker_rule_id")),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(v.GetBool("finished")),
			db.CastInterfaceString(v.GetBool("pending")),
			db.CastInterfaceString(v.GetBool("is_current")),
			db.CastInterfaceString(string(v.GetStringBytes("standing_method"))),
			db.CastInterfaceString(string(v.GetStringBytes("starting_at"))),
			db.CastInterfaceString(string(v.GetStringBytes("ending_at"))),
			db.CastInterfaceString(string(v.GetStringBytes("standings_recalculated_at"))),
			db.CastInterfaceString(v.GetBool("games_in_current_week")),
		)
	}

	query = strings.TrimRight(query, ",") + conflictQuery
	_, err := db.DBx.PGxPool.Exec(context.Background(), query)
	if err != nil {
		db.PGxConnError(taskData.Name(), "UpsertSeasons", err.(*pgconn.PgError))
	}

	UpsertSeasonsStatistics(d, taskData)
}

// UpsertSeasonsStatistics - seasons data processing
func UpsertSeasonsStatistics(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_statistics_seasons(id, season_id, type_id, value) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET season_id    = EXCLUDED.season_id
                                                      , type_id    = EXCLUDED.type_id
                                                      , value      = EXCLUDED.value
                                                      , updated_at = NOW()`
	var qi string
	for _, v := range data {
		for _, stat := range v.GetArray("statistics") {

			qi += fmt.Sprintf("(%s, %s, %s, %s),", db.CastInterfaceString(stat.GetInt("id")),
				db.CastInterfaceString(stat.GetInt("model_id")),
				db.CastInterfaceString(stat.GetInt("type_id")),
				db.CastInterfaceString(stat.GetObject("value").String()),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertSeasonsStatistics", err.(*pgconn.PgError))
		}
	}
}

// UpsertSeasonsTeams - teams for the season
func UpsertSeasonsTeams(d *fastjson.Value, taskData config.TaskData) {

	UpsertTeams(d, taskData)

	data := d.GetArray("data")

	query := `INSERT INTO provider_seasons_teams(season_id, team_id) VALUES `
	conflictQuery := ` ON CONFLICT (season_id, team_id) DO UPDATE SET season_id    = EXCLUDED.season_id
                                                                      , team_id    = EXCLUDED.team_id
                                                                      , updated_at = NOW()`
	var qi string
	for _, v := range data {
		qi += fmt.Sprintf("(%s, %s),", db.CastInterfaceString(taskData.Replace["season"]),
			db.CastInterfaceString(v.GetInt("id")),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertSeasonsTeams", err.(*pgconn.PgError))
		}
	}
}
