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

func UpsertOdds(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_odds(id, fixture_id, market_id, bookmaker_id, label, value, name, sort_order, 
                          market_description, probability, dp3, fractional, american, winning, stopped, total, 
                          handicap, participants, created_at, updated_at, original_label, latest_bookmaker_update) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET fixture_id           = EXCLUDED.fixture_id
                                                      , market_id          = EXCLUDED.market_id
                                                      , bookmaker_id       = EXCLUDED.bookmaker_id
                                                      , label              = EXCLUDED.label
                                                      , value              = EXCLUDED.value
                                                      , name               = EXCLUDED.name
                                                      , sort_order         = EXCLUDED.sort_order
                                                      , market_description = EXCLUDED.market_description
                                                      , probability        = EXCLUDED.probability
                                                      , dp3                = EXCLUDED.dp3
                                                      , fractional         = EXCLUDED.fractional
                                                      , american           = EXCLUDED.american
                                                      , winning            = EXCLUDED.winning
                                                      , stopped            = EXCLUDED.stopped
                                                      , total              = EXCLUDED.total
                                                      , handicap           = EXCLUDED.handicap
                                                      , participants       = EXCLUDED.participants
                                                      , created_at         = EXCLUDED.created_at
                                                      , updated_at         = EXCLUDED.updated_at
                                                      , original_label     = EXCLUDED.original_label
                                                      , latest_bookmaker_update = EXCLUDED.latest_bookmaker_update`

	var qi string
	for _, f := range data {
		for _, v := range f.GetArray("odds") {
			qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("fixture_id")),
				db.CastInterfaceString(v.GetInt("market_id")),
				db.CastInterfaceString(v.GetInt("bookmaker_id")),
				db.CastInterfaceString(string(v.GetStringBytes("label"))),
				db.CastInterfaceString(string(v.GetStringBytes("value"))),
				db.CastInterfaceString(string(v.GetStringBytes("name"))),
				db.CastInterfaceString(string(v.GetStringBytes("sort_order"))),
				db.CastInterfaceString(string(v.GetStringBytes("market_description"))),
				db.CastInterfaceString(string(v.GetStringBytes("probability"))),
				db.CastInterfaceString(string(v.GetStringBytes("dp3"))),
				db.CastInterfaceString(string(v.GetStringBytes("fractional"))),
				db.CastInterfaceString(string(v.GetStringBytes("american"))),
				db.CastInterfaceString(v.GetBool("winning")),
				db.CastInterfaceString(v.GetBool("stopped")),
				db.CastInterfaceString(string(v.GetStringBytes("total"))),
				db.CastInterfaceString(string(v.GetStringBytes("handicap"))),
				db.CastInterfaceString(string(v.GetStringBytes("participants"))),
				db.CastInterfaceString(string(v.GetStringBytes("created_at"))),
				db.CastInterfaceString(string(v.GetStringBytes("updated_at"))),
				db.CastInterfaceString(string(v.GetStringBytes("original_label"))),
				db.CastInterfaceString(string(v.GetStringBytes("latest_bookmaker_update"))),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertOdds", err.(*pgconn.PgError))
		}
	}
}
