package queries

import (
	"context"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
)

// NextRoundCurrentSeason возвращает данные следующего тура по ID лиги.
func NextRoundCurrentSeason(leagueID int) ([]analytics.NextRoundMatch, error) {
	info := analytics.LeagueSeasonInfo{LeagueID: leagueID}
	return analytics.GetNextRoundFixtures(context.Background(), db.DBx.PGxPool, &info)
}

// GetLast3CurrentSeasonRoundsIDs - возвращает id трех последних тура текущих сезонов, активны лиг
func GetLast3CurrentSeasonRoundsIDs() ([]int64, error) {
	var roundIds []int64

	query := `SELECT id
FROM (SELECT row_number() OVER (PARTITION BY pr.season_id ORDER BY pr.season_id, pr.name::INT) AS num,
                      pr.id AS id
      FROM provider_rounds AS pr
               LEFT JOIN provider_leagues AS pl ON (pl.id = pr.league_id)
               LEFT JOIN provider_seasons AS ps ON (pr.season_id = ps.id)
      WHERE pl.active
        AND ps.is_current
        AND (pr.is_current OR pr.finished)) AS it
WHERE num <= 3`

	//Выполняем запрос
	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("upsertRounds", "GetLast3CurrentSeasonRoundsIDs", err.(*pgconn.PgError))
		return nil, err
	}

	for rows.Next() {
		var id int64

		err = rows.Scan(&id)
		if err != nil {
			db.PGxConnPanic("upsertRounds", "GetLast3CurrentSeasonRoundsIDs", err.(*pgconn.PgError))
			return nil, err
		}

		roundIds = append(roundIds, id)
	}

	return roundIds, nil
}

