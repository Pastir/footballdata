package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// NextRoundMatch содержит полные данные одного матча предстоящего тура.
type NextRoundMatch struct {
	LeagueID   int
	FixtureID  int64
	LeagueName string
	SeasonID   int
	SeasonName string
	RoundID    int
	RoundName  string

	FixtureStart time.Time
	RoundStart   time.Time
	RoundEnd     time.Time

	VenueID   *int
	VenueName *string

	CountryID   int
	CountryName string

	TeamHomeID         int64
	TeamHomeName       string
	TeamHomeLogo       string
	FormHome           []string
	PositionHome       *int
	RulePositionHome   *string
	PointsHome         *int
	OverallMatchesHome *int

	TeamAwayID       int64
	TeamAwayName     string
	TeamAwayLogo     string
	FormAway         []string
	PositionAway     *int
	RulePositionAway *string
	PointsAway       *int
	MatchesAway      *int

	StateShort    *string
	OddsHome      string
	OddsDraw      string
	OddsAway      string
	BookmakerName string
	IsRival       bool
}

// GetNextRoundFixtures выбирает из БД все матчи следующего тура лиги
// с формой, позициями и коэффициентами.
func GetNextRoundFixtures(ctx context.Context, pool *pgxpool.Pool, info *LeagueSeasonInfo) ([]NextRoundMatch, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	query := `WITH next_round AS (
     SELECT *
     FROM get_neighbor_rounds($1)
), form AS (
    SELECT participant_id,
           nr.league_id,
           form,
           rounds,
           nr.last_round_id AS last_round_id,
           nr.prev_round_id AS prev_round_id
    FROM next_round AS nr
        CROSS JOIN LATERAL get_form_5last_rounds(nr.season_id, nr.next_round_starting_at)
), position AS (
    SELECT form.participant_id
           , form.form
           , form.last_round_id
           , psr.position
           , psr.standing_rule_id
           , pty.name
           , psr.points
           , psd.value AS overall_matches
    FROM provider_standings AS psr
    LEFT JOIN form ON (psr.round_id = form.last_round_id AND psr.participant_id = form.participant_id)
    LEFT JOIN provider_standings_position_rules AS psru ON (psru.id = psr.standing_rule_id)
    LEFT JOIN provider_types AS pty ON (psru.type_id = pty.id)
    LEFT JOIN provider_standings_details AS psd ON (psr.id = psd.standing_id AND psd.type_id = 129)
    WHERE psr.round_id = form.last_round_id AND psr.participant_id = form.participant_id
)  SELECT pl.id AS league_id
          , pf.id AS fixture_id
          , pl.name AS league_name
          , ps.id AS season_id
          , ps.name AS season_name
          , pf.round_id
          , round.name AS round_name
          , pf.starting_at AS fixture_start
          , round.starting_at AS round_start
          , round.ending_at AS round_end
          , pf.venue_id
          , pv.name AS venue_name
          , pl.country_id
          , pc.name AS country_name
          , fh.participant_id AS team_home_id
          , COALESCE(
                NULLIF(pth.locale->$2->>'short', ''),
                NULLIF(pth.locale->$2->>'long', ''),
                pth.name
            ) AS team_home_name
          , COALESCE(pth.image_path, '') AS team_home_logo
          , formh.form AS form_home
          , posh.position AS position_home
          , posh.name AS rule_position_home
          , posh.points AS points_home
          , posh.overall_matches AS overall_matches_home
          , fa.participant_id AS team_away_id
          , COALESCE(
                NULLIF(pta.locale->$2->>'short', ''),
                NULLIF(pta.locale->$2->>'long', ''),
                pta.name
            ) AS team_away_name
          , COALESCE(pta.image_path, '') AS team_away_logo
          , forma.form AS form_away
          , posa.position AS position_away
          , posa.name AS rule_position_away
          , posa.points AS points_away
          , posa.overall_matches AS overall_matches_away
          , pst.short_name AS state_short
          , odds.odds_data
          , odds.bookmaker_name
          , CASE WHEN rivals.fixture_id NOTNULL THEN TRUE ELSE FALSE END AS rivals
    FROM provider_fixtures AS pf
    LEFT JOIN provider_leagues AS pl ON (pf.league_id = pl.id)
    LEFT JOIN provider_states AS pst ON (pst.id = pf.state_id)
    LEFT JOIN provider_fixtures_participants AS fh ON (fh.fixture_id = pf.id AND fh.location = 'home')
    LEFT JOIN provider_teams AS pth ON (pth.id = fh.participant_id)
    LEFT JOIN provider_fixtures_participants AS fa ON (fa.fixture_id = pf.id AND fa.location = 'away')
    LEFT JOIN provider_teams AS pta ON (pta.id = fa.participant_id)
    LEFT JOIN provider_seasons AS ps ON (pf.season_id = ps.id)
    LEFT JOIN provider_venues AS pv ON (pv.id = pf.venue_id)
    LEFT JOIN provider_countries AS pc ON (pc.id = pl.country_id)
    LEFT JOIN form AS forma ON (fa.participant_id = forma.participant_id)
    LEFT JOIN form AS formh ON (fh.participant_id = formh.participant_id)
    LEFT JOIN position AS posa ON (posa.participant_id = pta.id)
    LEFT JOIN position AS posh ON (posh.participant_id = pth.id)
    LEFT JOIN LATERAL (
        SELECT id
               , name
               , starting_at
               , ending_at
        FROM provider_rounds
        WHERE season_id = ps.id
          AND DATE(NOW()) < starting_at
          AND finished = FALSE
        ORDER BY name::INT
        LIMIT 1
    ) AS round ON TRUE
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
        LEFT JOIN provider_markets AS pm ON(po.market_id = pm.id)
        LEFT JOIN provider_bookmakers AS pb ON(po.bookmaker_id = pb.id)
        WHERE po.market_id = 1 
          AND po.bookmaker_id = COALESCE(
              (SELECT po_sub.bookmaker_id FROM provider_odds po_sub WHERE po_sub.fixture_id = pf.id AND po_sub.market_id = 1 ORDER BY (po_sub.bookmaker_id = 2) DESC, po_sub.bookmaker_id ASC LIMIT 1),
              2
          )
          AND po.fixture_id = pf.id
        GROUP BY po.market_id,
                 po.bookmaker_id,
                 pm.name,
                 pb.name
    ) AS odds ON TRUE
    LEFT JOIN LATERAL (
        SELECT DISTINCT pfl.id AS fixture_id
        FROM provider_fixtures AS pfl
             JOIN provider_fixtures_participants AS pfpl ON pfl.id = pfpl.fixture_id
             JOIN provider_rivals AS prl ON pfpl.participant_id = prl.team_id
        WHERE pfl.round_id = pf.round_id
        GROUP BY pfl.id, prl.rival_id
        HAVING COUNT(DISTINCT pfpl.participant_id) > 1
    ) AS rivals ON (rivals.fixture_id = pf.id)
WHERE pf.league_id = forma.league_id AND ps.is_current AND pf.round_id = round.id
ORDER BY pf.starting_at ASC`

	locale := "ru"
	if info != nil && info.Locale != "" {
		locale = info.Locale
	}
	rows, err := pool.Query(ctx, query, info.SeasonID, locale)
	if err != nil {
		return nil, fmt.Errorf("GetNextRoundFixtures: %w", err)
	}
	defer rows.Close()

	var matches []NextRoundMatch
	for rows.Next() {
		var m NextRoundMatch
		var rawOdds []byte
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
			&m.RoundStart,
			&m.RoundEnd,
			&m.VenueID,
			&m.VenueName,
			&m.CountryID,
			&m.CountryName,
			&m.TeamHomeID,
			&m.TeamHomeName,
			&m.TeamHomeLogo,
			&m.FormHome,
			&m.PositionHome,
			&m.RulePositionHome,
			&m.PointsHome,
			&m.OverallMatchesHome,
			&m.TeamAwayID,
			&m.TeamAwayName,
			&m.TeamAwayLogo,
			&m.FormAway,
			&m.PositionAway,
			&m.RulePositionAway,
			&m.PointsAway,
			&m.MatchesAway,
			&m.StateShort,
			&rawOdds,
			&bookmakerName,
			&m.IsRival,
		); err != nil {
			return nil, fmt.Errorf("GetNextRoundFixtures scan: %w", err)
		}

		if bookmakerName != nil && *bookmakerName != "" {
			m.BookmakerName = *bookmakerName
		}

		if len(rawOdds) > 0 && string(rawOdds) != "null" {
			type oddItem struct {
				Value         string `json:"value"`
				OriginalLabel string `json:"original_label"`
			}
			var oddsMap map[string]oddItem
			if err := json.Unmarshal(rawOdds, &oddsMap); err == nil {
				for label, item := range oddsMap {
					lbl := strings.ToLower(label)
					orig := strings.ToLower(item.OriginalLabel)
					if lbl == "home" || lbl == "1" || orig == "1" {
						m.OddsHome = item.Value
					} else if lbl == "draw" || lbl == "x" || orig == "x" {
						m.OddsDraw = item.Value
					} else if lbl == "away" || lbl == "2" || orig == "2" {
						m.OddsAway = item.Value
					}
				}
			}
		}

		matches = append(matches, m)
	}

	return matches, rows.Err()
}
