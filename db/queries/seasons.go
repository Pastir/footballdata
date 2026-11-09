package queries

import (
	"context"
	"strconv"
	"strings"

	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
)

// GetSeasons возвращает мапу SeasonId -> SeasonData
func GetSeasons(taskData config.TaskData) map[int][]db.SeasonData {
	seasons := make(map[int][]db.SeasonData)
	var WhereAnd []string

	WhereAnd = append(WhereAnd, "active = 't'")

	if taskData.GetCurrentSeason() {
		WhereAnd = append(WhereAnd, "ps.is_current = 't'")
	}

	if taskData.GetLeagueId() != 0 {
		WhereAnd = append(WhereAnd, "pl.id = "+strconv.Itoa(taskData.GetLeagueId()))
	}

	if taskData.GetSeasonsId() != "" {
		WhereAnd = append(WhereAnd, "ps.id IN ("+taskData.GetSeasonsId()+")")
	}

	WhereAnds := strings.Join(WhereAnd, " AND ")

	query := `SELECT pl.id AS league_id
                     ,ps.id AS season_id
                     ,pl.uid AS league_uid
              FROM provider_leagues AS pl LEFT JOIN provider_seasons AS ps ON(pl.id = ps.league_id)
              WHERE ` + WhereAnds + " GROUP BY 1,2,3"

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnError("Init seasons", "GetSeasons", err)
		return nil
	}

	for rows.Next() {
		var seasData db.SeasonData

		err = rows.Scan(&seasData.LeagueId, &seasData.SeasonId, &seasData.LeagueUid)
		if err != nil {
			db.PGxScanPanic("init seasons", "GetSeasons", err)
			return nil
		}
		seasons[seasData.SeasonId] = append(seasons[seasData.LeagueId], seasData)
	}
	return seasons
}

func GetStrSeasonsIds(sIds map[int][]db.SeasonData) []int {
	var seasonIds []int
	for k, _ := range sIds {
		seasonIds = append(seasonIds, k)
	}

	return seasonIds
}

// GetCurrentSeasonIDs - возвращает список ID текущих сезонов для активных лиг
func GetCurrentSeasonIDs() ([]int, error) {
	var rounds []int

	query := `SELECT ps.id
FROM provider_seasons AS ps LEFT JOIN provider_leagues AS pl ON(pl.id = ps.league_id)
WHERE ps.is_current AND pl.active`

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("GetCurrentSeasonIDs", "Query GetCurrentSeasonIDs", err)
		return nil, err
	}

	for rows.Next() {
		var Id int

		err = rows.Scan(&Id)
		if err != nil {
			db.PGxConnPanic("GetCurrentSeasonIDs", "Scan GetCurrentSeasonIDs", err)
			return nil, err
		}

		rounds = append(rounds, Id)
	}

	return rounds, nil
}
