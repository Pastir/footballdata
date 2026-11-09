package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// RoundResultMatch содержит полные данные одного сыгранного матча тура.
type RoundResultMatch struct {
	LeagueID   int
	FixtureID  int64
	LeagueName string
	SeasonID   int
	SeasonName string
	RoundID    int
	RoundName  string

	FixtureStart time.Time
	VenueID      *int
	VenueName    *string

	CountryID   int
	CountryName *string

	TeamHomeID     int64
	TeamHomeName   string
	TeamHomeLogo   string
	TeamHomeWinner *bool
	TeamHomeResult *string

	TeamAwayID     int64
	TeamAwayName   string
	TeamAwayLogo   string
	TeamAwayWinner *bool
	TeamAwayResult *string

	StateShort  *string
	RefereeName string

	Scores MatchScores

	// Events — хронология матча (голы, карточки, замены, VAR)
	Events []MatchEvent

	// Stats — сравнительная статистика команд по метрикам
	Stats map[string]MatchStatMetric

	// Lineups — составы команд ("home" и "away")
	Lineups map[string][]MatchLineupPlayer

	// Coaches — главные тренеры ("home" и "away")
	Coaches MatchCoaches

	// Odds — коэффициенты букмекеров 1X2
	Odds MatchOdds

	// Expected Goals (xG) сыгранного матча из provider_expected_fixtures (type_id 5304)
	HomeXG *float64 `json:"home_xg,omitempty"`
	AwayXG *float64 `json:"away_xg,omitempty"`

	// H2H — история очных встреч и сводка
	H2H *H2HData

	// PreMatchStats — предматчевый статистический профиль обеих команд (xG, xGA, голы)
	PreMatchStats *PreMatchStatsComparison

	IsRival bool
}


// MatchScores описывает счёт матча по таймам и итоговый.
type MatchScores struct {
	HomeCurrent *int
	AwayCurrent *int
	HomeHT      *int
	AwayHT      *int
}

// FullScoreString возвращает итоговый счёт матча, например "0:3".
func (s MatchScores) FullScoreString() string {
	if s.HomeCurrent == nil || s.AwayCurrent == nil {
		return "-:-"
	}
	return fmt.Sprintf("%d:%d", *s.HomeCurrent, *s.AwayCurrent)
}

// HTScoreString возвращает счёт первого тайма, например "(0:0)".
func (s MatchScores) HTScoreString() string {
	if s.HomeHT == nil || s.AwayHT == nil {
		return ""
	}
	return fmt.Sprintf("(%d:%d)", *s.HomeHT, *s.AwayHT)
}

// XGString возвращает строку xG матча, например "2.10 – 0.19", либо пустую строку если xG отсутствует.
func (m RoundResultMatch) XGString() string {
	if m.HomeXG == nil || m.AwayXG == nil {
		return ""
	}
	return fmt.Sprintf("%.2f – %.2f", *m.HomeXG, *m.AwayXG)
}

// MatchEvent описывает событие матча (гол, карточка, замена и др.).
type MatchEvent struct {
	Minute        int    `json:"minute"`
	ExtraMinute   *int   `json:"extra_minute,omitempty"`
	TeamID        int64  `json:"team_id"`
	Type          string `json:"type"`      // например "goal", "yellowcard", "redcard", "substitution", "var"
	TypeName      string `json:"type_name"` // человекочитаемое название
	Player        string `json:"player"`
	RelatedPlayer string `json:"related_player,omitempty"`
	Result        string `json:"result,omitempty"`   // счёт после гола, например "1-0"
	Addition      string `json:"addition,omitempty"` // например "Penalty", "Own Goal"
}

// MatchStatMetric описывает сравнительный показатель статистики.
type MatchStatMetric struct {
	TypeID        int      `json:"type_id"`
	Name          string   `json:"name"` // "Fouls", "Corners", "Shots Total"
	Code          string   `json:"code"` // "fouls", "corners", "shots-total"
	DeveloperName string   `json:"developer_name"`
	Color         string   `json:"color,omitempty"`
	Home          *float64 `json:"home,omitempty"`
	Away          *float64 `json:"away,omitempty"`
}

