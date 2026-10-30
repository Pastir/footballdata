package queries

import (
	"context"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
)

var ListStandingRules map[string]string

func GetStandingRules() {
	ListStandingRules = make(map[string]string)

	query := `SELECT uid, name FROM matchcenter_standing_rules WHERE provider_id = $1`
	//Запрос
	rows, err := db.DBx.PGxPool.Query(context.Background(), query, db.Prov.Id)
	if err != nil {
		db.PGxConnPanic("Init StandingRules", "GetStandingRules", err.(*pgconn.PgError))
		return
	}

	for rows.Next() {
		var rule string
		var ruleUid string

		err = rows.Scan(&ruleUid, &rule)
		if err != nil {
			db.PGxConnPanic("Init StandingRules", "GetStandingRules", err.(*pgconn.PgError))
			return
		}
		ListStandingRules[rule] = ruleUid
	}
}
