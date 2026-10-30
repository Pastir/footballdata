package trends

/*
import (
	"fmt"
	"github.com/lib/pq"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/db/queries"
)

const Market1X2 = "3Way Result"
const DoubleChance = "Double Chance"

//X12TrendsH2H - поиск трендов по маркету 1X2
func X12TrendsH2H(matchData db.ListMatches, season string) {

	trend := make(map[string][]TrendData)
	query := fmt.Sprintf(`SELECT number_match
                                        ,number_trend
                                        ,percent
                                        ,home_team_name
                                        ,away_team_name
                                        ,home_team_id
                                        ,away_team_id
                                        ,league_name
                                        ,type_trend
                                        ,min_max
                                 FROM h2h_home_trends_1x2('{"limit": %d,
                                                            "league_id": %d,
                                                             "home_team_id": %d,
                                                             "away_team_id": %d,
                                                             "number_match": %d,
                                                             "percent": %f
                                                           }'::JSONB)`, VarTrend.Limit, matchData.LeagueId, matchData.HomeTeamId,
		matchData.AwayTeamId, VarTrend.NumberMatch, VarTrend.Percent)

	//Выполняем запрос
	rows, err := db.Db.Query(query)
	if err, ok := err.(*pq.Error); ok {
		db.PqErrorError("X12TrendsH2H", err)
		return
	}

	//Разбираем ответ
	for rows.Next() {
		var tr TrendData
		err = rows.Scan(&tr.NumberMatch, &tr.NumberTrend, &tr.Percent, &tr.HomeTeamName, &tr.AwayTeamName,
			&tr.TeamHomeId, &tr.TeamAwayId, &tr.LeagueName, &tr.TypeTrend, &tr.MaxMin)

		if err, ok := err.(*pq.Error); ok {
			db.PqErrorError("X12TrendsH2H", err)
			return
		}

		tr.MatchId = matchData.MatchId
		tr.Season = season
		trend[tr.TypeTrend] = append(trend[tr.TypeTrend], tr)
	}

	// Удаляем пустые типы трендов
	for key, val := range trend {
		if len(val) == 0 {
			delete(trend, key)
		}
	}

	// Факты для 1X2
	var ItogTrends []TrendData
	var facts1x2 TrendData
	facts1x2.HomeTeamName = matchData.HomeTeamName
	facts1x2.AwayTeamName = matchData.AwayTeamName
	facts1x2.LeagueName = matchData.LeagueName
	facts1x2.MatchId = matchData.MatchId
	facts1x2.MarketId = queries.ListMarket[Market1X2]
	facts1x2.TypeTrend = Market1X2
	facts1x2.Label = "1"
	facts1x2.Context = "h_h2h"
	facts1x2.WhatText = "fact"
	facts1x2.TextTrend = ReplaceTrendText(facts1x2, "fact")
	ItogTrends = append(ItogTrends, facts1x2)
	facts1x2.Label = "X"
	facts1x2.TextTrend = ReplaceTrendText(facts1x2, "fact")
	ItogTrends = append(ItogTrends, facts1x2)
	facts1x2.Label = "2"
	facts1x2.Context = "a_h2h"
	facts1x2.TextTrend = ReplaceTrendText(facts1x2, "fact")
	ItogTrends = append(ItogTrends, facts1x2)

	ItogTrends = append(ItogTrends, SelectTrend1X2(trend, &matchData)...)
	InsertTrendData(ItogTrends)
}

/*
win
unbeaten    -- побед + ничьи
lose_streak -- проигрыши
not_win     -- проигрыши + ничьи
draw        -- ничьи
*/
/*
func SelectTrend1X2(trends map[string][]TrendData, matchData *db.ListMatches) []TrendData {
	var itogTrends []TrendData
	// Тренд побед домашней команды, выбираем лучший процент
	if _, ok := trends["home_win"]; ok {
		trends["home_win"][0].Percent -= 1
		trends["home_win"][0].Context = "h_h2h"
		trends["home_win"][0].MarketId = queries.ListMarket[Market1X2]
		trends["home_win"][0].Label = "1"
		trends["home_win"][0].TypeTrend = Market1X2
		trends["home_win"][0].TeamId = matchData.HomeTeamId
		trends["home_win"][0].TextTrend = ReplaceTrendText(trends["home_win"][0], "trend")
		trends["home_win"][0].WhatText = "trend"
		itogTrends = append(itogTrends, trends["home_win"][0])
	}

	//Тренд не проигрыш домашней команды, выбираем самый длинный тренд, процент может быть не самым высоким
	if _, ok := trends["home_unbeaten"]; ok {
		MaxTrends := trends["home_unbeaten"][0]
		for _, v := range trends["home_unbeaten"][1:] {
			if v.NumberTrend >= MaxTrends.NumberTrend {
				MaxTrends = v
			}
		}

		MaxTrends.Percent -= 1
		MaxTrends.Context = "h_h2h"
		MaxTrends.MarketId = queries.ListMarket[DoubleChance]
		MaxTrends.Label = "1X"
		MaxTrends.TypeTrend = DoubleChance
		MaxTrends.TeamId = matchData.HomeTeamId

		//Проверяем на дубликат трендов
		if len(itogTrends) > 0 {
			if MaxTrends.NumberTrend != itogTrends[0].NumberTrend && MaxTrends.NumberMatch != itogTrends[0].NumberMatch {
				MaxTrends.TextTrend = ReplaceTrendText(MaxTrends, "fact")
				MaxTrends.WhatText = "fact"
				itogTrends = append(itogTrends, MaxTrends)

			}
		} else {
			MaxTrends.TextTrend = ReplaceTrendText(MaxTrends, "trend")
			MaxTrends.WhatText = "trend"
			itogTrends = append(itogTrends, MaxTrends)
		}
	}

	if len(itogTrends) == 0 {
		// Тренд побед гостей
		if _, ok := trends["home_lose_streak"]; ok {
			trends["home_lose_streak"][0].Percent -= 1
			trends["home_lose_streak"][0].Context = "a_h2h"
			trends["home_lose_streak"][0].MarketId = queries.ListMarket[Market1X2]
			trends["home_lose_streak"][0].Label = "2"
			trends["home_lose_streak"][0].TypeTrend = Market1X2
			trends["home_lose_streak"][0].TeamId = matchData.AwayTeamId
			trends["home_lose_streak"][0].WhatText = "trend"
			trends["home_lose_streak"][0].TextTrend = ReplaceTrendText(trends["home_lose_streak"][0], "trend")
			itogTrends = append(itogTrends, trends["home_lose_streak"][0])
		}

		//Тренд не проигрыш гостевой команды, выбираем самый длинный тренд, процент может быть не самым высоким
		if _, ok := trends["home_not_win"]; ok {
			MaxTrends := trends["home_not_win"][0]
			for _, v := range trends["home_not_win"][1:] {
				if v.NumberTrend >= MaxTrends.NumberTrend {
					MaxTrends = v
				}
			}

			MaxTrends.Percent -= 1
			MaxTrends.Context = "a_h2h"
			MaxTrends.MarketId = queries.ListMarket[DoubleChance]
			MaxTrends.Label = "X2"
			MaxTrends.TypeTrend = DoubleChance
			MaxTrends.TeamId = matchData.AwayTeamId

			//Проверяем на дубликат трендов
			if len(itogTrends) > 0 {
				if MaxTrends.NumberTrend != itogTrends[0].NumberTrend && MaxTrends.NumberMatch != itogTrends[0].NumberMatch {
					MaxTrends.WhatText = "fact"
					MaxTrends.TextTrend = ReplaceTrendText(MaxTrends, "fact")
					itogTrends = append(itogTrends, MaxTrends)

				}
			} else {
				MaxTrends.TextTrend = ReplaceTrendText(MaxTrends, "trend")
				MaxTrends.WhatText = "trend"
				itogTrends = append(itogTrends, MaxTrends)
			}
		}
	}
	return itogTrends
}*/
