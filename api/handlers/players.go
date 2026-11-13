package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
	"github.com/valyala/fastjson"
	"strings"
)

func UpsertPlayerIds(d *fastjson.Value, taskData config.TaskData) {

	query := `INSERT INTO provider_players(id, sport_id, country_id, nationality_id, city_id, position_id,
                             detailed_position_id, type_id, common_name, firstname, lastname, name, display_name,
                             image_path, height, weight, date_of_birth, gender) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id             = EXCLUDED.sport_id,
                                                      country_id           = EXCLUDED.country_id,
                                                      nationality_id       = EXCLUDED.nationality_id,
                                                      city_id              = EXCLUDED.city_id,
                                                      position_id          = EXCLUDED.position_id,
                                                      detailed_position_id = EXCLUDED.detailed_position_id,
                                                      type_id              = EXCLUDED.type_id,
                                                      common_name          = EXCLUDED.common_name,
                                                      firstname            = EXCLUDED.firstname,
                                                      lastname             = EXCLUDED.lastname,
                                                      name                 = EXCLUDED.name,
                                                      display_name         = EXCLUDED.display_name,
                                                      image_path           = EXCLUDED.image_path,
                                                      height               = EXCLUDED.height,
                                                      weight               = EXCLUDED.weight,
                                                      date_of_birth        = EXCLUDED.date_of_birth,
                                                      gender               = EXCLUDED.gender,
                                                      updated_at           = NOW()`
	var qi string
	v := d.Get("data")
	qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
		db.CastInterfaceString(v.GetInt("sport_id")),
		db.CastInterfaceString(v.GetInt("country_id")),
		db.CastInterfaceString(v.GetInt("nationality_id")),
		db.CastInterfaceString(v.GetInt("city_id")),
		db.CastInterfaceString(v.GetInt("position_id")),
		db.CastInterfaceString(v.GetInt("detailed_position_id")),
		db.CastInterfaceString(v.GetInt("type_id")),
		db.CastInterfaceString(string(v.GetStringBytes("common_name"))),
		db.CastInterfaceString(string(v.GetStringBytes("firstname"))),
		db.CastInterfaceString(string(v.GetStringBytes("lastname"))),
		db.CastInterfaceString(string(v.GetStringBytes("name"))),
		db.CastInterfaceString(string(v.GetStringBytes("display_name"))),
		db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
		db.CastInterfaceString(string(v.GetStringBytes("height"))),
		db.CastInterfaceString(string(v.GetStringBytes("weight"))),
		db.CastInterfaceString(string(v.GetStringBytes("date_of_birth"))),
		db.CastInterfaceString(string(v.GetStringBytes("gender"))),
	)

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertPlayers", err.(*pgconn.PgError))
		}
	}
}

func UpsertPlayersSeasons(d *fastjson.Value, taskData config.TaskData) {

	query := `INSERT INTO provider_players(id, sport_id, country_id, nationality_id, city_id, position_id,
                             detailed_position_id, type_id, common_name, firstname, lastname, name, display_name,
                             image_path, height, weight, date_of_birth, gender) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id             = EXCLUDED.sport_id,
                                                      country_id           = EXCLUDED.country_id,
                                                      nationality_id       = EXCLUDED.nationality_id,
                                                      city_id              = EXCLUDED.city_id,
                                                      position_id          = EXCLUDED.position_id,
                                                      detailed_position_id = EXCLUDED.detailed_position_id,
                                                      type_id              = EXCLUDED.type_id,
                                                      common_name          = EXCLUDED.common_name,
                                                      firstname            = EXCLUDED.firstname,
                                                      lastname             = EXCLUDED.lastname,
                                                      name                 = EXCLUDED.name,
                                                      display_name         = EXCLUDED.display_name,
                                                      image_path           = EXCLUDED.image_path,
                                                      height               = EXCLUDED.height,
                                                      weight               = EXCLUDED.weight,
                                                      date_of_birth        = EXCLUDED.date_of_birth,
                                                      gender               = EXCLUDED.gender,
                                                      updated_at           = NOW()`

	for _, pls := range d.GetArray("data") {
		var qi string
		for _, pl := range pls.GetArray("players") {
			v := pl.Get("player")
			qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("sport_id")),
				db.CastInterfaceString(v.GetInt("country_id")),
				db.CastInterfaceString(v.GetInt("nationality_id")),
				db.CastInterfaceString(v.GetInt("city_id")),
				db.CastInterfaceString(v.GetInt("position_id")),
				db.CastInterfaceString(v.GetInt("detailed_position_id")),
				db.CastInterfaceString(v.GetInt("type_id")),
				db.CastInterfaceString(string(v.GetStringBytes("common_name"))),
				db.CastInterfaceString(string(v.GetStringBytes("firstname"))),
				db.CastInterfaceString(string(v.GetStringBytes("lastname"))),
				db.CastInterfaceString(string(v.GetStringBytes("name"))),
				db.CastInterfaceString(string(v.GetStringBytes("display_name"))),
				db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
				db.CastInterfaceString(string(v.GetStringBytes("height"))),
				db.CastInterfaceString(string(v.GetStringBytes("weight"))),
				db.CastInterfaceString(string(v.GetStringBytes("date_of_birth"))),
				db.CastInterfaceString(string(v.GetStringBytes("gender"))),
			)
		}

		if strings.Compare(qi, "") != 0 {
			queryItog := query + strings.TrimRight(qi, ",") + conflictQuery
			_, err := db.DBx.PGxPool.Exec(context.Background(), queryItog)
			if err != nil {
				if err.(*pgconn.PgError).Code == "23503" {
					HandlingViolatesFKError(err, queryItog)
				} else {
					db.PGxConnError(taskData.Name(), "UpsertPlayersSeasons", err.(*pgconn.PgError))
				}
			}
		}
	}

}

