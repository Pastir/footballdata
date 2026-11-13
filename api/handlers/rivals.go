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

func upsertRivals(d *fastjson.Value, taskData config.TaskData) {
	data := d.GetArray("data")

	query := `INSERT INTO provider_rivals(id, team_id, rival_id) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET  team_id      = EXCLUDED.team_id
                                                      , rival_id    = EXCLUDED.rival_id                                           
                                                      , updated_at     = NOW()`

	var qi string
	for _, v := range data {

		qi += fmt.Sprintf("(%s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("team_id")),
			db.CastInterfaceString(v.GetInt("rival_id")),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "upsertRivals", err.(*pgconn.PgError))
		}
	}

}
