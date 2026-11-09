package queries

import (
	"context"
	"github.com/Pastir/footballdata/db"
	"os"
)

// GetListCountriesList
// Функция возвращает мапу id->uid лиг
func GetListCountriesList() []int {
	var countriesList []int

	query := `SELECT DISTINCT pc.id
FROM provider_seasons AS ps
  LEFT JOIN provider_rounds AS pr ON (ps.id = pr.season_id)
  LEFT JOIN provider_leagues pl ON (ps.league_id = pl.id)
  LEFT JOIN provider_countries pc ON (pl.country_id = pc.id)
WHERE ps.is_current
  AND pr.is_current`

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic("GetListCountriesList", "GetListCountriesList", err)
		os.Exit(1)
	}

	for rows.Next() {
		var countryId int

		err = rows.Scan(&countryId)
		if err != nil {
			db.PGxConnPanic("GetListCountriesList", "GetListCountriesList", err)
			os.Exit(1)
		}

		countriesList = append(countriesList, countryId)
	}

	return countriesList
}