// MatchLineupPlayer описывает игрока в протоколе матча.
type MatchLineupPlayer struct {
	PlayerID          int64  `json:"player_id"`
	Name              string `json:"name"`
	ImagePath         string `json:"image_path,omitempty"`
	Number            int    `json:"number"`
	PositionID        int    `json:"position_id"`
	FormationField    string `json:"formation_field,omitempty"` // координаты "1:1", "2:4" и т.д.
	FormationPosition *int   `json:"formation_position,omitempty"`
	IsStarter         bool   `json:"is_starter"`
}

// MatchCoachItem описывает тренера команды.
type MatchCoachItem struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ImagePath string `json:"image_path,omitempty"`
}

// MatchCoaches содержит тренеров обеих команд.
type MatchCoaches struct {
	Home *MatchCoachItem `json:"home,omitempty"`
	Away *MatchCoachItem `json:"away,omitempty"`
}

// MatchOdds описывает коэффициенты 1X2.
type MatchOdds struct {
	HomeOdds      string
	DrawOdds      string
	AwayOdds      string
	BookmakerName string
}

// RoundSummary содержит детерминированную аналитику и экспресс-статистику тура.
type RoundSummary struct {
	TotalMatches     int
	FinishedMatches  int
	TotalGoals       int
	AvgGoals         float64
	HomeWins         int
	Draws            int
	AwayWins         int
	BTTSCount        int // Обе забьют
	Over25Count      int // Тотал больше 2.5
	FavoritesWon     int
	FavoritesFailed  int
	TopScoringMatch  string
	TopScoringGoals  int
	TopUpsetMatch    string
	TopUpsetOdds     float64
	TotalYellowCards int
	TotalRedCards    int
	LateDramaEvent   string // гол на 90+ минуте
}

// GetRoundResults загружает все сыгранные матчи тура со всеми событиями,
// статистикой, составами и тренерами в рамках единого оптимизированного запроса.
func GetRoundResults(ctx context.Context, pool *pgxpool.Pool, info *LeagueSeasonInfo) ([]RoundResultMatch, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	targetRoundID := 0
	if info != nil && info.CurrRoundID > 0 {
		targetRoundID = info.CurrRoundID
	} else if info != nil && info.SeasonID > 0 {
		_ = pool.QueryRow(ctx, `SELECT last_round_id FROM get_neighbor_rounds($1)`, info.SeasonID).Scan(&targetRoundID)
	}
	if targetRoundID <= 0 {
		return nil, nil
	}

	locale := "ru"
	if info != nil && info.Locale != "" {
		locale = info.Locale
	}

	matches, err := queryFixturesDetailed(ctx, pool, "WHERE pf.round_id = $1 ORDER BY pf.starting_at ASC;", targetRoundID, locale)
	if err != nil {
		return nil, fmt.Errorf("GetRoundResults query: %w", err)
	}
	return matches, nil
}

// parseMatchScores извлекает счета из JSONB структуры:
func parseMatchScores(raw []byte) MatchScores {
	var s MatchScores
	if len(raw) == 0 {
		return s
	}

	var data map[string]map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return s
	}

	getInt := func(m map[string]interface{}, key string) *int {
		if v, ok := m[key]; ok {
			switch val := v.(type) {
			case float64:
				i := int(val)
				return &i
			case int:
				return &val
			case string:
				if i, err := strconv.Atoi(val); err == nil {
					return &i
				}
			}
		}
		return nil
	}

	if home, ok := data["home"]; ok {
		s.HomeCurrent = getInt(home, "CURRENT")
		s.HomeHT = getInt(home, "1ST_HALF")
	}
	if away, ok := data["away"]; ok {
		s.AwayCurrent = getInt(away, "CURRENT")
		s.AwayHT = getInt(away, "1ST_HALF")
	}

	return s
}

