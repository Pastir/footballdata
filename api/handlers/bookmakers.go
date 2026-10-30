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

func UpsertBookmakers(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_bookmakers(id, legacy_id, name) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET legacy_id  = EXCLUDED.legacy_id
                                                      , name       = EXCLUDED.name
                                                      , updated_at = NOW()`

	for _, v := range data {
		query += fmt.Sprintf("(%s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("legacy_id")),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
		)
	}

	query = strings.TrimRight(query, ",") + conflictQuery
	_, err := db.DBx.PGxPool.Exec(context.Background(), query)
	if err != nil {
		db.PGxConnError(taskData.Name(), "UpsertBookmakers", err.(*pgconn.PgError))
	}
}
