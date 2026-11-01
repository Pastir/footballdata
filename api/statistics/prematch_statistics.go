package statistics

import (
	"context"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/davecgh/go-spew/spew"
	"github.com/jackc/pgconn"
	pgx2 "github.com/jackc/pgx/v4"
	"os"
)

func PrematchStatistics(taskData config.TaskData) {

	nextRoundIds, err := GetNextRoundsDomestic()
	if err != nil {
		os.Exit(1)
	}

	spew.Dump(nextRoundIds)

}

func GetNextRoundsDomestic() ([]int, error) {
	var nextRoundIds []int
	query := `SELECT nr.next_round_id
FROM provider_seasons ps
JOIN provider_leagues pl ON pl.id = ps.league_id
CROSS JOIN LATERAL get_neighbor_rounds(ps.id) nr
WHERE ps.is_current AND pl.sub_type = 'domestic' AND pl.active = true
  AND nr.next_round_id IS NOT NULL`

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("PreMatchStatistics", "GetNextRoundsDomestic", err.(*pgconn.PgError))
		return nextRoundIds, err
	}

	for rows.Next() {
		var nextRound int
		err = rows.Scan(&nextRound)

		if err != nil {
			db.PGxScanPanic("PreMatchStatistics", "GetNextRoundsDomestic", err.(pgx2.ScanArgError))
			return nextRoundIds, err
		}

		nextRoundIds = append(nextRoundIds, nextRound)
	}

	return nextRoundIds, nil
}
