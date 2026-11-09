package queries

import (
	"context"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/db"
)

// NextRoundCurrentSeason возвращает данные следующего тура по ID лиги.
func NextRoundCurrentSeason(leagueID int) ([]analytics.NextRoundMatch, error) {
	info := analytics.LeagueSeasonInfo{LeagueID: leagueID}
	return analytics.GetNextRoundFixtures(context.Background(), db.DBx.PGxPool, &info)
}

// GetLast3CurrentSeasonRoundsIDs - возвращает id трех последних тура текущих сезонов, активны лиг
func GetLast3CurrentSeasonRoundsIDs() ([]int64, error) {
	var roundIds []int64

	query := `SELECT v.round_id
FROM provider_leagues AS pl
JOIN provider_seasons AS ps
     ON ps.league_id = pl.id
CROSS JOIN LATERAL get_neighbor_rounds(ps.id) AS ro
CROSS JOIN LATERAL (
    VALUES
        ('prev', ro.prev_round_id),
        ('last', ro.last_round_id),
        ('next', ro.next_round_id)
) AS v(kind, round_id)
WHERE pl.active
  AND ps.is_current
  AND v.round_id IS NOT NULL`

	//Выполняем запрос
	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("upsertRounds", "GetLast3CurrentSeasonRoundsIDs", err)
		return nil, err
	}

	for rows.Next() {
		var id int64

		err = rows.Scan(&id)
		if err != nil {
			db.PGxConnPanic("upsertRounds", "GetLast3CurrentSeasonRoundsIDs", err)
			return nil, err
		}

		roundIds = append(roundIds, id)
	}

	return roundIds, nil
}
