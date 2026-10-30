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

// UpsertTypes - types data processing
func UpsertTypes(d *fastjson.Value, taskData config.TaskData) {
	data := d.GetArray("data")

	query := `INSERT INTO provider_types(id, name, code, developer_name, model_type) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET name           = EXCLUDED.name
                                                      ,code           = EXCLUDED.code
                                                      ,developer_name = EXCLUDED.developer_name 
                                                      ,model_type    = EXCLUDED.model_type
                                                      ,updated_at     = NOW()`

	for _, v := range data {
		query += fmt.Sprintf("(%s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("code"))),
			db.CastInterfaceString(string(v.GetStringBytes("developer_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("model_type"))),
		)
	}

	query = strings.TrimRight(query, ",") + conflictQuery
	_, err := db.DBx.PGxPool.Exec(context.Background(), query)
	if err != nil {
		db.PGxConnError(taskData.Name(), "UpsertTypes", err.(*pgconn.PgError))
	}
}