// UpsertPlayers - players data processing
func UpsertPlayers(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_players(id, sport_id, country_id, nationality_id, city_id, position_id,
                             detailed_position_id, type_id, common_name, firstname, lastname, name, display_name,
                             image_path, height, weight, date_of_birth, gender) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id             = EXCLUDED.sport_id,
                                                      country_id           = EXCLUDED.country_id,
                                                      nationality_id       = EXCLUDED.nationality_id,
                                                      city_id              = EXCLUDED.city_id,
                                                      position_id          = EXCLUDED.position_id,
                                                      detailed_position_id = EXCLUDED.detailed_position_id,
                                                      type_id              = EXCLUDED.type_id,
                                                      common_name          = EXCLUDED.common_name,
                                                      firstname            = EXCLUDED.firstname,
                                                      lastname             = EXCLUDED.lastname,
                                                      name                 = EXCLUDED.name,
                                                      display_name         = EXCLUDED.display_name,
                                                      image_path           = EXCLUDED.image_path,
                                                      height               = EXCLUDED.height,
                                                      weight               = EXCLUDED.weight,
                                                      date_of_birth        = EXCLUDED.date_of_birth,
                                                      gender               = EXCLUDED.gender,
                                                      updated_at           = NOW()`
	var qi string
	for _, v := range data {

		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("country_id")),
			db.CastInterfaceString(v.GetInt("nationality_id")),
			db.CastInterfaceString(v.GetInt("city_id")),
			db.CastInterfaceString(v.GetInt("position_id")),
			db.CastInterfaceString(v.GetInt("detailed_position_id")),
			db.CastInterfaceString(v.GetInt("type_id")),
			db.CastInterfaceString(string(v.GetStringBytes("common_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("firstname"))),
			db.CastInterfaceString(string(v.GetStringBytes("lastname"))),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("display_name"))),
			db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
			db.CastInterfaceString(string(v.GetStringBytes("height"))),
			db.CastInterfaceString(string(v.GetStringBytes("weight"))),
			db.CastInterfaceString(string(v.GetStringBytes("date_of_birth"))),
			db.CastInterfaceString(string(v.GetStringBytes("gender"))),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertPlayers", err.(*pgconn.PgError))
		}
	}
}

// UpsertPlayerSeasonsStatistics -
func UpsertPlayerSeasonsStatistics(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_statistics_players_seasons(id, type_id, season_id, player_id, team_id, position_id
                                                              , jersey_number, home, highest, lowest, goals, "out", "in"
                                                              , total, away, average, penalties) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET type_id       = EXCLUDED.type_id,
                                                      season_id     = EXCLUDED.season_id,
                                                      player_id     = EXCLUDED.player_id,
                                                      team_id       = EXCLUDED.team_id,
                                                      position_id   = EXCLUDED.position_id,
                                                      jersey_number = EXCLUDED.jersey_number,
                                                      home          = EXCLUDED.home,
                                                      highest       = EXCLUDED.highest,
                                                      lowest        = EXCLUDED.lowest,
                                                      goals         = EXCLUDED.goals,
                                                      "out"         = EXCLUDED."out",
                                                      "in"          = EXCLUDED."in",
                                                      total         = EXCLUDED.total,
                                                      away          = EXCLUDED.away,
                                                      average       = EXCLUDED.average,
                                                      penalties     = EXCLUDED.penalties,
                                                      updated_at    = NOW()`

	var qi string

	for _, v := range data {
		for _, val := range v.GetArray("details") {
			sp := make(map[string]interface{})
			json.Unmarshal([]byte(val.Get("value").String()), &sp)
			qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(val.GetInt("id")),
				db.CastInterfaceString(val.GetInt("type_id")),
				db.CastInterfaceString(v.GetInt("season_id")),
				db.CastInterfaceString(v.GetInt("player_id")),
				db.CastInterfaceString(v.GetInt("team_id")),
				db.CastInterfaceString(v.GetInt("position_id")),
				db.CastInterfaceString(v.GetInt("jersey_number")),
				db.CastInterfaceString(sp["home"]),
				db.CastInterfaceString(sp["highest"]),
				db.CastInterfaceString(sp["lowest"]),
				db.CastInterfaceString(sp["goals"]),
				db.CastInterfaceString(sp["out"]),
				db.CastInterfaceString(sp["in"]),
				db.CastInterfaceString(sp["total"]),
				db.CastInterfaceString(sp["away"]),
				db.CastInterfaceString(sp["average"]),
				db.CastInterfaceString(sp["penalties"]),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			if err.(*pgconn.PgError).Code == "23503" {
				HandlingViolatesFKError(err, query)
			} else {
				db.PGxConnError(taskData.Name(), "UpsertPlayerSeasonsStatistics", err.(*pgconn.PgError))
			}
		}
	}
}
