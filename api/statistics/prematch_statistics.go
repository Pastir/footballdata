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
	query := `WITH curr_next_round AS (SELECT season_id
                                            , current_round_id
                                            , next_round_id
                                       FROM (SELECT ps.id                                               AS season_id
                                                  , pr.is_current                                       AS current_round
                                                  , pr.id                                               AS current_round_id
                                                  , LEAD(pr.id) OVER (ORDER BY season_id, pr.name::INT) AS next_round_id
                                             FROM provider_seasons AS ps
                                                      LEFT JOIN provider_rounds AS pr ON (ps.id = pr.season_id)
                                                      LEFT JOIN provider_leagues AS pl ON (pl.id = ps.league_id)
                                             WHERE ps.is_current
                                               AND pl.sub_type = 'domestic'
                                             ORDER BY pr.name::INT) AS it
                                       WHERE current_round
) SELECT cnr.next_round_id
  FROM curr_next_round AS cnr LEFT JOIN LATERAL (
       SELECT COUNT(*) OVER()  AS num
       , pf.id AS fixture_id
       , pf.season_id
       FROM provider_fixtures AS pf
       LEFT JOIN provider_fixtures_participants AS hpfp ON (pf.id = hpfp.fixture_id AND hpfp.location = 'home')
       LEFT JOIN provider_fixtures_participants AS apfp ON (pf.id = apfp.fixture_id AND apfp.location = 'away')
       LEFT JOIN provider_states AS ps ON (pf.state_id = ps.id)
       WHERE pf.round_id = cnr.current_round_id
       AND pf.starting_at < (SELECT provider_rounds.starting_at FROM provider_rounds WHERE id = cnr.next_round_id)
       AND state_id != 5
    ) AS ns ON (ns.season_id = cnr.season_id)
  WHERE num ISNULL`

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
