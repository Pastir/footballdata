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

// upsertExpectedFixtures - teams data processing expected(xG values)
func upsertExpectedFixtures(d *fastjson.Value, taskData config.TaskData) {
	data := d.GetArray("data")

	query := `INSERT INTO provider_expected_fixtures(id, fixture_id, league_id, type_id, participant_id, value, location) VALUES `
	conflictQuery := ` ON CONFLICT (id, league_id) DO UPDATE SET fixture_id    = EXCLUDED.fixture_id
                                                      , league_id  = EXCLUDED.league_id
                                                      , type_id    = EXCLUDED.type_id
                                                      , participant_id = EXCLUDED.participant_id
                                                      , value      = EXCLUDED.value
                                                      , location   = EXCLUDED.location
                                                      , updated_at = NOW()`
	var qi string
	for _, v := range data {
		for _, stat := range v.GetArray("xgfixture") {

			qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(stat.GetInt("id")),
				db.CastInterfaceString(stat.GetInt("fixture_id")),
				db.CastInterfaceString(v.GetInt("league_id")),
				db.CastInterfaceString(stat.GetInt("type_id")),
				db.CastInterfaceString(stat.GetInt("participant_id")),
				db.CastInterfaceString(stat.GetFloat64("data", "value")),
				db.CastInterfaceString(string(stat.GetStringBytes("location"))),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertSeasonsStatistics", err.(*pgconn.PgError))
		}
	}
}
