package handler

import (
	"context"
	"fmt"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/logs"
	"github.com/jackc/pgconn"
	"github.com/valyala/fastjson"
	"go.uber.org/zap"
	"strings"
	"time"
)

// UpsertCoachesTeamsSeasons -  coach data processing
func UpsertCoachesTeamsSeasons(d *fastjson.Value, taskData config.TaskData) {

	query := `INSERT INTO provider_coachs(id, player_id, sport_id, country_id, nationality_id, city_id, common_name, 
                            firstname, lastname, name, display_name, image_path, height, weight, date_of_birth, 
                            gender)  VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET player_id            = EXCLUDED.player_id,
                                                      sport_id             = EXCLUDED.sport_id,
                                                      country_id           = EXCLUDED.country_id,
                                                      nationality_id       = EXCLUDED.nationality_id,
                                                      city_id              = EXCLUDED.city_id,
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

	for _, tm := range d.GetArray("data") {
		var qi string
		coachId := make(map[int]bool)

		for _, ch := range tm.GetArray("coaches") {
			v := ch.Get("coach")
			if _, ok := coachId[v.GetInt("id")]; ok {
				continue
			}

			qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("player_id")),
				db.CastInterfaceString(v.GetInt("sport_id")),
				db.CastInterfaceString(v.GetInt("country_id")),
				db.CastInterfaceString(v.GetInt("nationality_id")),
				db.CastInterfaceString(v.GetInt("city_id")),
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

			coachId[v.GetInt("id")] = true
		}

		if strings.Compare(qi, "") != 0 {
			queryItog := query + strings.TrimRight(qi, ",") + conflictQuery
			_, err := db.DBx.PGxPool.Exec(context.Background(), queryItog)
			if err != nil {
				db.PGxConnError(taskData.Name(), "UpsertCoachesTeamsSeasons", err.(*pgconn.PgError))
			}
		}

		upsertTeamsCoaches(tm.GetArray("coaches"), taskData)
	}
}

// UpsertCoaches -  coach data processing
func UpsertCoaches(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_coachs(id, player_id, sport_id, country_id, nationality_id, city_id, common_name, 
                            firstname, lastname, name, display_name, image_path, height, weight, date_of_birth, 
                            gender)  VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET player_id            = EXCLUDED.player_id,
                                                      sport_id             = EXCLUDED.sport_id,
                                                      country_id           = EXCLUDED.country_id,
                                                      nationality_id       = EXCLUDED.nationality_id,
                                                      city_id              = EXCLUDED.city_id,
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

		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("player_id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("country_id")),
			db.CastInterfaceString(v.GetInt("nationality_id")),
			db.CastInterfaceString(v.GetInt("city_id")),
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
			db.PGxConnError(taskData.Name(), "UpsertCoaches", err.(*pgconn.PgError))
		}
	}
}

func upsertTeamsCoaches(d []*fastjson.Value, taskData config.TaskData) {

	query := `INSERT INTO provider_teams_coaches(id, team_id, coach_id, position_id, active, start, ends, temporary) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET team_id      = EXCLUDED.team_id,
                                                      coach_id     = EXCLUDED.coach_id,
                                                      position_id  = EXCLUDED.position_id,
                                                      active       = EXCLUDED.active,
                                                      start        = EXCLUDED.start,
                                                      ends         = EXCLUDED.ends,
                                                      temporary    = EXCLUDED.temporary,
                                                      updated_at   = NOW()`

	var qi string
	for _, v := range d {
		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("team_id")),
			db.CastInterfaceString(v.GetInt("coach_id")),
			db.CastInterfaceString(v.GetInt("position_id")),
			db.CastInterfaceString(v.GetBool("active")),
			db.CastInterfaceString(string(v.GetStringBytes("start"))),
			db.CastInterfaceString(string(v.GetStringBytes("end"))),
			db.CastInterfaceString(v.GetBool("temporary")),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "upsertTeamsCoaches", err.(*pgconn.PgError))
		}
	}
}

func UpsertCoachesIds(d *fastjson.Value, taskData config.TaskData) {

	query := `INSERT INTO provider_coachs(id, player_id, sport_id, country_id, nationality_id, city_id, common_name, 
                            firstname, lastname, name, display_name, image_path, height, weight, date_of_birth, 
                            gender)  VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET player_id            = EXCLUDED.player_id,
                                                      sport_id             = EXCLUDED.sport_id,
                                                      country_id           = EXCLUDED.country_id,
                                                      nationality_id       = EXCLUDED.nationality_id,
                                                      city_id              = EXCLUDED.city_id,
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
	v := d.Get("data")
	if v == nil || v.GetInt("id") == 0 {
		return
	}

	var qi string
	qi = fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
		db.CastInterfaceString(v.GetInt("player_id")),
		db.CastInterfaceString(v.GetInt("sport_id")),
		db.CastInterfaceString(v.GetInt("country_id")),
		db.CastInterfaceString(v.GetInt("nationality_id")),
		db.CastInterfaceString(v.GetInt("city_id")),
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
			db.PGxConnError(taskData.Name(), "UpsertCoachesIds", err.(*pgconn.PgError))
		}
	}
}

// UpsertCoachesFixturesErrorFK Error handling violates foreign key constraint
func UpsertCoachesFixturesErrorFK(d *fastjson.Value, taskName string) {

	logs.Logs["access"].Info("new request: ",
		zap.String("ts", time.Now().String()),
		zap.String("FuncName", "UpsertCoachesFixturesErrorFK"),
		zap.String("TaskName", taskName),
		zap.String("Event", "violates foreign key constraint"),
	)

	query := `INSERT INTO provider_coachs(id, player_id, sport_id, country_id, nationality_id, city_id, common_name, 
                            firstname, lastname, name, display_name, image_path, height, weight, date_of_birth, 
                            gender)  VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET player_id            = EXCLUDED.player_id,
                                                      sport_id             = EXCLUDED.sport_id,
                                                      country_id           = EXCLUDED.country_id,
                                                      nationality_id       = EXCLUDED.nationality_id,
                                                      city_id              = EXCLUDED.city_id,
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
	coaches := make(map[int]*fastjson.Value)

	for _, pf := range d.GetArray() {
		for _, v := range pf.GetArray("coaches") {
			coaches[v.GetInt("id")] = v
		}
	}

	for _, v := range coaches {
		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("player_id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("country_id")),
			db.CastInterfaceString(v.GetInt("nationality_id")),
			db.CastInterfaceString(v.GetInt("city_id")),
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
			db.PGxConnError(taskName, "UpsertCoachesFixturesErrorFK", err.(*pgconn.PgError))
		}
	}
}