// parseMatchOdds извлекает коэффициенты 1X2 из JSONB:
func parseMatchOdds(raw []byte, bookmakerName string) MatchOdds {
	odds := MatchOdds{BookmakerName: bookmakerName}
	if len(raw) == 0 {
		return odds
	}

	var data map[string]struct {
		Value         string `json:"value"`
		OriginalLabel string `json:"original_label"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return odds
	}

	for k, v := range data {
		switch strings.ToUpper(k) {
		case "1", "HOME":
			odds.HomeOdds = v.Value
		case "X", "DRAW":
			odds.DrawOdds = v.Value
		case "2", "AWAY":
			odds.AwayOdds = v.Value
		}
	}

	return odds
}

// ComputeRoundSummary выполняет детерминированный расчет всей аналитики и статистики тура.
func ComputeRoundSummary(matches []RoundResultMatch) RoundSummary {
	var summary RoundSummary
	summary.TotalMatches = len(matches)

	maxGoals := -1
	maxUpsetOdds := 1.0

	for _, m := range matches {
		if m.Scores.HomeCurrent == nil || m.Scores.AwayCurrent == nil {
			continue
		}

		summary.FinishedMatches++
		hg := *m.Scores.HomeCurrent
		ag := *m.Scores.AwayCurrent
		matchGoals := hg + ag
		summary.TotalGoals += matchGoals

		// Исход матча (1-X-2)
		var matchOutcome string
		if hg > ag {
			summary.HomeWins++
			matchOutcome = "1"
		} else if hg < ag {
			summary.AwayWins++
			matchOutcome = "2"
		} else {
			summary.Draws++
			matchOutcome = "X"
		}

		// Обе забьют и Тотал больше 2.5
		if hg > 0 && ag > 0 {
			summary.BTTSCount++
		}
		if matchGoals >= 3 {
			summary.Over25Count++
		}

		// Самый результативный матч
		if matchGoals > maxGoals {
			maxGoals = matchGoals
			summary.TopScoringGoals = matchGoals
			summary.TopScoringMatch = fmt.Sprintf("%s — %s (%d голов)", m.TeamHomeName, m.TeamAwayName, matchGoals)
		}

		// Анализ букмекеров (фавориты / сенсации)
		hOdds, errH := strconv.ParseFloat(m.Odds.HomeOdds, 64)
		aOdds, errA := strconv.ParseFloat(m.Odds.AwayOdds, 64)
		dOdds, errD := strconv.ParseFloat(m.Odds.DrawOdds, 64)

		if errH == nil && errA == nil && errD == nil && hOdds > 0 && aOdds > 0 {
			var favOutcome string
			if hOdds < aOdds && hOdds < dOdds {
				favOutcome = "1"
			} else if aOdds < hOdds && aOdds < dOdds {
				favOutcome = "2"
			}

			if favOutcome != "" {
				if matchOutcome == favOutcome {
					summary.FavoritesWon++
				} else {
					summary.FavoritesFailed++
					// Проверяем, насколько крупный апсет
					var winningOdds float64
					var outcomeDesc string
					if matchOutcome == "1" {
						winningOdds = hOdds
						outcomeDesc = fmt.Sprintf("победа хозяев за кэф %.2f", winningOdds)
					} else if matchOutcome == "2" {
						winningOdds = aOdds
						outcomeDesc = fmt.Sprintf("победа гостей за кэф %.2f", winningOdds)
					} else {
						winningOdds = dOdds
						outcomeDesc = fmt.Sprintf("ничья за кэф %.2f", winningOdds)
					}

					if winningOdds > maxUpsetOdds {
						maxUpsetOdds = winningOdds
						summary.TopUpsetOdds = winningOdds
						summary.TopUpsetMatch = fmt.Sprintf("%s — %s (%s)", m.TeamHomeName, m.TeamAwayName, outcomeDesc)
					}
				}
			}
		}

		// Дисциплина и события на последних минутах
		for _, ev := range m.Events {
			evType := strings.ToUpper(ev.Type)
			if strings.Contains(evType, "YELLOW") {
				summary.TotalYellowCards++
			} else if strings.Contains(evType, "RED") {
				summary.TotalRedCards++
			}

			// Драма на 90+ минуте (решающий гол)
			if strings.Contains(evType, "GOAL") && (ev.Minute >= 90 || (ev.ExtraMinute != nil && *ev.ExtraMinute > 0)) {
				if summary.LateDramaEvent == "" {
					summary.LateDramaEvent = fmt.Sprintf("Гол %s на %d' мин в матче %s — %s (%s)",
						ev.Player, ev.Minute, m.TeamHomeName, m.TeamAwayName, m.Scores.FullScoreString())
				}
			}
		}
	}

	if summary.FinishedMatches > 0 {
		summary.AvgGoals = math.Round(float64(summary.TotalGoals)/float64(summary.FinishedMatches)*10) / 10
	}

	return summary
}
