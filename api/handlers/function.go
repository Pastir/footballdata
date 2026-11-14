package handler

import (
	"context"
	"errors"
	"regexp"

	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/jackc/pgconn"
	"github.com/valyala/fastjson"
)

var Handlers map[string]func(dv *fastjson.Value, argsData config.TaskData)

func init() {
	Handlers = map[string]func(dv *fastjson.Value, argsData config.TaskData){
		"upsertCountries":               UpsertCountries,
		"upsertLeagues":                 UpsertLeagues,
		"upsertContinents":              UpsertContinents,
		"upsertSeasons":                 UpsertSeasons,
		"upsertSeasonsTeams":            UpsertSeasonsTeams,
		"upsertPlayers":                 UpsertPlayers,
		"upsertPlayerIds":               UpsertPlayerIds,
		"upsertPlayersSeasons":          UpsertPlayersSeasons,
		"upsertPlayerSeasonsStatistics": UpsertPlayerSeasonsStatistics,
		"upsertCoaches":                 UpsertCoaches,
		"upsertCoachesIds":              UpsertCoachesIds,
		"upsertCoachesTeamsSeasons":     UpsertCoachesTeamsSeasons,
		"upsertTeams":                   UpsertTeams,
		"upsertTeamById":                UpsertTeamById,
		"upsertSeasonStatisticsTeams":   UpsertSeasonStatisticsTeams,
		"upsertRounds":                  UpsertRounds,
		"upsertStages":                  UpsertStages,
		"upsertReferees":                UpsertReferees,
		"upsertRefereesSeasons":         UpsertRefereesSeasons,
		"upsertStates":                  UpsertStates,
		"upsertTypes":                   UpsertTypes,
		"upsertFixturesBetweenDate":     UpsertFixturesBetweenDate,
		"upsertFixturesDate":            UpsertFixturesBetweenDate,
		"upsertFixturesSeasons":         UpsertFixturesSeasons,
		"UpsertFixturesLive":            UpsertFixturesLive,
		"upsertStandings":               UpsertStandings,
		"upsertStandingsSeasons":        UpsertStandings,
		"upsertStandingsLive":           UpsertStandingsLive,
		"upsertBookmakers":              UpsertBookmakers,
		"upsertMarkets":                 UpsertMarkets,
		"upsertOdds":                    UpsertOdds,
		"upsertTopScorers":              UpsertTopScorers,
		"translateTeams":                TranslateTeams,
		"upsertVenues":                  UpsertVenues,
		"upsertFixturesId":              UpsertFixturesId,
		"upsertStatisticsStage":         UpsertStageStatistics,
		"upsertExpectedFixtures":        upsertExpectedFixtures,
		"upsertRivals":                  upsertRivals,
	}
}

// HandlingViolatesFKError function error handling violates foreign key constraint
func HandlingViolatesFKError(err error, query string) error {
	if err == nil {
		return nil
	}

	reReferenced := regexp.MustCompile(`.* is not present in table \"([\w_]*)\".`)
	reViolation := regexp.MustCompile(`Key \(.*\)=\(([0-9]*)\) .*`)

	count := 0
	for {
		pgErr, ok := err.(*pgconn.PgError)
		if !ok || pgErr.Code != "23503" {
			return err
		}

		count++
		if count > 10 {
			return errors.New("too many fk violation errors")
		}

		refMatches := reReferenced.FindAllSubmatch([]byte(pgErr.Detail), 1)
		if len(refMatches) == 0 || len(refMatches[0]) < 2 {
			return err
		}

		tableName := string(refMatches[0][1])
		violMatches := reViolation.FindAllSubmatch([]byte(pgErr.Detail), 1)
		if len(violMatches) == 0 || len(violMatches[0]) < 2 {
			return err
		}
		violationID := string(violMatches[0][1])

		var violationTask config.TaskData
		violationTask.Init()
		violationTask.Args.Include = "none"

		switch tableName {
		case "provider_fixtures":
			violationTask.Args.TaskName = "upsertFixturesId"
			violationTask.Args.FixturesId = violationID
		case "provider_teams":
			violationTask.Args.TaskName = "upsertTeamById"
			violationTask.Args.TeamsId = violationID
		case "provider_coachs":
			violationTask.Args.TaskName = "upsertCoachesIds"
			violationTask.Args.CoachesId = violationID
		case "provider_players":
			violationTask.Args.TaskName = "upsertPlayerIds"
			violationTask.Args.PlayersId = violationID
		default:
			// Неизвестная таблица для авто-дозагрузки — возвращаем ошибку
			return err
		}

		EndPoint(violationTask)
		_, err = db.DBx.PGxPool.Exec(context.Background(), query)
		if err == nil {
			return nil
		}
	}
}
