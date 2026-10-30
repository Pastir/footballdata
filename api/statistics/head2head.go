package statistics

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
	pgx2 "github.com/jackc/pgx/v4"
	"os"
	"strings"
)

type (
	DataStatistics struct {
		FixtureID         int `sql:"id"`
		ParticipantIdHome int `sql:"participant_id_home"`
		ParticipantIdAway int `sql:"participant_id_away"`
		ParticipantIdOne  int `sql:"participant_id_one"`
		ParticipantIdTwo  int `sql:"participant_id_two"`
		TypeId            int `sql:"type_id"`
		TotalMatch        int `sql:"total_match"`
		HomeMatch         int `sql:"total_home_one"`
		AwayMatch         int `sql:"total_away_one"`
	}

	//'{"home": {"participant_id": 9, "value": {"total": 36, "home": 17, "away": 19}}, "away": {"participant_id": 9, "value": {"total": 36, "home": 19, "away": 17}}}'

	Values struct {
		ParticipantId int `json:"participant_id"`
		Value         struct {
			Total int `json:"total"`
			Home  int `json:"home"`
			Away  int `json:"away"`
		} `json:"value"`
	}

	Participant struct {
		Home Values `json:"home"`
		Away Values `json:"away"`
	}
)

func Matches(taskData config.TaskData) {
	var ds []DataStatistics
	query := `SELECT pf.id
      , participant_id_home
      , participant_id_away
      , participant_id_one
      , participant_id_two
      , 52 AS type_id
      , count(1) OVER(PARTITION BY participant_id_one, participant_id_two ORDER BY starting_at ASC ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING) AS total_match
      , count(1) FILTER ( WHERE participant_id_one = participant_id_home) OVER(PARTITION BY participant_id_one, participant_id_two ORDER BY starting_at ASC ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING) AS total_home_one
      , count(1) FILTER ( WHERE participant_id_one = participant_id_away) OVER(PARTITION BY participant_id_one, participant_id_two ORDER BY starting_at ASC ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING) AS total_away_one
FROM provider_fixtures AS pf
ORDER BY participant_id_one, participant_id_two, starting_at ASC`

	rows, err := db.DBx.PGxPool.Query(context.Background(), query)
	if err != nil {
		db.PGxConnPanic(taskData.Name(), taskData.Name()+"->Matches", err.(*pgconn.PgError))
		os.Exit(1)
	}

	for rows.Next() {
		var pds DataStatistics
		err = rows.Scan(&pds.FixtureID,
			&pds.ParticipantIdHome,
			&pds.ParticipantIdAway,
			&pds.ParticipantIdOne,
			&pds.ParticipantIdTwo,
			&pds.TypeId,
			&pds.TotalMatch,
			&pds.HomeMatch,
			&pds.AwayMatch)

		if err != nil {
			db.PGxScanPanic(taskData.Name(), taskData.Name()+"->Matches", err.(pgx2.ScanArgError))
			os.Exit(1)
		}

		ds = append(ds, pds)
	}

	upsertMatches(taskData, ds)

}

func upsertMatches(taskData config.TaskData, ds []DataStatistics) {
	query := `INSERT INTO head2head(fixture_id, type_id, value) VALUES`
	var valueInsert string

	for _, val := range ds {
		var data Participant
		if val.TotalMatch > 0 {
			if val.ParticipantIdHome == val.ParticipantIdOne {
				data.Home.ParticipantId = val.ParticipantIdOne
				data.Home.Value.Home = val.HomeMatch
				data.Home.Value.Away = val.AwayMatch
				data.Home.Value.Total = val.TotalMatch
				data.Away.ParticipantId = val.ParticipantIdTwo
				data.Away.Value.Home = val.AwayMatch
				data.Away.Value.Away = val.HomeMatch
				data.Away.Value.Total = val.TotalMatch
			} else {
				data.Home.ParticipantId = val.ParticipantIdTwo
				data.Home.Value.Home = val.HomeMatch
				data.Home.Value.Away = val.AwayMatch
				data.Home.Value.Total = val.TotalMatch
				data.Away.ParticipantId = val.ParticipantIdOne
				data.Away.Value.Home = val.AwayMatch
				data.Away.Value.Away = val.HomeMatch
				data.Away.Value.Total = val.TotalMatch
			}

			jsonData, err := json.Marshal(data)
			if err != nil {
				os.Exit(0)
			}

			valueInsert += fmt.Sprintf("(%s, %s, %s),",
				db.CastInterfaceString(val.FixtureID),
				db.CastInterfaceString(val.TypeId),
				db.CastInterfaceString(string(jsonData)),
			)
		}
	}

	if strings.Compare(valueInsert, "") != 0 {
		query = query + strings.TrimRight(valueInsert, ",")
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "upsertMatches", err.(*pgconn.PgError))
		}
	}
}
