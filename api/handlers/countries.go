package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/logs"
	"github.com/jackc/pgconn"
	"github.com/valyala/fastjson"
	"strings"
)

func UpsertCountries(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_countries(id, geonameid, continent_id, name, official_name, fifa_name, iso2, iso3, 
                               longitude, latitude, image_path, borders) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET geonameid     = EXCLUDED.geonameid,
                                                      continent_id  = EXCLUDED.continent_id,
                                                      name          = EXCLUDED.name,
                                                      official_name = EXCLUDED.official_name,
                                                      fifa_name     = EXCLUDED.fifa_name,
                                                      iso2          = EXCLUDED.iso2,
                                                      iso3          = EXCLUDED.iso3,
                                                      latitude      = EXCLUDED.latitude,
                                                      longitude     = EXCLUDED.longitude,
                                                      image_path    = EXCLUDED.image_path,
                                                      borders       = EXCLUDED.borders,
                                                      updated_at   = NOW()`

	var qi string
	for _, v := range data {
		var borderArray []string
		for _, borders := range v.GetArray("borders") {
			borderArray = append(borderArray, string(borders.GetStringBytes()))
		}

		j, err := json.Marshal(borderArray)
		if err != nil {
			logs.ErrorLog(taskData.Name(), "UpsertCountries", err)
		}

		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("geonameid")),
			db.CastInterfaceString(v.GetInt("continent_id")),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("official_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("fifa_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("iso2"))),
			db.CastInterfaceString(string(v.GetStringBytes("iso3"))),
			db.CastInterfaceString(string(v.GetStringBytes("longitude"))),
			db.CastInterfaceString(string(v.GetStringBytes("latitude"))),
			db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
			db.CastInterfaceString(string(j)),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertCountries", err.(*pgconn.PgError))
		}
	}
}
