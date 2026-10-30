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
   -task upsertSeasonsTeams -leagueId 2 -seasonIsCurrent true
*/

// UpsertTeams - team data processing
func UpsertTeams(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_teams(id, sport_id, country_id, venue_id, gender, name, short_code, image_path,
                           founded, type, placeholder, last_played_at) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id           = EXCLUDED.sport_id
                                                      ,country_id        = EXCLUDED.country_id
                                                      ,venue_id          = EXCLUDED.venue_id
                                                      ,gender            = EXCLUDED.gender
                                                      ,name              = EXCLUDED.name
                                                      ,short_code        = EXCLUDED.short_code
                                                      ,image_path        = EXCLUDED.image_path
                                                      ,founded           = EXCLUDED.founded                                                      
                                                      ,type              = EXCLUDED.type                                                   
                                                      ,placeholder       = EXCLUDED.placeholder
                                                      ,last_played_at    = EXCLUDED.last_played_at
                                                      ,updated_at        = NOW()`

	var qi string
	for _, v := range data {
		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("country_id")),
			db.CastInterfaceString(v.GetInt("venue_id")),
			db.CastInterfaceString(string(v.GetStringBytes("gender"))),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(string(v.GetStringBytes("short_code"))),
			db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
			db.CastInterfaceString(v.GetInt("founded")),
			db.CastInterfaceString(string(v.GetStringBytes("type"))),
			db.CastInterfaceString(v.GetBool("placeholder")),
			db.CastInterfaceString(string(v.GetStringBytes("last_played_at"))),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertTeams", err.(*pgconn.PgError))
		}
	}
}

func UpsertSeasonStatisticsTeams(d *fastjson.Value, taskData config.TaskData) {
	data := d.GetArray("data", "teams")

	query := `INSERT INTO provider_statistics_teams_season(id, team_statistic_id, team_id, season_id, type_id, stats_data) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET team_statistic_id    = EXCLUDED.team_statistic_id
                                                      ,team_id    = EXCLUDED.team_id
                                                      ,season_id  = EXCLUDED.season_id
                                                      ,type_id    = EXCLUDED.type_id
                                                      ,stats_data = EXCLUDED.stats_data
                                                      ,updated_at = NOW()`

	var qi string
	for _, stats := range data {
		for _, details := range stats.GetArray("statistics") {
			for _, v := range details.GetArray("details") {
				qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
					db.CastInterfaceString(details.GetInt("id")),
					db.CastInterfaceString(details.GetInt("team_id")),
					db.CastInterfaceString(details.GetInt("season_id")),
					db.CastInterfaceString(v.GetInt("type_id")),
					db.CastInterfaceString(v.GetObject("value").String()),
				)
			}
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			// error handling violates foreign key constraint
			if err.(*pgconn.PgError).Code == "23503" {
				HandlingViolatesFKError(err, query)
			} else {
				db.PGxConnError(taskData.Name(), "UpsertTeams", err.(*pgconn.PgError))
			}
		}
	}
}

// UpsertTeamById - team by id data processing
func UpsertTeamById(d *fastjson.Value, taskData config.TaskData) {

	query := `INSERT INTO provider_teams(id, sport_id, country_id, venue_id, gender, name, short_code, image_path,
                           founded, type, placeholder, last_played_at) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id           = EXCLUDED.sport_id
                                                      ,country_id        = EXCLUDED.country_id
                                                      ,venue_id          = EXCLUDED.venue_id
                                                      ,gender            = EXCLUDED.gender
                                                      ,name              = EXCLUDED.name
                                                      ,short_code        = EXCLUDED.short_code
                                                      ,image_path        = EXCLUDED.image_path
                                                      ,founded           = EXCLUDED.founded                                                      
                                                      ,type              = EXCLUDED.type                                                   
                                                      ,placeholder       = EXCLUDED.placeholder
                                                      ,last_played_at    = EXCLUDED.last_played_at
                                                      ,updated_at        = NOW()`

	var qi string
	v := d.Get("data")
	qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
		db.CastInterfaceString(v.GetInt("sport_id")),
		db.CastInterfaceString(v.GetInt("country_id")),
		db.CastInterfaceString(v.GetInt("venue_id")),
		db.CastInterfaceString(string(v.GetStringBytes("gender"))),
		db.CastInterfaceString(string(v.GetStringBytes("name"))),
		db.CastInterfaceString(string(v.GetStringBytes("short_code"))),
		db.CastInterfaceString(string(v.GetStringBytes("image_path"))),
		db.CastInterfaceString(v.GetInt("founded")),
		db.CastInterfaceString(string(v.GetStringBytes("type"))),
		db.CastInterfaceString(v.GetBool("placeholder")),
		db.CastInterfaceString(string(v.GetStringBytes("last_played_at"))),
	)

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertTeamById", err.(*pgconn.PgError))
		}
	}
}

func TranslateTeams(d *fastjson.Value, taskData config.TaskData) {
	query := `UPDATE provider_teams SET locale = locale || ('{"ru": {"short": "", "long": "'|| $1 ||'"}}')::JSONB WHERE id = $2`
	teams := d.GetArray("data")
	for _, v := range teams {
		teamId := v.GetInt("id")
		teamName := strings.TrimRight(string(v.GetStringBytes("name")), ".")
		if teamId > 0 {
			_, err := db.DBx.PGxPool.Exec(context.Background(), query, teamName, teamId)
			if err != nil {
				db.PGxConnError(taskData.Name(), "TranslateTeams", err.(*pgconn.PgError))
			}
		}
	}
}
