package handler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/logs"
	"github.com/davecgh/go-spew/spew"
	"github.com/jackc/pgconn"
	"github.com/valyala/fastjson"
)

// UpsertStandings - standings processing
func UpsertStandings(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_standings(id, participant_id, sport_id, league_id, season_id, stage_id, group_id, 
                               round_id, position, result, points, standing_rule_id) VALUES `
	conflictQuery := ` ON CONFLICT (participant_id, season_id, round_id) DO UPDATE SET id = EXCLUDED.id
                                                      ,sport_id         = EXCLUDED.sport_id
                                                      ,league_id        = EXCLUDED.league_id
                                                      ,stage_id         = EXCLUDED.stage_id
                                                      ,group_id         = EXCLUDED.group_id
                                                      ,position         = EXCLUDED.position
                                                      ,result           = EXCLUDED.result
                                                      ,points           = EXCLUDED.points
                                                      ,standing_rule_id = EXCLUDED.standing_rule_id
                                                      ,updated_at   = NOW()`

	var qi string
	for _, v := range data {
		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("participant_id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("league_id")),
			db.CastInterfaceString(v.GetInt("season_id")),
			db.CastInterfaceString(v.GetInt("stage_id")),
			db.CastInterfaceString(v.GetInt("group_id")),
			db.CastInterfaceString(v.GetInt("round_id")),
			db.CastInterfaceString(v.GetInt("position")),
			db.CastInterfaceString(string(v.GetStringBytes("result"))),
			db.CastInterfaceString(v.GetInt("points")),
			db.CastInterfaceString(v.GetInt("standing_rule_id")),
		)
	}

	if strings.Compare(qi, "") != 0 {
		q := query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), q)
		if err != nil {
			spew.Dump(err)
			logs.ErrorLog("upsertStandings", "UpsertStandings", err)
		}
	}

	UpsertStandingsDetails(d)
	UpsertStandingsForms(d)
	UpsertGroup(d)
	UpsertStandingsPositionRules(d)
}

// UpsertStandingsPositionRules upsert standings position rules
func UpsertStandingsPositionRules(d *fastjson.Value) {
	data := d.GetArray("data")

	query := `INSERT INTO provider_standings_position_rules(id, model_type, model_id, type_id, position) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET model_type = EXCLUDED.model_type
                                                      ,model_id= EXCLUDED.model_id
                                                      ,type_id = EXCLUDED.type_id
                                                      ,position = EXCLUDED.position
                                                      ,updated_at = NOW()`

	var qi string
	for _, form := range data {
		v := form.Get("rule")
		if v.GetInt("id") != 0 {
			qi += fmt.Sprintf("(%s, %s, %s, %s, %s),",
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(string(v.GetStringBytes("model_type"))),
				db.CastInterfaceString(v.GetInt("model_id")),
				db.CastInterfaceString(v.GetInt("type_id")),
				db.CastInterfaceString(v.GetInt("position")))
		}
	}

	if strings.Compare(qi, "") != 0 {
		q := query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), q)
		if err != nil {
			logs.ErrorLog("upsertStandings", "UpsertStandingsPositionRules", err)
		}
	}

}

// UpsertStandingsForms - upsert standings details processing
func UpsertStandingsForms(d *fastjson.Value) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_standings_forms(id, standing_type, standing_id, fixture_id, form, sort_order) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET standing_id = EXCLUDED.standing_id
                                                      ,standing_type = EXCLUDED.standing_type
                                                      ,fixture_id = EXCLUDED.fixture_id
                                                      ,form = EXCLUDED.form
                                                      ,sort_order = EXCLUDED.sort_order
                                                      ,updated_at = NOW()`
	var qi string
	for _, form := range data {
		for _, v := range form.GetArray("form") {
			qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s),",
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(string(v.GetStringBytes("standing_type"))),
				db.CastInterfaceString(v.GetInt("standing_id")),
				db.CastInterfaceString(v.GetInt("fixture_id")),
				db.CastInterfaceString(string(v.GetStringBytes("form"))),
				db.CastInterfaceString(v.GetInt("sort_order")),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		q := query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), q)
		if err != nil {
			logs.ErrorLog("upsertStandings", "UpsertStandingsForms", err)
		}
	}
}

