package main

import (
	"fmt"
	"time"

	"github.com/Pastir/ManageServer"
	handler "github.com/Pastir/footballdata/api/handlers"
	"github.com/Pastir/footballdata/api/statistics"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/logs"
	"go.uber.org/zap"
)

func main() {
	var mServer = ManageServer.ServiceWatcher{}
	mServer.AddService(&config.Config)
	mServer.AddService(&logs.Logs)
	mServer.AddService(&db.DBx)
	mServer.AddService(&db.Prov)

	err := mServer.InitServices()
	if err != nil {
		fmt.Println(err)
		return
	}

	//statistics.HandlerOverCorners()
	//os.Exit(0)
	var taskData config.TaskData
	taskData.Init()
	taskData.Args = config.Config.Args
	logs.Logs["access"].Info("info", zap.String("ts", time.Now().String()),
		zap.String("task", taskData.Name()),
		zap.String("event", "start"))

	switch taskData.Name() {
	case "upsertHead2Head":
		handler.UpsertHead2Head(taskData)
		break
	case "prematchStatistics":
		statistics.PrematchStatistics(taskData)

	default:
		handler.EndPoint(taskData)
	}

}

/*
  -task upsertOdds -startDate 2023-05-05 -endDate 2023-05-06
  -task upsertStandingsSeasons  -seasonIsCurrent true
  -task upsertTypes
  -task upsertTopScorers
  -task upsertBookmakers
  -task upsertLeagues
  -task upsertSeasons
  -task upsertRounds
  -task upsertStages
  -task upsertSeasonsTeams -seasonIsCurrent true
  -task upsertPlayersSeasons -seasonIsCurrent true
  -task upsertCoachesTeamsSeasons -seasonIsCurrent true
  -task upsertCoachesIds
  -task upsertRefereesSeasons -seasonIsCurrent true
  -task upsertPlayerIds -playerId 37673230,37675253,37675754e
  -task upsertFixtures
  -task upsertFixturesSeasons -seasonIsCurrent true
       -task upsertFixturesSeasonsTimeline -seasonIsCurrent true
       -task upsertFixturesSeasonsStatistics -seasonIsCurrent true
       -task upsertFixtureSeasonsPeriodsStatistics -seasonIsCurrent true
  -task upsertExpectedFixtures -startDate 2026-09-10 -endDate 2026-09-11
*/
