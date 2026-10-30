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

func UpsertMarkets(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_markets(id, legacy_id, name, developer_name, has_winning_calculations) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET legacy_id        = EXCLUDED.legacy_id
                                                      , name           = EXCLUDED.name
                                                      , developer_name = EXCLUDED.developer_name
                                                      , has_winning_calculations = EXCLUDED.has_winning_calculations
                                                      , updated_at     = NOW()`

	for _, v := range data {
		query += fmt.Sprintf("(%s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("legacy_id")),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("developer_name"))),
			db.CastInterfaceString(v.GetBool("has_winning_calculations")),
		)
	}

	query = strings.TrimRight(query, ",") + conflictQuery
	_, err := db.DBx.PGxPool.Exec(context.Background(), query)
	if err != nil {
		db.PGxConnError(taskData.Name(), "UpsertMarkets", err.(*pgconn.PgError))
	}
}
