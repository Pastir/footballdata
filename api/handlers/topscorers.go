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

func UpsertTopScorers(d *fastjson.Value, taskData config.TaskData) {

	data := d.GetArray("data")
	if len(data) == 0 {
		return
	}

	if taskData.Cycle == 0 {
		seasonID := data[0].GetInt("season_id")
		_, err := db.DBx.PGxPool.Exec(context.Background(), "DELETE FROM provider_seasons_topscores WHERE season_id = $1", seasonID)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertTopScorers", err.(*pgconn.PgError))
		}
	}

	valuesQuery := ""
	seen := make(map[[3]int]struct{}, len(data))

	for _, v := range data {
		seasonID := v.GetInt("season_id")
		playerID := v.GetInt("player_id")
		typeID := v.GetInt("type_id")
		key := [3]int{seasonID, playerID, typeID}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		valuesQuery += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(seasonID),
			db.CastInterfaceString(playerID),
			db.CastInterfaceString(typeID),
			db.CastInterfaceString(v.GetInt("position")),
			db.CastInterfaceString(v.GetInt("total")),
			db.CastInterfaceString(string(v.GetStringBytes("participant_type"))),
			db.CastInterfaceString(v.GetInt("participant_id")),
		)
	}

	if strings.Compare(valuesQuery, "") != 0 {
		query := fmt.Sprintf(`INSERT INTO provider_seasons_topscores(season_id, player_id, type_id, position, total, participant_type, participant_id)
SELECT v.season_id::int, v.player_id::bigint, v.type_id::int, v.position::int, v.total::int, v.participant_type::text, v.participant_id::int
FROM (VALUES %s) AS v(season_id, player_id, type_id, position, total, participant_type, participant_id)
WHERE NOT EXISTS (
	SELECT 1 FROM provider_seasons_topscores pst
	WHERE pst.season_id = v.season_id::int
	  AND pst.player_id = v.player_id::bigint
	  AND pst.type_id = v.type_id::int
)`, strings.TrimRight(valuesQuery, ","))
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			if err.(*pgconn.PgError).Code == "23503" {
				HandlingViolatesFKError(err, query)
			} else {
				db.PGxConnError(taskData.Name(), "UpsertTopScorers", err.(*pgconn.PgError))
			}
		}
	}
}
