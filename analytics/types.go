package analytics

// Standing — текущее положение команды в таблице.
//
// Rule — человекочитаемое название из provider_standings_rules,
// например:
//
//	UEFA Champions League
//	UEFA Europa League
//	Relegation
type Standing struct {
	ID int64

	TeamID      int64 // participant_id
	TeamName    string
	TeamLogoURL string

	Points   int
	Played   int // сыграно матчей (OVERALL_MATCHES)
	Position int
	Trend    int // изменение позиции: prev_position - curr_position

	Won   int      // победы (OVERALL_WINS)
	Drawn int      // ничьи (OVERALL_DRAWS)
	Lost  int      // поражения (OVERALL_LOST)
	GF    int      // забито (OVERALL_SCORED)
	GA    int      // пропущено (OVERALL_CONCEDED)
	GD    int      // разница мячей (OVERALL_GOAL_DIFFERENCE)
	Form  []string // форма последних 5 матчей от старого к новому ["W", "D", "L", ...]

	Rule           string // человекочитаемое название зоны (psr.name)
	StandingRuleID int
	RuleDevName    string // из provider_types.developer_name / provider_standings_rules
}

// StandingDetails — детализация статистики (из provider_standings_details).
type StandingDetails struct {
	Played int
	Won    int
	Drawn  int
	Lost   int
	GF     int // Goals For  (OVERALL_SCORED)
	GA     int // Goals Against (OVERALL_CONCEDED)
	GD     int // Goal Difference (OVERALL_GOAL_DIFFERENCE)
}

// StatCard — карточка Key Stats.
type StatCard struct {
	Label string
	Team  string
	Value string
}

// RoundExportPayload описывает структуру выгрузки данных тура для Telegram Mini App.
type RoundExportPayload struct {
	LeagueID      int                `json:"league_id"`
	LeagueName    string             `json:"league_name"`
	SeasonID      int                `json:"season_id"`
	SeasonName    string             `json:"season_name"`
	RoundID       int                `json:"round_id"`
	RoundName     string             `json:"round_name"`
	PrevRoundID   int                `json:"prev_round_id,omitempty"`
	PrevRoundName string             `json:"prev_round_name,omitempty"`
	NextRoundID   int                `json:"next_round_id,omitempty"`
	NextRoundName string             `json:"next_round_name,omitempty"`
	Summary       RoundSummary       `json:"summary"`
	Matches       []RoundResultMatch `json:"matches"`
	Standings     []Standing         `json:"standings,omitempty"`
}
