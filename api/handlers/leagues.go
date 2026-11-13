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
  -task upsertLeagues
*/

// UpsertLeagues - league data processing
func UpsertLeagues(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_leagues(id, sport_id, country_id, name, active, short_code, image_path, type, 
                             sub_type, last_played_at, category, has_jerseys) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id         = EXCLUDED.sport_id
                                                      , country_id     = EXCLUDED.country_id
                                                      , name           = EXCLUDED.name
                                                      , active         = EXCLUDED.active
                                                      , short_code     = EXCLUDED.short_code                                               
                                                      , image_path     = EXCLUDED.image_path                                                    
                                                      , type           = EXCLUDED.type
                                                      , sub_type       = EXCLUDED.sub_type
                                                      , last_played_at = EXCLUDED.last_played_at
                                                      , category       = EXCLUDED.category
                                                      , has_jerseys    = EXCLUDED.has_jerseys                                            
                                                      , updated_at     = NOW()`

	var qi string
	for _, v := range data {

		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("country_id")),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(v.GetBool("active")),
			db.CastInterfaceString(string(v.GetStringBytes("short_code"))),
			db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
			db.CastInterfaceString(string(v.GetStringBytes("type"))),
			db.CastInterfaceString(string(v.GetStringBytes("sub_type"))),
			db.CastInterfaceString(string(v.GetStringBytes("last_played_at"))),
			db.CastInterfaceString(v.GetInt("category")),
			db.CastInterfaceString(v.GetBool("has_jerseys")),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertLeagues", err.(*pgconn.PgError))
		}
	}
}
