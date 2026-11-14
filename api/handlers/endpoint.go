package handler

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Pastir/footballdata/client"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/db/queries"
	"github.com/Pastir/footballdata/logs"
	"github.com/valyala/fastjson"
	"go.uber.org/zap"
)

func EndPoint(taskData config.TaskData) {

	switch db.Prov.Endpoints[taskData.Name()].Cycle {
	case "seasons":
		seasonsId := queries.GetSeasons(taskData)
		for k, _ := range seasonsId {
			taskData.SetReplace("season", k)
			RequestEndpoint(taskData)
		}

	case "countries":
		countriesIds := queries.GetListCountriesList()

		for _, v := range countriesIds {
			taskData.SetReplace("countries", v)
			RequestEndpoint(taskData)
		}

	case "teamsID":
		teamIds := strings.Split(taskData.GetTeamId(), ",")

		for _, v := range teamIds {
			ids, _ := strconv.Atoi(v)
			taskData.SetReplace("team", ids)
			RequestEndpoint(taskData)
		}

	case "seasons,teams":
		seasonsId := queries.GetSeasons(taskData)
		for k, _ := range seasonsId {
			for _, teamV := range queries.GetTeamsIdFromSeasons(k) {
				taskData.SetReplace("season", k)
				taskData.SetReplace("team", teamV)
				RequestEndpoint(taskData)
			}
		}

	case "players":
		playerIds := strings.Split(taskData.GetPlayerId(), ",")
		for _, v := range playerIds {
			ids, _ := strconv.Atoi(v)
			taskData.SetReplace("players", ids)
			RequestEndpoint(taskData)
		}

	case "date":
		taskData.SetReplace("start_date", taskData.GetStartDate())
		taskData.SetReplace("end_date", taskData.GetEndDate())
		taskData.SetReplace("date", taskData.GetDate())
		RequestEndpoint(taskData)

	case "coaches":
		coachesIds := strings.Split(taskData.GetCoachesId(), ",")
		for _, v := range coachesIds {
			ids, _ := strconv.Atoi(v)
			taskData.SetReplace("coaches", ids)
			RequestEndpoint(taskData)
		}

	case "rounds":
		var roundsIds []int64
		roundsIds, _ = queries.GetLast3CurrentSeasonRoundsIDs()

		for _, v := range roundsIds {
			taskData.SetReplace("rounds", v)
			RequestEndpoint(taskData)
		}

	case "fixtures":
		fixturesIds := strings.Split(taskData.GetFixtureId(), ",")
		for _, v := range fixturesIds {
			ids, _ := strconv.Atoi(v)
			taskData.SetReplace("fixtures", ids)
			RequestEndpoint(taskData)
		}

	case "stage":
		stagesIds := GetStagesOfSeason(taskData.GetCurrentSeason())
		for _, v := range stagesIds {
			taskData.SetReplace("stage", v)
			RequestEndpoint(taskData)
		}

	case "leagues":
		leagueIds := queries.GetActiveLeagueIDs(taskData)
		for _, v := range leagueIds {
			taskData.SetReplace("leagues", v)
			RequestEndpoint(taskData)
		}

	default:
		RequestEndpoint(taskData)
	}

	TaskLastUpdate(taskData)
}

