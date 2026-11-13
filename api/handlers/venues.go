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

// UpsertVenues - добавляет/обновляет стадионы
func UpsertVenues(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_venues(id, country_id, city_id, name, address, zipcode, latitude, longitude,
                                          capacity, image_path, city_name, surface, national_team) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET country_id    = EXCLUDED.country_id
                                                      , city_id     = EXCLUDED.city_id
                                                      , name        = EXCLUDED.name
                                                      , address     = EXCLUDED.address
                                                      , zipcode     = EXCLUDED.zipcode
                                                      , latitude    = EXCLUDED.latitude 
                                                      , longitude   = EXCLUDED.longitude
                                                      , capacity    = EXCLUDED.capacity
                                                      , image_path  = EXCLUDED.image_path
                                                      , city_name   = EXCLUDED.city_name
                                                      , surface     = EXCLUDED.surface
                                                      , national_team = EXCLUDED.national_team
                                                      , updated_at   = NOW()`

	var qi string
	for _, v := range data {
		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("country_id")),
			db.CastInterfaceString(v.GetInt("city_id")),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("address"))),
			db.CastInterfaceString(string(v.GetStringBytes("zipcode"))),
			db.CastInterfaceString(string(v.GetStringBytes("latitude"))),
			db.CastInterfaceString(string(v.GetStringBytes("longitude"))),
			db.CastInterfaceString(v.GetInt("capacity")),
			db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
			db.CastInterfaceString(string(v.GetStringBytes("city_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("surface"))),
			db.CastInterfaceString(v.GetBool("national_team")),
		)
	}

	if strings.Compare(qi, "") != 0 {
		q := query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), q)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertVenues", err.(*pgconn.PgError))
		}
	}
}
