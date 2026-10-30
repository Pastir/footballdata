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

func UpsertTopScorers(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	if taskData.Cycle == 0 && len(data) > 0 {
		seasonID := data[0].GetInt("season_id")
		_, err := db.DBx.PGxPool.Exec(context.Background(), "DELETE FROM provider_seasons_topscores WHERE season_id = $1", seasonID)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertTopScorers", err.(*pgconn.PgError))
		}
	}

	valuesQuery := ""
	query := `INSERT INTO provider_seasons_topscores(season_id, player_id,type_id,position,total,
                                       participant_type,participant_id) VALUES `

	for _, v := range data {
		valuesQuery += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("season_id")),
			db.CastInterfaceString(v.GetInt("player_id")),
			db.CastInterfaceString(v.GetInt("type_id")),
			db.CastInterfaceString(v.GetInt("position")),
			db.CastInterfaceString(v.GetInt("total")),
			db.CastInterfaceString(string(v.GetStringBytes("participant_type"))),
			db.CastInterfaceString(v.GetInt("participant_id")),
		)
	}

	if strings.Compare(valuesQuery, "") != 0 {
		query = query + strings.TrimRight(valuesQuery, ",")
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			if err.(*pgconn.PgError).Code == "23503" {
				HandlingViolatesFKError(err, query)
			} else {
				db.PGxConnError(taskData.Name(), "UpsertTopScorers", err.(*pgconn.PgError))
			}
		}
	}
}
