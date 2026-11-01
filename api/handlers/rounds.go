package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
	"github.com/valyala/fastjson"
)

// UpsertRounds - round data processing
func UpsertRounds(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_rounds(id, 
                            sport_id, 
                            league_id, 
                            season_id, 
                            stage_id, 
                            name, 
                            finished, 
                            is_current, 
                            starting_at, 
                            ending_at, 
                            games_in_current_week) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id     = EXCLUDED.sport_id
                                                      ,league_id   = EXCLUDED.league_id
                                                      ,season_id   = EXCLUDED.season_id
                                                      ,stage_id    = EXCLUDED.stage_id
                                                      ,name        = EXCLUDED.name
                                                      ,finished    = EXCLUDED.finished
                                                      ,is_current  = EXCLUDED.is_current
                                                      ,starting_at = EXCLUDED.starting_at
                                                      ,ending_at   = EXCLUDED.ending_at
                                                      ,games_in_current_week = EXCLUDED.games_in_current_week
                                                      ,updated_at  = NOW()`

	var qi string

	for _, r := range data {
		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(r.GetInt("id")),
			db.CastInterfaceString(r.GetInt("sport_id")),
			db.CastInterfaceString(r.GetInt("league_id")),
			db.CastInterfaceString(r.GetInt("season_id")),
			db.CastInterfaceString(r.GetInt("stage_id")),
			db.CastInterfaceString(string(r.GetStringBytes("name"))),
			db.CastInterfaceString(r.GetBool("finished")),
			db.CastInterfaceString(r.GetBool("is_current")),
			db.CastInterfaceString(string(r.GetStringBytes("starting_at"))),
			db.CastInterfaceString(string(r.GetStringBytes("ending_at"))),
			db.CastInterfaceString(r.GetBool("games_in_current_week")),
		)
	}

	if strings.Compare(qi, "") != 0 {

		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertRounds", err.(*pgconn.PgError))
		}
	}
}
