package queries

import (
	"context"
	"os"

	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
)

// GetActiveLeagueIDs возвращает список ID активных лиг (или конкретный LeagueId из taskData).
func GetActiveLeagueIDs(taskData config.TaskData) []int {
	if taskData.GetLeagueId() > 0 {
		return []int{taskData.GetLeagueId()}
	}

	var leagues []int
	query := `SELECT id FROM provider_leagues WHERE active = 't' ORDER BY id`
	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnError("GetActiveLeagueIDs", "Query GetActiveLeagueIDs", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			db.PGxScanPanic("GetActiveLeagueIDs", "Scan GetActiveLeagueIDs", err)
			return nil
		}
		leagues = append(leagues, id)
	}
	return leagues
}

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
