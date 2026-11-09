package queries

import (
	"context"
	"github.com/Pastir/footballdata/db"
	"strconv"
	"strings"
)

// GetTeamsIdFromSeasons - returns an array of team IDs for the specified season
func GetTeamsIdFromSeasons(seasonId int) []int {
	var teamsId []int
	var WhereAnd []string

	if seasonId > 0 {
		WhereAnd = append(WhereAnd, "WHERE season_id = "+strconv.Itoa(seasonId))
	} else {
		return []int{}
	}

	WhereAnds := strings.Join(WhereAnd, " AND ")

	query := `SELECT team_id FROM provider_seasons_teams ` + WhereAnds

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnError("query", "GetTeamsIdFromSeasons", err)
		return nil
	}

	for rows.Next() {
		var t int

		err = rows.Scan(&t)
		if err != nil {
			db.PGxScanPanic("scan", "GetTeamsIdFromSeasons", err)
			return nil
		}
		teamsId = append(teamsId, t)
	}

	return teamsId
}
