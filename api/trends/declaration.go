package trends

type (
	Facts struct {
		Limit       int
		NumberMatch int
		Percent     float64
	}

	FactData struct {
		// Матч ID
		MathId       int
		TeamId       int
		TeamName     string
		HomePosition string
		AwayPosition string
		Label        string
		Value        float64
		MarketId     int
		BookmakerId  int
		Probability  string
		NumberMatch  int
		NumberTrends int
		HomeTeamName string
		AwayTeamName string
		Types        string
		LeagueName   string
		Context      string
	}

	TrendData struct {
		// Количество последних матчей для которых тренд имеет определенный процент
		NumberMatch int

		// Количество матчей в которых тренд подтвержден
		NumberTrend int

		//MarketId - id типа ставки(1X2, Double Chance etc)
		MarketId int

		// Пример: маркет "3Way Result" (1X2), 1, X и 2 - являются указателями(Label)
		Label string

		//Total
		Total string

		// Вероятность, отношение подтвержденных трендов к последним матчам
		Percent float64

		// Среднее количество события за матч
		AvgEvent float64

		WinReturn    []string
		MaxMin       string `json:"max_min_trend"`
		TypeTrend    string
		TextTrend    string
		TypeTotal    string

		// Флаг для определения принадлежности текста(trend или fact)
		WhatText string

		// Флаг
		Exists   bool

		TeamId   int
		TeamName string
	}

	Trend struct {
		Limit       int
		NumberMatch int
		Percent     float64
	}

	HandicapDataTrend struct {
		Handicap []TrendData
		Len      int
		HomeAway string
	}
)

var VarFacts = Facts{Limit: 20, NumberMatch: 5, Percent: 80}
var VarTrend = Trend{Limit: 20, NumberMatch: 5, Percent: 80}
