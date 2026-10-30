package trends

/*
import (
	"fmt"
	"github.com/lib/pq"
	"github.com/valyala/fastjson"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/db/queries"
	"github.com/Pastir/footballdata/utility"
	"strconv"
	"strings"
)

const ThreeWayHandicap = "Handicap Result"

func HandicapTotalTrends(val db.ListMatches, season string) error {

	/*
		Фора(-) - хозяева, очные встречи 'home_int_minus'
		Фора(+) - хозяева, очные встречи 'home_int_plus'
		Фора(-) - хозяева, очные встречи 'home_float_minus'
		Фора(+) - хозяева, очные встречи 'home_float_plus'
		Фора(-) - гости, очные встречи   'away_int_minus'
		Фора(+) - гости, очные встречи   'away_int_plus'
		Фора(-) - гости, очные встречи   'away_float_minus'
		Фора(+) - гости, очные встречи   'away_float_plus'
*/
/*
	//Home Team
	FoundTrends := make(map[string]HandicapDataTrend)
	FoundTrends["home_int_minus"], _ = SearchHandicapH2hHomeAway(val, season, "home_int_minus", "h_h2h")
	FoundTrends["home_float_minus"], _ = SearchHandicapH2hHomeAway(val, season, "home_float_minus", "h_h2h")
	//если тренды Фора+ найдены, поиск Фора- не имеет смысла.
	if FoundTrends["home_int_minus"].Len == 0 && FoundTrends["home_float_minus"].Len == 0 {
		FoundTrends["home_float_plus"], _ = SearchHandicapH2hHomeAway(val, season, "home_float_plus", "h_h2h")
		FoundTrends["home_int_plus"], _ = SearchHandicapH2hHomeAway(val, season, "home_int_plus", "h_h2h")
	}

	//Away Team
	FoundTrends["away_int_plus"], _ = SearchHandicapH2hHomeAway(val, season, "away_int_plus", "a_h2h")
	FoundTrends["away_float_plus"], _ = SearchHandicapH2hHomeAway(val, season, "away_float_plus", "a_h2h")
	//если тренды Фора- найдены, поиск Фора+ не имеет смысла.
	if FoundTrends["away_int_plus"].Len == 0 && FoundTrends["away_float_plus"].Len == 0 {
		FoundTrends["away_float_minus"], _ = SearchHandicapH2hHomeAway(val, season, "away_float_minus", "a_h2h")
		FoundTrends["away_int_minus"], _ = SearchHandicapH2hHomeAway(val, season, "away_int_minus", "a_h2h")
	}

	//Выбираем если типы трендов Фора не найдены,удаляем из массива
	for k, v := range FoundTrends {
		// Удаляем типы трендов Фора(+/-) которые не найдены
		if v.Len == 0 {
			delete(FoundTrends, k)
		}
	}

	//fmt.Println(len(FoundTrends))
	// Тренды не найдены
	if len(FoundTrends) == 0 {
		return nil
	}

	//Рандомно выбираем генерацию факта или тренда
	WhatText := []string{"trend", "fact"}
	if utility.Rand(2) == 1 {
		WhatText[0] = "fact"
		WhatText[1] = "trend"
	}

	ItogTrend := make(map[string]TrendData)
	//Выбираем фору для хозяев поля.
	for k, v := range FoundTrends {
		// Тренды не найдены
		if v.Len == 0 {
			continue
		}

		// Ищем тренд с самым высоким процентом
		if strings.Index(k, "home") != -1 {
			v.Handicap[0].Label = "1"
			v.Handicap[0].WhatText = WhatText[0]
			ItogTrend["home"] = v.Handicap[0]
			for _, hv := range v.Handicap {
				hv.Label = "1"
				hv.WhatText = WhatText[0]
				if ItogTrend["home"].Percent < hv.Percent {
					ItogTrend["home"] = hv
				}
			}
		}
	}

	//Выбираем фору для гостей.
	for k, v := range FoundTrends {
		// Тренды не найдены
		if v.Len == 0 {
			continue
		}

		if strings.Index(k, "away") != -1 {
			for _, hv := range v.Handicap {
				hv.Label = "2"
				hv.WhatText = WhatText[1]
				if ItogTrend["home"].Total != hv.Total {
					ItogTrend["away"] = hv
				}
			}
		}
	}

	InsertHandicap(ItogTrend)
	return nil
}

func InsertHandicap(dataTrend map[string]TrendData) {

	queryInsertTrend := `INSERT INTO matchcenter_trend (number_match, number_trend,market_id,label, total,percent, context, match_id,team_id, trend_text, xxhash64, types) VALUES `

	for _, val := range dataTrend {

		textJson, _ := fastjson.Parse(queries.ListTrendsText[ThreeWayHandicap])
		var arrayText []*fastjson.Value

		// парсим победы и возвраты
		var WinReturn string
		if len(val.WinReturn) > 0 {
			WinReturn = fmt.Sprintf(", (побед %s, возвратов %s)", val.WinReturn[0], val.WinReturn[1])
		}

		arrayText = textJson.GetArray(val.WhatText, val.Label)
		t, _ := arrayText[utility.Rand(len(arrayText))].StringBytes()
		val.TextTrend = string(t)
		val.TypeTrend = ThreeWayHandicap

		var ItogText string
		if val.Total == "0" {
			ItogText = ReplaceTrendText(val, "fact")
		} else {
			ItogText = ReplaceTrendText(val, val.WhatText)
		}
		ItogText = strings.Replace(ItogText, "[WINRETURN]", WinReturn, 1)

		var sectionValues strings.Builder
		sectionValues.WriteString("(")
		sectionValues.WriteString(strconv.Itoa(val.NumberMatch) + ", ")
		sectionValues.WriteString(strconv.Itoa(val.NumberTrend) + ", ")
		sectionValues.WriteString(strconv.Itoa(val.MarketId) + ", ")
		sectionValues.WriteString("'" + val.Label + "', ")
		sectionValues.WriteString("'" + val.Total + "', ")
		sectionValues.WriteString(strconv.FormatFloat(val.Percent, 'f', 1, 64) + ", ")
		sectionValues.WriteString("'" + val.Context + "', ")
		sectionValues.WriteString(strconv.Itoa(val.MatchId) + ", ")
		sectionValues.WriteString(strconv.Itoa(val.TeamId) + ", ")
		sectionValues.WriteString("'" + ItogText + "', ")

		var h utility.HashData
		h.Init()
		h.MarkertId = val.MarketId
		h.MatchId = val.MatchId
		h.TeamId = val.TeamId
		*h.Label = val.Label
		*h.Total = val.Total
		*h.Context = val.Context
		*h.WhatText = val.WhatText

		sectionValues.WriteString(strconv.FormatUint(utility.GenerateHash64(h), 10) + ", ")
		sectionValues.WriteString("'" + val.WhatText + "'), ")

		if sectionValues.String() != "" {
			queryInsert := queryInsertTrend + strings.Trim(sectionValues.String(), ", ") + " ON CONFLICT (xxhash64) DO NOTHING"
			//Выполняем запрос
			_, err := db.Db.Exec(queryInsert)
			if err, ok := err.(*pq.Error); ok {
				db.PqErrorError("InsertHandicap", err)
			}
		}
	}
}

func SearchHandicapH2hHomeAway(val db.ListMatches, season string, handicap string, context string) (HandicapDataTrend, error) {
	var trend []TrendData
	var s HandicapDataTrend
	query := fmt.Sprintf(`SELECT number_match
                                       ,number_trend
                                       ,market_id
                                       ,total
                                       ,percent
                                       ,context
                                       ,match_id
                                       ,team_id
                                       ,win_return
                                       ,team_name
                                       ,home_team_name
                                       ,away_team_name
                                       ,league_name
                                 FROM h2h_home_away_trends_handicap('{
                                                                       "limit": %d,
                                                                       "league_id": %d,
                                                                       "home_team_id": %d,
                                                                       "away_team_id": %d,
                                                                       "context": "%s",
                                                                       "match_id": %d,
                                                                       "seasons": "%s",
                                                                       "market_id": %d,
                                                                       "number_match": %d,
                                                                       "percent": %f,
                                                                       "handicap": "%s"
                                                                     }'::JSONB)`, VarTrend.Limit, val.LeagueId,
		val.HomeTeamId, val.AwayTeamId, context, val.MatchId,
		season, queries.ListMarket["Handicap Result"],
		VarTrend.NumberMatch, VarTrend.Percent, handicap)

	//Выполняем запрос
	rows, err := db.Db.Query(query)
	if err, ok := err.(*pq.Error); ok {
		db.PqErrorError("SearchHandicapH2hHomeAway", err)
		return s, err
	}

	for rows.Next() {
		var tr TrendData

		err = rows.Scan(&tr.NumberMatch, &tr.NumberTrend, &tr.MarketId, &tr.Total, &tr.Percent, &tr.Context,
			&tr.MatchId, &tr.TeamId, pq.Array(&tr.WinReturn), &tr.TeamName, &tr.HomeTeamName, &tr.AwayTeamName,
			&tr.LeagueName)
		if err, ok := err.(*pq.Error); ok {
			db.PqErrorError("SearchHandicapH2hHomeAway", err)
			return s, err
		}
		trend = append(trend, tr)
	}

	s.Len = len(trend)
	s.Handicap = trend

	return s, nil
}
*/
