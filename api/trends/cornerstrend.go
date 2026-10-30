package trends

/*
import (
	"fmt"
	"github.com/lib/pq"
	"github.com/valyala/fastjson"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/db/queries"
	"github.com/Pastir/footballdata/utility"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

var CornersMarket = map[string]int {
  "Corners Over Under": 976384,
  "Home Corners Over/Under": 136830894,
  "Away Corners Over/Under": 136830897,
}

func CornersTotalTrends(matchData db.ListMatches, season string) error {
	var (
		total TrendData
	    home  TrendData
	    away  TrendData
	)

	//Поиск трендов
	home.MarketId = CornersMarket["Home Corners Over/Under"]
	H2HCornersHomeTotalTrends(matchData, &home)

	away.MarketId = CornersMarket["Away Corners Over/Under"]
	H2HCornersAwayTotalTrends(matchData, &away)

	total.MarketId = CornersMarket["Corners Over Under"]
	H2HCornersTotalTrends(matchData, &total)

	//Структура для хранения одной из Home или Away команд
	var HomeAway TrendData

	//Если найдены индивидуальные тоталы обоих команд, выбираем какой из них используем
	if home.NumberTrend > 0 && away.NumberMatch > 0 {
		if utility.Rand(2) == 1 {
			HomeAway = home
		} else {
			HomeAway = away
		}
	} else if home.NumberTrend > 0 && away.NumberTrend == 0 {
		HomeAway = home
	} else if home.NumberTrend == 0 && away.NumberTrend > 0 {
		HomeAway = away
	}

	// 1) Если индивидуальные тренды не найдены а тренд матча найден, генерим тренд
	// 2) Если найдены тренды обоих групп, для тотал матча генерим тренд, а для индивидуального тотала факт
	if HomeAway.NumberTrend == 0 && total.NumberTrend > 0 {
		total.WhatText = "trend"
		InsertTrendAndFacts(matchData, total)
	} else if HomeAway.NumberTrend > 0 && total.NumberTrend > 0 {
		//Рандомно решаем вопрос о том какой тренд уходит на факты
		r := utility.Rand(2)
		if r == 1 {
			HomeAway.WhatText = "trend"
			total.WhatText = "fact"
		} else {
			HomeAway.WhatText = "fact"
			total.WhatText = "trend"
		}
	}

	InsertTrendAndFacts(matchData, HomeAway)
	InsertTrendAndFacts(matchData, total)
	return nil
}

//InsertTrendAndFacts - функция генерит тренды или факты исходя из найденных тенденций
func InsertTrendAndFacts(matchData db.ListMatches, data TrendData) {

	floatTotal, _ := strconv.ParseFloat(data.Total, 10)

	//Text
	tJ, _ := fastjson.Parse(queries.ListTrendsText[data.TypeTrend])
	textJson := tJ.Get(data.WhatText)

	//Массив текстов
	var textTrend []*fastjson.Value

	//Определяем минимальный и максимальный тотал
	maxMinTotal, _ := fastjson.Parse(data.MaxMin)
	max, _ := maxMinTotal.Get("0").StringBytes()
	//min, _ := maxMinTotal.Get("1").StringBytes()

	maxTotal, _ := strconv.ParseFloat(string(max), 64)
	//minTotal, _ := strconv.ParseFloat(string(min), 64)

	if floatTotal == maxTotal {
		if utility.Rand(2) == 1 {
			data.Label = "Under"
			floatTotal += 1.0
		} else {
			data.Label = "Over"
		}
	} else if floatTotal < maxTotal {
		data.Label = "Over"
	}

	//Окончательная строка текста тренда или факта
	var FinalText string
	var Location string

	switch data.MarketId {
	case CornersMarket["Corners Over Under"]:
		textTrend = textJson.GetArray(data.Label)

		if len(textTrend) == 0 {
			return
		}
		rnText := utility.Rand(len(textTrend))
		t, _ := textTrend[rnText].StringBytes()
		FinalText = string(t)

	case CornersMarket["Home Corners Over/Under"]:
		textTrend = textJson.GetArray(data.Label)
		if len(textTrend) == 0 {
			return
		}
		rnText := utility.Rand(len(textTrend))
		t, _ := textTrend[rnText].StringBytes()
		FinalText = string(t)

		//Определяем текст места игры команды
		var l []*fastjson.Value
		l = textJson.GetArray("LocationHome")
		rnText = utility.Rand(len(l))
		t, _ = l[rnText].StringBytes()
		Location = string(t)


	case CornersMarket["Away Corners Over/Under"]:
		textTrend = textJson.GetArray(data.Label)
		if len(textTrend) == 0 {
			return
		}

		rnText := utility.Rand(len(textTrend))
		t, _ := textTrend[rnText].StringBytes()
		FinalText = string(t)

		//Определяем текст места игры команды
		var l []*fastjson.Value
        l = textJson.GetArray("LocationAway")
		rnText = utility.Rand(len(l))
		t, _ = l[rnText].StringBytes()
		Location = string(t)
	}

	queryInsert := `INSERT INTO matchcenter_trend(number_match, number_trend,market_id,label, total,percent, match_id,team_id, trend_text, xxhash64, types) VALUES `
	// Если количество тренда равно матчам пишем тольк количество матчей инача "количество трендов" из "количество матчей"
	var trend string
	if data.NumberMatch == data.NumberTrend {
		trend = strconv.Itoa(data.NumberMatch)
	} else {
		trend = strconv.Itoa(data.NumberTrend) + " из " + strconv.Itoa(data.NumberMatch)
	}

	word := strings.Replace(FinalText, "[TEAM]", data.TeamName, 1)
	word = strings.Replace(word, "[TREND]", trend, 1)

	ln, _ := fastjson.Parse(matchData.LeagueName)
	lnSize := ln.GetArray()
	rn := rand.Intn(len(lnSize))
	leagueName, _ := lnSize[rn].StringBytes()

	word = strings.Replace(word, "[LEAGUE]", string(leagueName), 1)
	word = strings.Replace(word, "[SEASON]", matchData.Season, 1)
	word = strings.Replace(word, "[TEAM1]", matchData.HomeTeamName, 1)
	word = strings.Replace(word, "[TEAM2]", matchData.AwayTeamName, 1)
	word = strings.Replace(word, "[CARDS]", "желтых карточек", 1)

	if math.Mod(floatTotal, 1.0) > 0 {
		word = strings.Replace(word, "[TOTAL]", strconv.FormatFloat(floatTotal, 'f', 1, 64), 1)
	} else {
		word = strings.Replace(word, "[TOTAL]", strconv.FormatFloat(floatTotal, 'f', 0, 64), 1)
	}
	word = strings.Replace(word, "[LOCATION]", Location, 1)

	var sectionValues strings.Builder
	sectionValues.WriteString("(")
	sectionValues.WriteString(strconv.Itoa(data.NumberMatch) + ", ")
	sectionValues.WriteString(strconv.Itoa(data.NumberTrend) + ", ")
	sectionValues.WriteString(strconv.Itoa(data.MarketId) + ", ")
	sectionValues.WriteString("'" + data.Label + "', ")
	sectionValues.WriteString("'" + strconv.FormatFloat(floatTotal, 'f', 1, 64) + "', ")
	sectionValues.WriteString(strconv.FormatFloat(data.Percent, 'f', 1, 64) + ", ")
	sectionValues.WriteString(strconv.Itoa(matchData.MatchId) + ", ")
	sectionValues.WriteString(strconv.Itoa(data.TeamId) + ", ")
	sectionValues.WriteString("'" + word + "', ")

	var h utility.HashData
	h.Init()
	h.MarkertId = data.MarketId
	h.MatchId = matchData.MatchId
	h.TeamId = data.TeamId
	*h.WhatText = data.WhatText

	sectionValues.WriteString(strconv.FormatUint(utility.GenerateHash64(h), 10) + ", ")
	sectionValues.WriteString("'" + data.WhatText + "'), ")

	if sectionValues.String() != "" {
		queryInsertTrend := queryInsert + strings.Trim(sectionValues.String(), ", ") + " ON CONFLICT (xxhash64) DO NOTHING"

		//Выполняем запрос
		_, err := db.Db.Exec(queryInsertTrend)
		if err, ok := err.(*pq.Error); ok {
			db.PqErrorError("InsertTrendAndFacts", err)
		}
	}
}

//H2HCornersTotalTrends - поиск трендов: тотал - угловые
func H2HCornersTotalTrends(matchData db.ListMatches, tD *TrendData) error {

	query := `SELECT itog.trend_total
                     , itog.value
                     , itog.matches_played
                     , itog.avg_event
                     , itog.league_name
                     , ROUND(trend_total::NUMERIC(10,2)/matches_played::NUMERIC(10,2) *100, 2) AS percent
              FROM(

                  WITH match AS (
                  SELECT COUNT(*) OVER (PARTITION BY mcm.league_uid)                         AS matches_played
                       , ROUND(AVG(mcms1.total_event) OVER (PARTITION BY mcm.league_uid), 2) AS avg_event
                       , mcms1.total_event
                       , , mcm.league_uid
                  FROM matchcenter_match mcm
                           LEFT JOIN matchcenter_season mcs ON mcs.uid = mcm.season_uid
                           LEFT JOIN matchcenter_league mcl ON mcl.uid = mcm.league_uid
                           LEFT JOIN LATERAL ( SELECT SUM(mcms.corners) AS total_event,
                                                      mcms.match_id
                                               FROM matchcenter_matchstatistic mcms
                                               WHERE mcms.match_id = mcm.id
                                               GROUP BY mcms.match_id
                               ) AS mcms1 ON (mcms1.match_id = mcm.id)
                  WHERE mcm.league_id = $3
                    AND DATE(mcm.match_date) < DATE(NOW())
                    AND ((home_team_id = $1 AND away_team_id = $2) OR (home_team_id = $2 AND away_team_id = $1))
                    AND mcm.status = 'FT'::text
                    AND total_event NOTNULL
                  ORDER BY match_date DESC
              ), total AS (
                  SELECT t.s::numeric * 0.5 AS total,
                         t.s::numeric * 0.5 AS r_min,
                         11.5               AS r_max
                  FROM generate_series(15, 23) t(s)
              ) SELECT COUNT(*)                                 AS trend_total
                       , total.total                            AS value
                       , MAX(match.matches_played)              AS matches_played
                       , MAX(match.avg_event)                   AS avg_event
                       , array[altname_1, altname_2, altname_3] AS league_name
                 FROM match LEFT JOIN total ON (match.total_event::numeric > total.total)
                            LEFT JOIN matchcenter_translation AS mctr ON (mctr.uid = match.league_uid)
                 WHERE total.total IS NOT NULL
                 GROUP BY total.total, 5
                  ) AS itog
              WHERE matches_played > 4
                AND (trend_total::NUMERIC(10,2)/matches_played::NUMERIC(10,2) *100) > 70
              ORDER BY (trend_total::NUMERIC(10,2)/matches_played::NUMERIC(10,2) *100) DESC
              LIMIT 1`

	//Выполняем запрос
	rows, err := db.Db.Query(query, matchData.HomeTeamId, matchData.AwayTeamId, matchData.LeagueId)
	if err, ok := err.(*pq.Error); ok {
		db.PqErrorError("H2HCornersTotalTrends", err)
		return err
	}

	//Разбираем ответ
	for rows.Next() {
		err = rows.Scan(&tD.NumberTrend, &tD.Total, &tD.NumberMatch, &tD.AvgEvent, &tD.Percent)

		if err, ok := err.(*pq.Error); ok {
			db.PqErrorError("H2HCornersTotalTrends", err)
			return err
		}
	}

	return nil
}


//H2HCornersHomeTotalTrends - поиск трендов по угловым
func H2HCornersHomeTotalTrends(matchData db.ListMatches, tD *TrendData) error {

	var aTr []TrendData

	query := `SELECT itog.*
                     , ROUND(trend_total::NUMERIC(10,2)/matches_played::NUMERIC(10,2) *100, 2) AS percent
              FROM(

                  WITH match AS (
                  SELECT COUNT(*) OVER (PARTITION BY mcm.league_uid)                         AS matches_played
                       , ROUND(AVG(mcms1.total_event) OVER (PARTITION BY mcm.league_uid), 2) AS avg_event
                       , mcms1.total_event
                  FROM matchcenter_match mcm
                           LEFT JOIN matchcenter_season mcs ON mcs.uid = mcm.season_uid
                           LEFT JOIN matchcenter_league mcl ON mcl.uid = mcm.league_uid
                           LEFT JOIN LATERAL ( SELECT SUM(mcms.corners) AS total_event,
                                                      mcms.match_id
                                               FROM matchcenter_matchstatistic mcms
                                               WHERE mcms.match_id = mcm.id
                                               GROUP BY mcms.match_id
                               ) AS mcms1 ON (mcms1.match_id = mcm.id)
                  WHERE mcm.league_id = $3
                    AND DATE(mcm.match_date) < DATE(NOW())
                    AND ((home_team_id = $1 AND away_team_id = $2) OR (home_team_id = $2 AND away_team_id = $1))
                    AND mcm.status = 'FT'::text
                    AND total_event NOTNULL
                  ORDER BY match_date DESC
              ), total AS (
                  SELECT t.s::numeric * 0.5 AS total,
                         t.s::numeric * 0.5 AS r_min,
                         11.5               AS r_max
                  FROM generate_series(15, 23) t(s)
              ) SELECT COUNT(*)                    AS trend_total
                       , total.total               AS value
                       , MAX(match.matches_played) AS matches_played
                       , MAX(match.avg_event)      AS avg_event
                 FROM match LEFT JOIN total ON (match.total_event::numeric > total.total)
                 WHERE total.total IS NOT NULL
                 GROUP BY total.total
                  ) AS itog
              WHERE matches_played > 4
                AND (trend_total::NUMERIC(10,2)/matches_played::NUMERIC(10,2) *100) > 70
              ORDER BY (trend_total::NUMERIC(10,2)/matches_played::NUMERIC(10,2) *100) DESC
              LIMIT 1`

	//Выполняем запрос
	rows, err := db.Db.Query(query)
	if err, ok := err.(*pq.Error); ok {
		db.PqErrorError("H2HCornersTotalTrends", err)
		return aTr, err
	}

	//Разбираем ответ
	for rows.Next() {
		var tr TrendData
		err = rows.Scan(&tr.NumberMatch, &tr.NumberTrend, &tr.MarketId, &tr.Total, &tr.Percent, &tr.Context,
			&tr.MatchId, &tr.TeamId, &tr.TeamName, &tr.HomeTeamName, &tr.AwayTeamName, &tr.LeagueName, &tr.MaxMin)

		if err, ok := err.(*pq.Error); ok {
			db.PqErrorError("H2HCornersTotalTrends", err)
			return aTr, err
		}
		tr.TypeTrend = CornersTotalMarket
		aTr = append(aTr, tr)
	}

	return aTr, nil
}

//H2HCornersAwayTotalTrends - поиск трендов по угловым
func H2HCornersAwayTotalTrends(matchData db.ListMatches, tD *TrendData) error {

	var aTr []TrendData

	query := `SELECT itog.*
                     , ROUND(trend_total::NUMERIC(10,2)/matches_played::NUMERIC(10,2) *100, 2) AS percent
              FROM(

                  WITH match AS (
                  SELECT COUNT(*) OVER (PARTITION BY mcm.league_uid)                         AS matches_played
                       , ROUND(AVG(mcms1.total_event) OVER (PARTITION BY mcm.league_uid), 2) AS avg_event
                       , mcms1.total_event
                  FROM matchcenter_match mcm
                           LEFT JOIN matchcenter_season mcs ON mcs.uid = mcm.season_uid
                           LEFT JOIN matchcenter_league mcl ON mcl.uid = mcm.league_uid
                           LEFT JOIN LATERAL ( SELECT SUM(mcms.corners) AS total_event,
                                                      mcms.match_id
                                               FROM matchcenter_matchstatistic mcms
                                               WHERE mcms.match_id = mcm.id
                                               GROUP BY mcms.match_id
                               ) AS mcms1 ON (mcms1.match_id = mcm.id)
                  WHERE mcm.league_id = $3
                    AND DATE(mcm.match_date) < DATE(NOW())
                    AND ((home_team_id = $1 AND away_team_id = $2) OR (home_team_id = $2 AND away_team_id = $1))
                    AND mcm.status = 'FT'::text
                    AND total_event NOTNULL
                  ORDER BY match_date DESC
              ), total AS (
                  SELECT t.s::numeric * 0.5 AS total,
                         t.s::numeric * 0.5 AS r_min,
                         11.5               AS r_max
                  FROM generate_series(15, 23) t(s)
              ) SELECT COUNT(*)                    AS trend_total
                       , total.total               AS value
                       , MAX(match.matches_played) AS matches_played
                       , MAX(match.avg_event)      AS avg_event
                 FROM match LEFT JOIN total ON (match.total_event::numeric > total.total)
                 WHERE total.total IS NOT NULL
                 GROUP BY total.total
                  ) AS itog
              WHERE matches_played > 4
                AND (trend_total::NUMERIC(10,2)/matches_played::NUMERIC(10,2) *100) > 70
              ORDER BY (trend_total::NUMERIC(10,2)/matches_played::NUMERIC(10,2) *100) DESC
              LIMIT 1`

	//Выполняем запрос
	rows, err := db.Db.Query(query)
	if err, ok := err.(*pq.Error); ok {
		db.PqErrorError("H2HCornersTotalTrends", err)
		return aTr, err
	}

	//Разбираем ответ
	for rows.Next() {
		var tr TrendData
		err = rows.Scan(&tr.NumberMatch, &tr.NumberTrend, &tr.MarketId, &tr.Total, &tr.Percent, &tr.Context,
			&tr.MatchId, &tr.TeamId, &tr.TeamName, &tr.HomeTeamName, &tr.AwayTeamName, &tr.LeagueName, &tr.MaxMin)

		if err, ok := err.(*pq.Error); ok {
			db.PqErrorError("H2HCornersTotalTrends", err)
			return aTr, err
		}
		tr.TypeTrend = CornersTotalMarket
		aTr = append(aTr, tr)
	}

	return aTr, nil
}*/
