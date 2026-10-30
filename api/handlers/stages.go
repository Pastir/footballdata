package handler

import (
	"context"
	"fmt"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
	"github.com/valyala/fastjson"
	"os"
	"strings"
)

// UpsertStages - добавляет/обновляет команды
// ./prematch -conf config.yaml -task upsertStages -seasonIsCurrent true
func UpsertStages(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	queryNameStages := `INSERT INTO provider_stages_name(name) VALUES `
	conflictQueryNameStages := ` ON CONFLICT (name) DO UPDATE SET name     = EXCLUDED.name
                                                      ,updated_at  = NOW()`

	query := `INSERT INTO provider_stages(id, sport_id, league_id, season_id, type_id, name, sort_order, 
                            finished, pending, is_current, starting_at, ending_at, games_in_current_week) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id     = EXCLUDED.sport_id
                                                      ,league_id   = EXCLUDED.league_id
                                                      ,season_id   = EXCLUDED.season_id
                                                      ,type_id     = EXCLUDED.type_id
                                                      ,name        = EXCLUDED.name
                                                      ,sort_order  = EXCLUDED.sort_order
                                                      ,finished    = EXCLUDED.finished
                                                      ,pending     = EXCLUDED.pending
                                                      ,is_current  = EXCLUDED.is_current
                                                      ,starting_at = EXCLUDED.starting_at
                                                      ,ending_at   = EXCLUDED.ending_at
                                                      ,games_in_current_week = EXCLUDED.games_in_current_week
                                                      ,updated_at  = NOW()`

	var qi string
	var qiNameStages string
	qiNameStagesMap := make(map[string]bool)

	for _, v := range data {
		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("league_id")),
			db.CastInterfaceString(v.GetInt("season_id")),
			db.CastInterfaceString(v.GetInt("type_id")),
			db.CastInterfaceString(string(v.GetStringBytes("name"))),
			db.CastInterfaceString(v.GetInt("sort_order")),
			db.CastInterfaceString(v.GetBool("finished")),
			db.CastInterfaceString(v.GetBool("pending")),
			db.CastInterfaceString(v.GetBool("is_current")),
			db.CastInterfaceString(string(v.GetStringBytes("starting_at"))),
			db.CastInterfaceString(string(v.GetStringBytes("ending_at"))),
			db.CastInterfaceString(v.GetBool("games_in_current_week")),
		)

		qiNameStagesMap[string(v.GetStringBytes("name"))] = true
	}

	for k, _ := range qiNameStagesMap {
		qiNameStages += "(" + db.CastInterfaceString(k) + "),"
	}

	if strings.Compare(qi, "") != 0 {
		// Insert stages name (FK)
		queryNameStages = queryNameStages + strings.TrimRight(qiNameStages, ",") + conflictQueryNameStages

		_, err := db.DBx.PGxPool.Exec(context.Background(), queryNameStages)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertStages", err.(*pgconn.PgError))
		}

		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err = db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertStages", err.(*pgconn.PgError))
		}
	}
}

// UpsertStageStatistics - seasons data processing
func UpsertStageStatistics(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_statistics_stages(id, model_id, type_id, relation_id, value) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET model_id    = EXCLUDED.model_id
                                                      , type_id    = EXCLUDED.type_id
                                                      , relation_id = EXCLUDED.relation_id
                                                      , value      = EXCLUDED.value
                                                      , updated_at = NOW()`
	var qi string
	for _, v := range data {
		qi += fmt.Sprintf("(%s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("model_id")),
			db.CastInterfaceString(v.GetInt("type_id")),
			db.CastInterfaceString(v.GetInt("relation_id")),
			db.CastInterfaceString(v.GetObject("value").String()),
		)
	}

	if strings.Compare(qi, "") != 0 {

		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertStageStatistics", err.(*pgconn.PgError))
		}
	}
}

func GetStagesOfSeason(currentSeason bool) []int {
	var stages []int

	var Where []string
	var WhereAnd string

	if currentSeason {
		Where = append(Where, "ps.is_current")
	}

	if len(Where) > 0 {
		WhereAnd = "WHERE " + strings.Join(Where, " AND ")
	}

	query := `SELECT pst.id
              FROM provider_stages AS pst
                  LEFT JOIN provider_seasons AS ps ON(ps.id = pst.season_id)
                  LEFT JOIN provider_leagues pl ON(pst.league_id = pl.id)` + WhereAnd

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("UpsertStageStatistics", "Quey GetStagesOfSeason", err.(*pgconn.PgError))
		os.Exit(1)
	}

	for rows.Next() {
		var Id int

		err = rows.Scan(&Id)
		if err != nil {
			db.PGxConnPanic("UpsertStageStatistics", "Scan GetStagesOfSeason", err.(*pgconn.PgError))
			os.Exit(1)
		}

		stages = append(stages, Id)
	}

	return stages
}
