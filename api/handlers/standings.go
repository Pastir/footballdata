package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/logs"
	"github.com/davecgh/go-spew/spew"
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
	UpsertGroup(d)
	UpsertStandingsPositionRules(d)
}

// UpsertStandingsLive обрабатывает живую таблицу из эндпоинта /v3/football/standings/live/leagues/{leagueID},
// привязывая позиции, очки и детали к активному туру (round_id) без перезаписи первичного ключа id прошлых туров.
func UpsertStandingsLive(d *fastjson.Value, taskData config.TaskData) {
	data := d.GetArray("data")
	if len(data) == 0 {
		return
	}

	seasonID := data[0].GetInt("season_id")
	targetRoundID := taskData.GetRoundId()
	if targetRoundID <= 0 && seasonID > 0 {
		roundQuery := `
			SELECT COALESCE(
				(SELECT pf.round_id
				 FROM provider_fixtures pf
				 WHERE pf.season_id = $1
				   AND pf.round_id IS NOT NULL
				   AND pf.starting_at BETWEEN ((NOW() AT TIME ZONE 'UTC') - INTERVAL '12 hours')
				                          AND ((NOW() AT TIME ZONE 'UTC') + INTERVAL '12 hours')
				 ORDER BY pf.starting_at DESC
				 LIMIT 1),
				(SELECT pr.id
				 FROM provider_rounds pr
				 WHERE pr.season_id = $1
				   AND pr.starting_at <= (NOW() AT TIME ZONE 'UTC')::date + INTERVAL '1 day'
				 ORDER BY pr.starting_at DESC
				 LIMIT 1),
				0
			)`
		_ = db.DBx.PGxPool.QueryRow(context.Background(), roundQuery, seasonID).Scan(&targetRoundID)
	}
	if targetRoundID <= 0 {
		targetRoundID = data[0].GetInt("round_id")
	}

	query := `INSERT INTO provider_standings(id, participant_id, sport_id, league_id, season_id, stage_id, group_id, 
                               round_id, position, result, points, standing_rule_id) VALUES `
	conflictQuery := ` ON CONFLICT (participant_id, season_id, round_id) DO UPDATE SET
                                                       sport_id         = EXCLUDED.sport_id
                                                      ,league_id        = EXCLUDED.league_id
                                                      ,stage_id         = EXCLUDED.stage_id
                                                      ,group_id         = EXCLUDED.group_id
                                                      ,position         = EXCLUDED.position
                                                      ,result           = EXCLUDED.result
                                                      ,points           = EXCLUDED.points
                                                      ,standing_rule_id = EXCLUDED.standing_rule_id
                                                      ,updated_at       = NOW()
                       RETURNING id, participant_id`

	var qi string
	for _, v := range data {
		qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),",
			db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("participant_id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("league_id")),
			db.CastInterfaceString(v.GetInt("season_id")),
			db.CastInterfaceString(v.GetInt("stage_id")),
			db.CastInterfaceString(v.GetInt("group_id")),
			db.CastInterfaceString(targetRoundID),
			db.CastInterfaceString(v.GetInt("position")),
			db.CastInterfaceString(string(v.GetStringBytes("result"))),
			db.CastInterfaceString(v.GetInt("points")),
			db.CastInterfaceString(v.GetInt("standing_rule_id")),
		)
	}

	standingByParticipant := make(map[int]int64, len(data))
	if strings.Compare(qi, "") != 0 {
		q := query + strings.TrimRight(qi, ",") + conflictQuery
		rows, err := db.DBx.PGxPool.Query(context.Background(), q)
		if err != nil {
			logs.ErrorLog("upsertStandingsLive", "UpsertStandingsLive", err)
			return
		}
		for rows.Next() {
			var sID int64
			var pID int
			if scanErr := rows.Scan(&sID, &pID); scanErr == nil {
				standingByParticipant[pID] = sID
			}
		}
		rows.Close()
	}

	var standingIDs []string
	for _, sID := range standingByParticipant {
		standingIDs = append(standingIDs, strconv.FormatInt(sID, 10))
	}
	if len(standingIDs) > 0 {
		delQuery := fmt.Sprintf("DELETE FROM provider_standings_details WHERE standing_id IN (%s)", strings.Join(standingIDs, ","))
		_, _ = db.DBx.PGxPool.Exec(context.Background(), delQuery)
	}

	detailsQuery := `INSERT INTO provider_standings_details(id, standing_id, type_id, standing_type, value) VALUES `
	detailsConflict := ` ON CONFLICT (id) DO UPDATE SET standing_id   = EXCLUDED.standing_id
                                                       ,type_id       = EXCLUDED.type_id
                                                       ,value         = EXCLUDED.value
                                                       ,standing_type = EXCLUDED.standing_type
                                                       ,updated_at    = NOW()`
	var dqi string
	for _, tm := range data {
		pID := tm.GetInt("participant_id")
		sID, ok := standingByParticipant[pID]
		if !ok || sID <= 0 {
			continue
		}
		for _, v := range tm.GetArray("details") {
			typeID := v.GetInt("type_id")
			if typeID <= 0 {
				continue
			}
			syntheticDetailID := sID*10000 + int64(typeID)
			dqi += fmt.Sprintf("(%s, %s, %s, %s, %s),",
				db.CastInterfaceString(syntheticDetailID),
				db.CastInterfaceString(sID),
				db.CastInterfaceString(typeID),
				db.CastInterfaceString(string(v.GetStringBytes("standing_type"))),
				db.CastInterfaceString(v.GetInt("value")),
			)
		}
	}

	if strings.Compare(dqi, "") != 0 {
		q := detailsQuery + strings.TrimRight(dqi, ",") + detailsConflict
		if _, err := db.DBx.PGxPool.Exec(context.Background(), q); err != nil {
			logs.ErrorLog("upsertStandingsLive", "UpsertStandingsDetailsLive", err)
		}
	}

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
			logs.ErrorLog("upsertStandings", "UpsertStandingsDetails", err)
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

