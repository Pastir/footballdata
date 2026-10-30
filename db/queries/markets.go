package queries

import (
	"context"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
	"os"
)

type Market struct {
	MarketId    int     `db:"market_id"`
	BookmakerId int     `db:"bookmaker_id"`
	MatchId     int     `db:"match_id"`
	Value       float64 `db:"value"`
	Handicap    string  `db:"handicap"`
	Total       string  `db:"total"`
	Label       string  `db:"label"`
	Probability string  `db:"probability"`
	Winning     string  `db:"winning"`
	LastUpdate  string  `db:"last_update"`
	Xxhash64    uint64  `db:"xxhash64"`
}

var ListMarket map[string]int

func GetMarkets() {
	ListMarket = make(map[string]int)

	query := `SELECT id, name FROM matchcenter_market WHERE active`
	//Выполняем запрос
	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("Init Markets", "GetMarkets", err.(*pgconn.PgError))
		os.Exit(1)
	}

	for rows.Next() {
		var name string
		var mId int

		err = rows.Scan(&mId, &name)
		if err != nil {
			db.PGxConnPanic("Init Markets", "GetMarkets", err.(*pgconn.PgError))
			os.Exit(1)
		}

		ListMarket[name] = mId
	}
}

/*func SaveMarketTotal(m []Market) {
	for _, v := range m {
		_, err := db.DBx.NamedExec(`INSERT INTO matchcenter_odds(market_id, bookmaker_id, match_id, total, label, value, xxhash64, probability, last_update, handicap)
                            VALUES(:market_id, :bookmaker_id, :match_id, :total, :label, :value, :xxhash64, :probability, :last_update, :handicap)
                            ON CONFLICT (xxhash64) DO UPDATE SET updated_at = NOW(),
                                                       value = EXCLUDED.value,
                                                       probability = EXCLUDED.probability,
                                                       winning = EXCLUDED.winning,
                                                       last_update = NOW()`, &v)

		if err, ok := err.(*pgx.PgError); ok {
			db.PGxError("SaveMarketTotal", err)
		}
	}
}*/
