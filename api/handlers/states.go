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

// UpsertStates - states processing
func UpsertStates(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_states(id, state, name, short_name, developer_name, sections_active) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET state                   = EXCLUDED.state
                                                      ,name                    = EXCLUDED.name
                                                      ,short_name              = EXCLUDED.short_name
                                                      ,developer_name          = EXCLUDED.developer_name
                                                      ,sections_active         = EXCLUDED.sections_active
                                                      ,updated_at              = NOW()`

	var qi string
	for _, v := range data {

		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(string(v.GetStringBytes("state"))),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("short_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("developer_name"))),
			db.CastInterfaceString(v.GetBool("sections_active")),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertStates", err.(*pgconn.PgError))
		}
	}
}
