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

// UpsertReferees - обновляет данные тренеров
func UpsertReferees(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_referees(id, sport_id, country_id, city_id, common_name, firstname, lastname, name, 
                             display_name, image_path, height, weight, date_of_birth, gender) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id       = EXCLUDED.sport_id
                                                      ,country_id    = EXCLUDED.country_id
                                                      ,city_id       = EXCLUDED.city_id
                                                      ,common_name   = EXCLUDED.common_name
                                                      ,firstname     = EXCLUDED.firstname
                                                      ,lastname      = EXCLUDED.lastname
                                                      ,name          = EXCLUDED.name
                                                      ,display_name  = EXCLUDED.display_name
                                                      ,image_path    = EXCLUDED.image_path
                                                      ,height        = EXCLUDED.height
                                                      ,weight        = EXCLUDED.weight
                                                      ,date_of_birth = EXCLUDED.date_of_birth
                                                      ,gender        = EXCLUDED.gender
                                                      ,updated_at    = NOW()`

	var qi string
	// referee
	for _, v := range data {

		qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),`, db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("country_id")),
			db.CastInterfaceString(v.GetInt("city_id")),
			db.CastInterfaceString(string(v.GetStringBytes("common_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("firstname"))),
			db.CastInterfaceString(string(v.GetStringBytes("lastname"))),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("display_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
			db.CastInterfaceString(v.GetInt("height")),
			db.CastInterfaceString(v.GetInt("weight")),
			db.CastInterfaceString(string(v.GetStringBytes("date_of_birth"))),
			db.CastInterfaceString(string(v.GetStringBytes("gender"))),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertReferees", err.(*pgconn.PgError))
		}
	}
}

func UpsertRefereesSeasons(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_referees(id, sport_id, country_id, city_id, common_name, firstname, lastname, name, 
                             display_name, image_path, height, weight, date_of_birth, gender) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id       = EXCLUDED.sport_id
                                                      ,country_id    = EXCLUDED.country_id
                                                      ,city_id       = EXCLUDED.city_id
                                                      ,common_name   = EXCLUDED.common_name
                                                      ,firstname     = EXCLUDED.firstname
                                                      ,lastname      = EXCLUDED.lastname
                                                      ,name          = EXCLUDED.name
                                                      ,display_name  = EXCLUDED.display_name
                                                      ,image_path    = EXCLUDED.image_path
                                                      ,height        = EXCLUDED.height
                                                      ,weight        = EXCLUDED.weight
                                                      ,date_of_birth = EXCLUDED.date_of_birth
                                                      ,gender        = EXCLUDED.gender
                                                      ,updated_at    = NOW()`

	var qi string
	// referee
	for _, v := range data {

		qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),`, db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("country_id")),
			db.CastInterfaceString(v.GetInt("city_id")),
			db.CastInterfaceString(string(v.GetStringBytes("common_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("firstname"))),
			db.CastInterfaceString(string(v.GetStringBytes("lastname"))),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("display_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
			db.CastInterfaceString(v.GetInt("height")),
			db.CastInterfaceString(v.GetInt("weight")),
			db.CastInterfaceString(string(v.GetStringBytes("date_of_birth"))),
			db.CastInterfaceString(string(v.GetStringBytes("gender"))),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertRefereesSeasons", err.(*pgconn.PgError))
		}
	}
}
