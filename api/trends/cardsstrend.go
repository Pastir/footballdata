package trends

/*
import (
	"fmt"
	"github.com/lib/pq"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/db/queries"
	"github.com/Pastir/footballdata/utility"
	"os"
)

const YellowCardsMarket = "Yellow Cards Over Under"

func CardsTotalTrends(matchData db.ListMatches, season string) error {
	//Поиск трендов
	h2hHome, _ := CardsH2HTotalTrends(matchData, season, "h_h2h")
	h2hAway, _ := CardsH2HTotalTrends(matchData, season, "a_h2h")
	h2h, _ := CardsH2HTotalTrends(matchData, season, "h2h")

	//Структура для хранения одной из Home или Away команд
	var HomeAway []TrendData

	//Количество трендов по типам
	h2hHomeSize := len(h2hHome)
	h2hAwaySize := len(h2hAway)
	h2hSize := len(h2h)

	//Если найдены индивидуальные тоталы обоих команд, выбираем какой из них используем
	if h2hHomeSize > 0 && h2hAwaySize > 0 {
		if utility.Rand(2) == 1 {
			HomeAway = h2hHome
		} else {
			HomeAway = h2hAway
		}
	} else if h2hHomeSize > 0 && h2hAwaySize == 0 {
		HomeAway = h2hHome
	} else if h2hHomeSize == 0 && h2hAwaySize > 0 {
		HomeAway = h2hAway
	}

	// 1) Если индивидуальные тренды не найдены а тренды h2h найдены, генерим факт
	// 2) Если найдены тренды обоих групп, генерим рандомно тренд и факт
	if len(HomeAway) == 0 && h2hSize > 0 {
		InsertTrendAndFacts(h2h[utility.Rand(h2hSize)], "trend")
	} else if len(HomeAway) > 0 && h2hSize > 0 {
		r := utility.Rand(2)
		//Рандомно решаем вопрос о том какой тренд уходит на факты
		var trendOrFact []string

		if r == 1 {
			trendOrFact = append(trendOrFact, "fact")
			trendOrFact = append(trendOrFact, "trend")
		} else {
			trendOrFact = append(trendOrFact, "trend")
			trendOrFact = append(trendOrFact, "fact")
		}

		InsertTrendAndFacts(HomeAway[utility.Rand(len(HomeAway))], trendOrFact[0])
		InsertTrendAndFacts(h2h[utility.Rand(h2hSize)], trendOrFact[1])
	}
	return nil
}

//CardsH2HTotalTrends Home Corners Over/Under - market_id: 136830894, поиск трендов H2H для хозяев, гостей и игры
func CardsH2HTotalTrends(matchData db.ListMatches, season string, context string) ([]TrendData, error) {

	var aTr []TrendData

	query := fmt.Sprintf(`SELECT   number_match
                                        , number_trend
                                        , market_id
                                        , total
                                        , percent
                                        , context
                                        , match_id
                                        , team_id
                                        , team_name
                                        , home_team_name
                                        , away_team_name
                                        , league_name
                                        , jsonb_build_array(MAX(total) OVER(), MIN(total) OVER()) AS max_min_trend
                                 FROM search_statistic_h2h_total_trends(
                                          '{"limit": %d,
                                             "league_id": %d,
                                             "home_team_id": %d,
                                             "away_team_id": %d,
                                             "event": "yellowcards",
                                             "context": "%s",
                                             "match_id": %d,
                                             "seasons": "%s",
                                             "market_id": %d,
                                             "number_match": 5
                                            }'::JSONB)
                                 ORDER BY percent DESC`,
		VarTrend.Limit, matchData.LeagueId, matchData.HomeTeamId, matchData.AwayTeamId, context, matchData.MatchId, season, queries.ListMarket[YellowCardsMarket])
    fmt.Println(query)
	os.Exit(0)
	//Выполняем запрос
	rows, err := db.Db.Query(query)
	if err, ok := err.(*pq.Error); ok {
		db.PqErrorError("CardsH2HTotalTrends", err)
		return aTr, err
	}

	//Разбираем ответ
	for rows.Next() {
		var tr TrendData
		err = rows.Scan(&tr.NumberMatch, &tr.NumberTrend, &tr.MarketId, &tr.Total, &tr.Percent, &tr.Context,
			&tr.MatchId, &tr.TeamId, &tr.TeamName, &tr.HomeTeamName, &tr.AwayTeamName, &tr.LeagueName, &tr.MaxMin)

		if err, ok := err.(*pq.Error); ok {
			db.PqErrorError("CardsH2HTotalTrends", err)
			return aTr, err
		}

		// Добавляем маркет
		tr.TypeTrend = YellowCardsMarket
		aTr = append(aTr, tr)
	}

	return aTr, nil
}*/