// UpsertStandingsDetails - upsert standings details processing
func UpsertStandingsDetails(d *fastjson.Value) {

	data := d.GetArray("data")

	var standingIds []string
	for _, tm := range data {
		if sId := tm.GetInt("id"); sId > 0 {
			standingIds = append(standingIds, strconv.Itoa(sId))
		}
	}
	if len(standingIds) > 0 {
		delQuery := fmt.Sprintf("DELETE FROM provider_standings_details WHERE standing_id IN (%s)", strings.Join(standingIds, ","))
		_, _ = db.DBx.PGxPool.Exec(context.Background(), delQuery)
	}

	query := `INSERT INTO provider_standings_details(id, standing_id, type_id, standing_type, value) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET standing_id = EXCLUDED.standing_id
                                                      ,type_id    = EXCLUDED.type_id
                                                      ,value      = EXCLUDED.value
                                                      ,standing_type = EXCLUDED.standing_type
                                                      ,updated_at = NOW()`
	var qi string
	for _, tm := range data {
		standingId := tm.GetInt("id")
		detailsArr := tm.GetArray("details")
		for _, v := range detailsArr {
			sId := v.GetInt("standing_id")
			if sId == 0 {
				sId = standingId
			}
			qi += fmt.Sprintf("(%s, %s, %s, %s, %s),",
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(sId),
				db.CastInterfaceString(v.GetInt("type_id")),
				db.CastInterfaceString(string(v.GetStringBytes("standing_type"))),
				db.CastInterfaceString(v.GetInt("value")),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		q := query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), q)
		if err != nil {
			logs.ErrorLog("upsertStandings", "UpsertStandingsForms", err)
		}
	}
}

// UpsertGroup - upsert standings processing
func UpsertGroup(d *fastjson.Value) {

	data := d.GetArray("data")

	query := `INSERT INTO provider_groups(id, sport_id, league_id, season_id, stage_id, name, starting_at, ending_at) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id     = EXCLUDED.sport_id
                                                      ,league_id   = EXCLUDED.league_id
                                                      ,season_id   = EXCLUDED.season_id
                                                      ,stage_id    = EXCLUDED.stage_id
                                                      ,name        = EXCLUDED.name
                                                      ,starting_at = EXCLUDED.starting_at
                                                      ,ending_at   = EXCLUDED.ending_at
                                                      ,updated_at  = NOW()`

	var qi string
	groupId := make(map[int]string)

	for _, st := range data {
		v := st.Get("group")

		if v != nil && v.GetInt("id") > 0 {
			groupId[v.GetInt("id")] = fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s),",
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("sport_id")),
				db.CastInterfaceString(v.GetInt("league_id")),
				db.CastInterfaceString(v.GetInt("season_id")),
				db.CastInterfaceString(v.GetInt("stage_id")),
				db.CastInterfaceString(string(v.GetStringBytes("name"))),
				db.CastInterfaceString(string(v.GetStringBytes("starting_at"))),
				db.CastInterfaceString(string(v.GetStringBytes("ending_at"))),
			)
		}
	}

	if len(groupId) > 0 {
		for _, v := range groupId {
			qi += v
		}

		q := query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), q)
		if err != nil {
			logs.ErrorLog("upsertStandings", "UpsertGroup", err)
		}
	}
}

func GetFinishedRoundsOfSeason(currentSeason bool, seasonId string) []int {
	var rounds []int

	var Where []string
	var WhereAnd string

	Where = append(Where, "pr.finished")

	if strings.Compare(seasonId, "") != 0 {
		Where = append(Where, "pr.season_id IN ("+seasonId+")")
	}

	if currentSeason {
		Where = append(Where, "ps.is_current")
	}
	Where = append(Where, "pl.sub_type = 'domestic'")
	Where = append(Where, "pl.type = 'league'")

	if len(Where) > 0 {
		WhereAnd = "WHERE " + strings.Join(Where, " AND ")
	}

	query := `SELECT pr.id
              FROM provider_rounds AS pr 
                  LEFT JOIN provider_seasons AS ps ON(ps.id = pr.season_id)
                  LEFT JOIN provider_leagues pl ON(pr.league_id = pl.id)
              ` + WhereAnd

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("UpsertStandingsRounds", "Query GetRoundsAllFinishSeason", err.(*pgconn.PgError))
		os.Exit(1)
	}

	for rows.Next() {
		var Id int

		err = rows.Scan(&Id)
		if err != nil {
			db.PGxConnPanic("UpsertStandingsRounds", "Scan GetRoundsAllFinishSeason", err.(*pgconn.PgError))
			os.Exit(1)
		}

		rounds = append(rounds, Id)
	}

	return rounds
}

