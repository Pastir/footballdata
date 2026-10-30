package prediction

import (
	"time"
)

// BetType определяет тип ставки (ординар или экспресс).
type BetType string

const (
	BetTypeSingle      BetType = "single"
	BetTypeAccumulator BetType = "accumulator"
)

// BetStatus определяет статус расчета прогноза.
type BetStatus string

const (
	StatusPending   BetStatus = "pending"
	StatusWon       BetStatus = "won"
	StatusLost      BetStatus = "lost"
	StatusRefund    BetStatus = "refund"
	StatusHalfWon   BetStatus = "half_won"
	StatusHalfLost  BetStatus = "half_lost"
	StatusCancelled BetStatus = "cancelled"
)

// MarketKey определяет ключ рынка для автоматического расчёта.
type MarketKey string

const (
	MarketKey1X2          MarketKey = "1x2"
	MarketKeyDoubleChance MarketKey = "double_chance"
	MarketKeyOverUnder    MarketKey = "over_under"
	MarketKeyBTTS         MarketKey = "btts"
	MarketKeyHandicap     MarketKey = "handicap"
)

// Prediction представляет карточку прогноза (ординар или экспресс).
type Prediction struct {
	ID               int64            `json:"id"`
	BetType          BetType          `json:"bet_type"`
	Title            string           `json:"title"`
	AnalysisText     string           `json:"analysis_text"`
	TotalOdds        float64          `json:"total_odds"`
	Confidence       float64          `json:"confidence"`
	ValueEdge        float64          `json:"value_edge"`
	RecommendedStake float64          `json:"recommended_stake"` // в юнитах, default 1.0u
	Status           BetStatus        `json:"status"`
	ProfitUnits      float64          `json:"profit_units"`
	SettledAt        *time.Time       `json:"settled_at,omitempty"`
	PublicationID    *int64           `json:"publication_id,omitempty"`
	Items            []PredictionItem `json:"items,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// PredictionItem представляет один исход внутри ординара или экспресса.
type PredictionItem struct {
	ID                int64     `json:"id"`
	PredictionID      int64     `json:"prediction_id"`
	FixtureID         int       `json:"fixture_id"`
	LeagueID          int       `json:"league_id"`
	HomeTeamID        int       `json:"home_team_id"`
	AwayTeamID        int       `json:"away_team_id"`
	MatchStartingAt   time.Time `json:"match_starting_at"`
	MarketName        string    `json:"market_name"`
	MarketKey         MarketKey `json:"market_key"`
	Selection         string    `json:"selection"`
	Odds              float64   `json:"odds"`
	BookmakerName     string    `json:"bookmaker_name"`
	ActualScoreFT     string    `json:"actual_score_ft,omitempty"`
	ActualScoreHT     string    `json:"actual_score_ht,omitempty"`
	SettlementStatus  BetStatus `json:"settlement_status"`
	SettlementDetails string    `json:"settlement_details,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// PerformanceReport содержит метрики эффективности ставок (ROI, Yield, WinRate).
type PerformanceReport struct {
	TotalBets     int     `json:"total_bets"`
	WonBets       int     `json:"won_bets"`
	LostBets      int     `json:"lost_bets"`
	RefundBets    int     `json:"refund_bets"`
	WinRate       float64 `json:"win_rate"`       // WinRate % = Won / (Won + Lost) * 100
	TotalStaked   float64 `json:"total_staked"`   // Суммарный объём ставок в юнитах
	TotalProfit   float64 `json:"total_profit"`   // Чистая прибыль/убыток в юнитах
	ROI           float64 `json:"roi"`            // ROI % = TotalProfit / TotalStaked * 100
	AvgOdds       float64 `json:"avg_odds"`       // Средний коэффициент сыгравших ставок
	ProfitFactor  float64 `json:"profit_factor"`  // Gross Profit / Gross Loss
}
