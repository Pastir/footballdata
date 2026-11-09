package queries

import (
	"context"
	"github.com/Pastir/footballdata/db"
	"os"
)

// GetListLeaguesId
// Функция возвращает мапу id->uid лиг
func GetListLeaguesId() map[int]string {
	listLeague := make(map[int]string)

	query := `SELECT id, uid FROM matchcenter_league WHERE provider_id = $1`

	rows, err := db.DBx.PGxPool.Query(context.Background(), query, db.Prov.Id)
	if err != nil {
		db.PGxConnPanic("Init ListLeaguesId", "GetListLeaguesId", err)
		os.Exit(1)
	}

	for rows.Next() {
		var leagueId int
		var leagueUid string

		err = rows.Scan(&leagueId, &leagueUid)
		if err != nil {
			db.PGxConnPanic("Init ListLeaguesId", "GetListLeaguesId", err)
			os.Exit(1)
		}

		listLeague[leagueId] = leagueUid
	}

	return listLeague
}