func GetPrevRounds() []int {
	var rounds []int
	query := `SELECT id 
              FROM (
                    SELECT ROW_NUMBER() OVER(PARTITION BY season_id ORDER BY prev.name::INT DESC) AS cnt
                         , prev.id
                         , prev.ending_at
                         , prev.name::INT
                    FROM provider_rounds AS prev
                        LEFT JOIN provider_seasons AS ps ON(prev.season_id = ps.id)
                    WHERE ps.is_current
                      AND prev.ending_at < NOW()
                    ORDER BY prev.name::INT DESC) AS it
              WHERE cnt = 1`

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("GetPrevRounds", "Query GetPrevRounds", err.(*pgconn.PgError))
		os.Exit(1)
	}

	for rows.Next() {
		var Id int

		err = rows.Scan(&Id)
		if err != nil {
			db.PGxConnPanic("GetPrevRounds", "Scan GetPrevRounds", err.(*pgconn.PgError))
			os.Exit(1)
		}

		rounds = append(rounds, Id)
	}

	return rounds
}

func GetCurrentRounds() []int {
	var rounds []int
	query := `SELECT cur.id
              FROM provider_rounds AS cur 
                  LEFT JOIN provider_seasons AS ps ON(cur.season_id = ps.id)
              WHERE cur.is_current
                AND ps.is_current`

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("GetCurrentRounds", "Query GetCurrentRounds", err.(*pgconn.PgError))
		os.Exit(1)
	}

	for rows.Next() {
		var Id int

		err = rows.Scan(&Id)
		if err != nil {
			db.PGxConnPanic("GetCurrentRounds", "Scan GetCurrentRounds", err.(*pgconn.PgError))
			os.Exit(1)
		}

		rounds = append(rounds, Id)
	}

	return rounds
}

func GetStageIdFromRounds(roundId int) (int, error) {
	var stageId int

	query := `SELECT stage_id FROM provider_rounds WHERE id = $1`

	rows, err := db.DBx.PGxPool.Query(context.Background(), query, roundId)
	if err != nil {
		db.PGxConnPanic("GetStageIdFromRounds", "Query GetStageIdFromRounds", err.(*pgconn.PgError))
		return 0, err
	}

	for rows.Next() {
		err = rows.Scan(&stageId)
		if err != nil {
			db.PGxConnPanic("GetStageIdFromRounds", "Scan GetStageIdFromRounds", err.(*pgconn.PgError))
			return 0, err
		}
	}

	return stageId, nil
}

func GetRoundsIdFromSeasonId(season_id string, currentSeasons bool) ([]int, error) {
	var roundsId []int
	where := ""

	if currentSeasons {
		where = "AND (pr.finished OR pr.is_current) "
	}

	query := `SELECT pr.id
FROM provider_rounds AS pr LEFT JOIN provider_seasons AS ps ON(pr.season_id = ps.id)
WHERE pr.season_id IN (` + season_id + `) ` + where + `ORDER BY pr.name::INT`

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("GetRoundsIdFromSeasonId", "Query GetRoundsIdFromSeasonId", err.(*pgconn.PgError))
		return roundsId, err
	}

	for rows.Next() {
		var rId int
		err = rows.Scan(&rId)
		if err != nil {
			db.PGxConnPanic("GetRoundsIdFromSeasonId", "Scan GetRoundsIdFromSeasonId", err.(*pgconn.PgError))
			return roundsId, err
		}

		roundsId = append(roundsId, rId)
	}

	return roundsId, nil
}
