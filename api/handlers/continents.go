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

func UpsertContinents(data *fastjson.Value, taskData config.TaskData) {
	query := `INSERT INTO provider_continents(id, name, code) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET name  = EXCLUDED.name
                                                      , code = EXCLUDED.code
	                                                  , updated_at   = NOW()`
	continent := data.Get("data")
	var qi string

	for _, v := range continent.GetArray() {
		qi += fmt.Sprintf("(%s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("code"))),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertContinents", err.(*pgconn.PgError))
		}
	}

}