func RequestEndpoint(taskData config.TaskData) {

	var paginationData PaginationData
	var rateLimit RateLimit
	paginationData.Init()
	rateLimit.Init()
	getParams := url.Values{}
	for k, v := range db.Prov.Params {
		getParams[k] = append([]string(nil), v...)
	}
	endpointData := db.Prov.Endpoints[taskData.Name()]
	getParams.Set("page", "1")
	var filter string

	// add parameters include
	if val, ok := db.Prov.Endpoints[taskData.Name()].Additional["include"]; ok {
		getParams.Set("include", val.(string))
	}

	if strings.Compare(taskData.Args.Include, "") != 0 {
		getParams.Del("include")
		getParams.Set("include", taskData.Args.Include)
	}

	if strings.Compare(taskData.Args.Include, "none") == 0 {
		getParams.Del("include")
	}

	if val, ok := db.Prov.Endpoints[taskData.Name()].Additional["filters"]; ok {
		if _, okS := taskData.Replace["season"]; okS {
			filter = strings.Replace(val.(string), "{seasonID}", strconv.Itoa(taskData.Replace["season"].(int)), 1)
		} else {
			filter = val.(string)
		}
		getParams.Set("filters", filter)
	}
	// Set console
	if strings.Compare(taskData.Args.Locale, "") == 0 {
		getParams.Del("locale")
	} else {
		getParams.Set("locale", taskData.Args.Locale)
	}

	// pagination
	for {
		taskData.Cycle = *paginationData.CurrentPage
		// query constructor
		url := endpointData.Endpoint
		url = url + "?" + getParams.Encode()

		//getParams.Del("include")
		//getParams.Del("filters")
		url = replaceToken(url, taskData.Replace)
		// request to provider endpoint
		fmt.Println(url)
		resp, err := client.RequestToEndpoint(url)
		if err != nil {
			logs.ErrorLog(taskData.Name(), "RequestToEndpoint", err)
			return
		}

		var p fastjson.Parser
		dv, err := p.ParseBytes(resp)
		if err != nil {
			logs.Logs["error"].Error("endpoint parser", zap.Error(err))
			return
		}

		// Handler
		Handlers[endpointData.TaskName](dv, taskData)

		// Rate limit
		RateLimitHandler(dv.Get("rate_limit"), &rateLimit)
		if *rateLimit.Remaining == 0 {
			logs.InfoLog(taskData.Name(), "EndPoint", "rate limit exceeded, sleep "+strconv.FormatInt(*rateLimit.ResetsInSeconds, 10))
			time.Sleep(time.Duration(*rateLimit.ResetsInSeconds+60) * time.Second)
		} else {
			logs.Logs["access"].Info("meta",
				zap.String("ts", time.Now().String()),
				zap.String("task", taskData.Name()),
				zap.Int("remaining", *rateLimit.Remaining),
				zap.Int64("reset", *rateLimit.ResetsInSeconds),
			)
		}

		// Pagination
		PaginationHandler(dv.Get("pagination"), &paginationData)
		if !*paginationData.HasMore {
			return
		}

		getParams.Set("page", strconv.Itoa(*paginationData.CurrentPage+1))
	}

}

func TaskLastUpdate(taskData config.TaskData) {
	query := "UPDATE provider_endpoints SET updated_at = NOW() WHERE task_name = $1"
	_, err := db.DBx.PGxPool.Exec(context.Background(), query, taskData.Name())
	if err != nil {
		db.PGxConnError(taskData.Name(), "TaskLastUpdate", err)
	}
}

func formatTokenValue(v interface{}) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case int:
		return strconv.Itoa(val), true
	case int64:
		return strconv.FormatInt(val, 10), true
	default:
		return "", false
	}
}

func replaceToken(url string, replaceTokens map[string]interface{}) string {
	tokenPlaceholders := map[string]string{
		"start_date": "{start_date}",
		"end_date":   "{end_date}",
		"date":       "{date}",
		"season":     "{seasonID}",
		"team":       "{teamID}",
		"players":    "{playersID}",
		"coaches":    "{coachesID}",
		"countries":  "{countryID}",
		"rounds":     "{roundID}",
		"fixtures":   "{fixtureID}",
		"stage":      "{stageID}",
		"leagues":    "{leagueID}",
	}

	for key, v := range replaceTokens {
		placeholder, ok := tokenPlaceholders[key]
		if !ok {
			continue
		}
		if strVal, valid := formatTokenValue(v); valid {
			url = strings.Replace(url, placeholder, strVal, 1)
		}
	}

	return url
}
