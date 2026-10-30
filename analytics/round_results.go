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

	query := `WITH next_round AS (
     SELECT *
     FROM get_neighbor_rounds($1)
) 
SELECT pl.id AS league_id
     , pf.id AS fixture_id
     , pl.name AS league_name
     , ps.id AS season_id
     , ps.name AS season_name
     , pf.round_id
     , next_round.last_round_name AS round_name
     , pf.starting_at AS fixture_start
     , pf.venue_id
     , pv.name AS venue_name
     , pl.country_id
     , pc.name AS country_name
     
     , fh.participant_id AS team_home_id
     , COALESCE(
           NULLIF(pth.locale->$3->>'short', ''),
           NULLIF(pth.locale->$3->>'long', ''),
           pth.name
       ) AS team_home_name
     , COALESCE(pth.image_path, '') AS team_home_logo
     , fh.winner AS team_home_winner
     , fh.result AS team_home_result
     
     , fa.participant_id AS team_away_id
     , COALESCE(
           NULLIF(pta.locale->$3->>'short', ''),
           NULLIF(pta.locale->$3->>'long', ''),
           pta.name
       ) AS team_away_name
     , COALESCE(pta.image_path, '') AS team_away_logo
     , fa.winner AS team_away_winner
     , fa.result AS team_away_result
     
     , pst.short_name AS state_short
     , ref.referee_name
     
     , scores.scores
     , events.events
     , stats.match_stats
     , lineups.lineups
     , coaches.coaches
     , odds.odds_data
     , odds.bookmaker_name
     
     , xgh.value AS home_xg
     , xga.value AS away_xg
     
     , CASE WHEN EXISTS (
           SELECT 1 
           FROM provider_rivals pr 
           WHERE pr.team_id = fh.participant_id AND pr.rival_id = fa.participant_id
       ) THEN TRUE ELSE FALSE END AS rivals

FROM provider_fixtures AS pf 
JOIN next_round ON (pf.season_id = next_round.season_id AND pf.round_id = (CASE WHEN $2::int > 0 THEN $2::int ELSE next_round.last_round_id END))
JOIN provider_seasons AS ps ON (pf.season_id = ps.id AND ps.is_current)
LEFT JOIN provider_leagues AS pl ON (pf.league_id = pl.id)
LEFT JOIN provider_states AS pst ON (pst.id = pf.state_id)
LEFT JOIN provider_fixtures_participants AS fh ON (fh.fixture_id = pf.id AND fh.location = 'home')
LEFT JOIN provider_teams AS pth ON (pth.id = fh.participant_id)
LEFT JOIN provider_fixtures_participants AS fa ON (fa.fixture_id = pf.id AND fa.location = 'away')
LEFT JOIN provider_teams AS pta ON (pta.id = fa.participant_id)
LEFT JOIN provider_expected_fixtures AS xgh ON (xgh.fixture_id = pf.id AND xgh.participant_id = fh.participant_id AND xgh.type_id = 5304)
LEFT JOIN provider_expected_fixtures AS xga ON (xga.fixture_id = pf.id AND xga.participant_id = fa.participant_id AND xga.type_id = 5304)
LEFT JOIN provider_venues AS pv ON (pv.id = pf.venue_id)
LEFT JOIN provider_countries AS pc ON (pc.id = pl.country_id)

LEFT JOIN LATERAL (
    SELECT COALESCE(pr.display_name, pr.common_name, pr.name, '') AS referee_name
    FROM provider_fixtures_referees pfr
    JOIN provider_referees pr ON (pfr.referee_id = pr.id)
    WHERE pfr.fixture_id = pf.id
    LIMIT 1
) AS ref ON TRUE

LEFT JOIN LATERAL (
    SELECT po.market_id
         , po.bookmaker_id
         , pm.name AS market_name
         , jsonb_object_agg(
               po.label,
               jsonb_build_object(
                   'value', po.value,
                   'original_label', CASE WHEN po.original_label = 'Draw' THEN 'X' ELSE po.original_label END
               )
           ) AS odds_data
         , pb.name AS bookmaker_name
    FROM provider_odds AS po
    LEFT JOIN provider_markets AS pm ON (po.market_id = pm.id)
    LEFT JOIN provider_bookmakers AS pb ON (po.bookmaker_id = pb.id)
    WHERE po.market_id = 1 
      AND po.bookmaker_id = COALESCE(
          (SELECT po_sub.bookmaker_id FROM provider_odds po_sub WHERE po_sub.fixture_id = pf.id AND po_sub.market_id = 1 ORDER BY (po_sub.bookmaker_id = 2) DESC, po_sub.bookmaker_id ASC LIMIT 1),
          2
      )
      AND po.fixture_id = pf.id
    GROUP BY po.market_id, po.bookmaker_id, pm.name, pb.name
) AS odds ON TRUE

LEFT JOIN LATERAL (
    SELECT jsonb_object_agg(t.participant, t.obj) AS scores
    FROM (
        SELECT pfs.participant,
               jsonb_build_object('participant_id', pfs.participant_id)
               || jsonb_object_agg(pfs.description, pfs.goals) AS obj
        FROM provider_fixtures_scores AS pfs
        WHERE pfs.fixture_id = pf.id
          AND pfs.type_id IN (1525, 1)
        GROUP BY pfs.participant, pfs.participant_id
    ) AS t
) AS scores ON TRUE

LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'minute', pe.minute,
                'extra_minute', pe.extra_minute,
                'team_id', pe.participant_id,
                'type', pty.developer_name,
                'type_name', pty.name,
                'player', COALESCE(pp.display_name, pp.common_name, pp.name, ''),
                'related_player', COALESCE(ppr.display_name, ppr.common_name, ppr.name, ''),
                'result', COALESCE(pe.result, ''),
                'addition', COALESCE(pe.addition, '')
            ) ORDER BY pe.minute ASC, pe.extra_minute ASC NULLS FIRST, pe.id ASC
        ),
        '[]'::jsonb
    ) AS events
    FROM provider_events pe
    LEFT JOIN provider_types pty ON (pe.type_id = pty.id)
    LEFT JOIN provider_players pp ON (pe.player_id = pp.id)
    LEFT JOIN provider_players ppr ON (pe.related_player_id = ppr.id)
    WHERE pe.fixture_id = pf.id
) AS events ON TRUE

LEFT JOIN LATERAL (
    SELECT jsonb_object_agg(
        s.key,
        jsonb_build_object(
            'type_id', s.type_id,
            'name', s.name,
            'code', s.code,
            'developer_name', s.developer_name,
            'color', s.color,
            'home', s.home,
            'away', s.away
        )
    ) AS match_stats
    FROM (
        SELECT COALESCE(pt.developer_name, pt.code, pt.id::text) AS key,
               pt.id AS type_id,
               pt.name,
               pt.code,
               pt.developer_name,
               COALESCE(pt.color, '') AS color,
               MAX(CASE WHEN pfs.location = 'home' THEN pfs.value ELSE NULL END) AS home,
               MAX(CASE WHEN pfs.location = 'away' THEN pfs.value ELSE NULL END) AS away
        FROM provider_fixtures_statistics pfs
        LEFT JOIN provider_types pt ON (pfs.type_id = pt.id)
        WHERE pfs.fixture_id = pf.id AND pfs.league_id = pf.league_id
        GROUP BY pt.id, pt.name, pt.code, pt.developer_name, pt.color
    ) s
) AS stats ON TRUE

LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_object_agg(
            l.location,
            l.players
        ),
        '{}'::jsonb
    ) AS lineups
    FROM (
        SELECT CASE WHEN plu.team_id = fh.participant_id THEN 'home' ELSE 'away' END AS location,
               jsonb_agg(
                   jsonb_build_object(
                       'player_id', plu.player_id,
                       'name', COALESCE(ppl.display_name, ppl.common_name, ppl.name, ''),
                       'image_path', COALESCE(ppl.image_path, ''),
                       'number', plu.jersey_number,
                       'position_id', plu.position_id,
                       'formation_field', plu.formation_field,
                       'formation_position', plu.formation_position,
                       'is_starter', (plu.type_id = 11)
                   ) ORDER BY plu.type_id ASC, plu.formation_position ASC NULLS LAST
               ) AS players
        FROM (
            SELECT DISTINCT ON (team_id, player_id) *
            FROM provider_lineups
            WHERE fixture_id = pf.id AND league_id = pf.league_id
            ORDER BY team_id, player_id, id DESC
        ) plu
        LEFT JOIN provider_players ppl ON (plu.player_id = ppl.id)
        GROUP BY plu.team_id
    ) l
) AS lineups ON TRUE

LEFT JOIN LATERAL (
    SELECT jsonb_build_object(
        'home', (
            SELECT jsonb_build_object(
                'id', pch.id,
                'name', COALESCE(pch.display_name, pch.common_name, pch.name, ''),
                'image_path', COALESCE(pch.image_path, '')
            )
            FROM provider_fixtures_coaches pfch
            JOIN provider_coachs pch ON (pfch.coaches_id = pch.id)
            WHERE pfch.fixture_id = pf.id AND pfch.participant_id = fh.participant_id
            LIMIT 1
        ),
        'away', (
            SELECT jsonb_build_object(
                'id', pca.id,
                'name', COALESCE(pca.display_name, pca.common_name, pca.name, ''),
                'image_path', COALESCE(pca.image_path, '')
            )
            FROM provider_fixtures_coaches pfca
            JOIN provider_coachs pca ON (pfca.coaches_id = pca.id)
            WHERE pfca.fixture_id = pf.id AND pfca.participant_id = fa.participant_id
            LIMIT 1
        )
    ) AS coaches
) AS coaches ON TRUE

ORDER BY pf.starting_at ASC;`

	currRoundOverride := 0
	if info.CurrRoundID > 0 {
		currRoundOverride = info.CurrRoundID
	}
	locale := "ru"
	if info != nil && info.Locale != "" {
		locale = info.Locale
	}

	rows, err := pool.Query(ctx, query, info.SeasonID, currRoundOverride, locale)
	if err != nil {
		return nil, fmt.Errorf("GetRoundResults query: %w", err)
	}
	defer rows.Close()

	var matches []RoundResultMatch

	for rows.Next() {
		var m RoundResultMatch
		var scoresRaw, eventsRaw, statsRaw, lineupsRaw, coachesRaw, oddsRaw []byte
		var bookmakerName *string

		if err := rows.Scan(
			&m.LeagueID,
			&m.FixtureID,
			&m.LeagueName,
			&m.SeasonID,
			&m.SeasonName,
			&m.RoundID,
			&m.RoundName,
			&m.FixtureStart,
			&m.VenueID,
			&m.VenueName,
			&m.CountryID,
			&m.CountryName,

			&m.TeamHomeID,
			&m.TeamHomeName,
			&m.TeamHomeLogo,
			&m.TeamHomeWinner,
			&m.TeamHomeResult,

			&m.TeamAwayID,
			&m.TeamAwayName,
			&m.TeamAwayLogo,
			&m.TeamAwayWinner,
			&m.TeamAwayResult,

			&m.StateShort,
			&m.RefereeName,

			&scoresRaw,
			&eventsRaw,
			&statsRaw,
			&lineupsRaw,
			&coachesRaw,
			&oddsRaw,
			&bookmakerName,
			&m.HomeXG,
			&m.AwayXG,
			&m.IsRival,
		); err != nil {
			return nil, fmt.Errorf("GetRoundResults scan: %w", err)
		}

		if bookmakerName != nil {
			m.Odds.BookmakerName = *bookmakerName
		}

		// Парсинг Scores
		m.Scores = parseMatchScores(scoresRaw)

		// Парсинг Events
		if len(eventsRaw) > 0 {
			_ = json.Unmarshal(eventsRaw, &m.Events)
		}

		// Парсинг Stats
		if len(statsRaw) > 0 {
			_ = json.Unmarshal(statsRaw, &m.Stats)
		}

		// Парсинг Lineups
		if len(lineupsRaw) > 0 {
			_ = json.Unmarshal(lineupsRaw, &m.Lineups)
		}

		// Парсинг Coaches
		if len(coachesRaw) > 0 {
			_ = json.Unmarshal(coachesRaw, &m.Coaches)
		}

		// Парсинг Odds
		m.Odds = parseMatchOdds(oddsRaw, m.Odds.BookmakerName)

		matches = append(matches, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetRoundResults rows: %w", err)
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
