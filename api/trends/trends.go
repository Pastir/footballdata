package trends

/*
import (
	"github.com/lib/pq"
	"github.com/valyala/fastjson"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/db/queries"
	"github.com/Pastir/footballdata/utility"
	"math/rand"
	"strconv"
	"strings"
)

func ReplaceTrendText(data TrendData, typeText string) string {

	// Если количество тренда равно матчам пишем тольк количество матчей инача "количество трендов" из "количество матчей"
	var trendText string
	var Location string
	var word string

	if data.NumberMatch == data.NumberTrend {
		trendText = strconv.Itoa(data.NumberMatch)
	} else {
		trendText = strconv.Itoa(data.NumberTrend) + " из " + strconv.Itoa(data.NumberMatch)
	}

	//Text
	tJ, _ := fastjson.Parse(queries.ListTrendsText[data.TypeTrend])
	textJson := tJ.GetArray(typeText, data.Label)

	//Подготавливаем текст
	lenText := len(textJson)
	if lenText > 0 {
		//Рандомно выбираем текст
		TextItog, _ := textJson[utility.Rand(lenText)].StringBytes()

		word = strings.Replace(string(TextItog), "[HOME_AWAY]", data.HomeAway, 1)
		word = strings.Replace(word, "[TEAM]", data.TeamName, 1)
		word = strings.Replace(word, "[TREND]", trendText, 1)

		ln, _ := fastjson.Parse(data.LeagueName)
		lnSize := ln.GetArray()
		rn := rand.Intn(len(lnSize))
		leagueName, _ := lnSize[rn].StringBytes()

		word = strings.Replace(word, "[LEAGUE]", string(leagueName), 1)
		word = strings.Replace(word, "[SEASON]", data.Season, 1)
		word = strings.Replace(word, "[TEAM1]", data.HomeTeamName, 1)
		word = strings.Replace(word, "[TEAM2]", data.AwayTeamName, 1)
		word = strings.Replace(word, "[CARDS]", "желтых карточек", 1)
		word = strings.Replace(word, "[TOTAL]", data.Total, 1)

		// Локация игры
		switch data.Context {
		case "h2h":
		case "a_h2h", "h_h2h":
			//Определяем текст места игры команды
			var l []*fastjson.Value
			if data.Context == "a_h2h" {
				l = tJ.GetArray(typeText, "LocationAway")
			} else {
				l = tJ.GetArray(typeText, "LocationHome")
			}

			rnText, _ := l[utility.Rand(len(l))].StringBytes()
			Location = string(rnText)
		}

		word = strings.Replace(word, "[LOCATION]", Location, 1)
	}

	return word
}

//InsertTrendData - Функция сохраняет данные по тренду в базу
func InsertTrendData(trendData []TrendData) {
	queryInsert := `INSERT INTO matchcenter_trend(number_match, number_trend,market_id,label, total, percent, context, match_id,team_id, trend_text, xxhash64, types) VALUES `

	for _, v := range trendData {
		var sectionValues strings.Builder
		sectionValues.WriteString("(")
		sectionValues.WriteString(strconv.Itoa(v.NumberMatch) + ", ")
		sectionValues.WriteString(strconv.Itoa(v.NumberTrend) + ", ")
		sectionValues.WriteString(strconv.Itoa(v.MarketId) + ", ")
		sectionValues.WriteString("'" + v.Label + "', ")
		// Тотал, если отсутсвует пустая строка
		sectionValues.WriteString("'" + v.Total + "', ")
		sectionValues.WriteString(strconv.FormatFloat(v.Percent, 'f', 1, 64) + ", ")
		sectionValues.WriteString("'" + v.Context + "', ")
		sectionValues.WriteString(strconv.Itoa(v.MatchId) + ", ")
		sectionValues.WriteString(strconv.Itoa(v.TeamId) + ", ")
		sectionValues.WriteString("'" + v.TextTrend + "', ")

		var h utility.HashData
		h.Init()
		h.MarkertId = v.MarketId
		h.MatchId = v.MatchId
		h.TeamId = v.TeamId
		*h.Label = v.Label
		*h.Total = v.Total
		*h.Context = v.Context
		*h.WhatText = v.WhatText

		sectionValues.WriteString(strconv.FormatUint(utility.GenerateHash64(h), 10) + ", ")
		sectionValues.WriteString("'" + v.WhatText + "'), ")

		if sectionValues.String() != "" {
			queryInsertTrend := queryInsert + strings.Trim(sectionValues.String(), ", ") + " ON CONFLICT (xxhash64) DO NOTHING"

			//Выполняем запрос
			_, err := db.Db.Exec(queryInsertTrend)
			if err, ok := err.(*pq.Error); ok {
				db.PqErrorError("InsertTrendData", err)
			}
		}
	}
}

func MatchIsVisible(matchId int) {
	query := `UPDATE matchcenter_match SET is_visible = 't' WHERE id = $1`
	_, err := db.Db.Exec(query, matchId)
	if err, ok := err.(*pq.Error); ok {
		db.PqErrorError("MatchIsVisible", err)
	}
}*/
