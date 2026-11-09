package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/jackc/pgx/v4/pgxpool"
)

// LeagueArchiveRound описывает тур/раунд сезона в архиве.
type LeagueArchiveRound struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	StartingAt    string `json:"starting_at"`
	EndingAt      string `json:"ending_at"`
	IsCurrent     bool   `json:"is_current"`
	FinishedCount int    `json:"finished_count"`
	TotalMatches  int    `json:"total_matches"`
}

// LeagueArchiveSeason описывает сезон турнира с перечнем всех туров.
type LeagueArchiveSeason struct {
	ID        int                  `json:"id"`
	Name      string               `json:"name"`
	IsCurrent bool                 `json:"is_current"`
	Rounds    []LeagueArchiveRound `json:"rounds"`
}

// LeagueArchiveData содержит полный архив сезонов и туров для селектора в Mini App.
type LeagueArchiveData struct {
	LeagueID   int                   `json:"league_id"`
	LeagueName string                `json:"league_name"`
	Seasons    []LeagueArchiveSeason `json:"seasons"`
}

// GetLeagueArchive извлекает структурированный архив сезонов и туров турнира из БД.
func GetLeagueArchive(ctx context.Context, pool *pgxpool.Pool, leagueID int) (*LeagueArchiveData, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	leagueQuery := `SELECT name FROM provider_leagues WHERE id = $1 LIMIT 1`
	var leagueName string
	if err := pool.QueryRow(ctx, leagueQuery, leagueID).Scan(&leagueName); err != nil {
		return nil, fmt.Errorf("failed to get league name for %d: %w", leagueID, err)
	}

	query := `
		SELECT 
			ps.id AS season_id,
			ps.name AS season_name,
			ps.is_current AS season_is_current,
			pr.id AS round_id,
			pr.name AS round_name,
			COALESCE(pr.starting_at::text, '') AS starting_at,
			COALESCE(pr.ending_at::text, '') AS ending_at,
			COALESCE(pr.is_current, false) AS round_is_current,
			COUNT(pf.id) AS total_matches,
			COUNT(CASE WHEN pf.state_id IN (5, 7, 8) THEN 1 END) AS finished_matches
		FROM provider_seasons ps
		JOIN provider_rounds pr ON pr.season_id = ps.id
		LEFT JOIN provider_fixtures pf ON pf.round_id = pr.id
		WHERE ps.league_id = $1
		GROUP BY ps.id, ps.name, ps.is_current, pr.id, pr.name, pr.starting_at, pr.ending_at, pr.is_current
		ORDER BY ps.name DESC, pr.starting_at ASC, pr.id ASC
	`

	rows, err := pool.Query(ctx, query, leagueID)
	if err != nil {
		return nil, fmt.Errorf("archive query failed: %w", err)
	}
	defer rows.Close()

	seasonMap := make(map[int]*LeagueArchiveSeason)
	var seasonsOrder []*LeagueArchiveSeason

	for rows.Next() {
		var (
			seasonID        int
			seasonName      string
			seasonIsCurrent bool
			roundID         int
			roundName       string
			startingAt      string
			endingAt        string
			roundIsCurrent  bool
			totalMatches    int
			finishedMatches int
		)

		if err := rows.Scan(
			&seasonID, &seasonName, &seasonIsCurrent,
			&roundID, &roundName, &startingAt, &endingAt,
			&roundIsCurrent, &totalMatches, &finishedMatches,
		); err != nil {
			return nil, fmt.Errorf("scan archive row: %w", err)
		}

		s, exists := seasonMap[seasonID]
		if !exists {
			s = &LeagueArchiveSeason{
				ID:        seasonID,
				Name:      seasonName,
				IsCurrent: seasonIsCurrent,
				Rounds:    make([]LeagueArchiveRound, 0, 38),
			}
			seasonMap[seasonID] = s
			seasonsOrder = append(seasonsOrder, s)
		}

		s.Rounds = append(s.Rounds, LeagueArchiveRound{
			ID:            roundID,
			Name:          roundName,
			StartingAt:    startingAt,
			EndingAt:      endingAt,
			IsCurrent:     roundIsCurrent,
			FinishedCount: finishedMatches,
			TotalMatches:  totalMatches,
		})
	}

	resSeasons := make([]LeagueArchiveSeason, len(seasonsOrder))
	for i, s := range seasonsOrder {
		resSeasons[i] = *s
	}

	return &LeagueArchiveData{
		LeagueID:   leagueID,
		LeagueName: leagueName,
		Seasons:    resSeasons,
	}, nil
}

// GetRoundResultsByRoundID загружает матчи любого раунда (включая исторические туры) со всей аналитикой.
func GetRoundResultsByRoundID(ctx context.Context, pool *pgxpool.Pool, roundID int, locale string) ([]RoundResultMatch, *LeagueSeasonInfo, error) {
	if pool == nil {
		return nil, nil, fmt.Errorf("database pool is not initialized")
	}

	infoQuery := `
		WITH cur AS (
			SELECT 
				r.id AS round_id,
				r.name AS round_name,
				NULLIF(regexp_replace(r.name, '[^0-9]', '', 'g'), '')::int AS round_num,
				r.starting_at,
				r.season_id,
				s.name AS season_name,
				s.league_id,
				l.name AS league_name
			FROM provider_rounds r
			JOIN provider_seasons s ON s.id = r.season_id
			JOIN provider_leagues l ON l.id = s.league_id
			WHERE r.id = $1
			LIMIT 1
		)
		SELECT 
			c.round_id,
			c.round_name,
			c.season_id,
			c.season_name,
			c.league_id,
			c.league_name,
			COALESCE((
				SELECT pr_prev.id
				FROM provider_rounds pr_prev
				WHERE pr_prev.season_id = c.season_id
				  AND pr_prev.id <> c.round_id
				  AND (
				      (c.round_num IS NOT NULL AND NULLIF(regexp_replace(pr_prev.name, '[^0-9]', '', 'g'), '')::int < c.round_num)
				      OR (c.round_num IS NULL AND pr_prev.starting_at < c.starting_at)
				  )
				ORDER BY NULLIF(regexp_replace(pr_prev.name, '[^0-9]', '', 'g'), '')::int DESC NULLS LAST, pr_prev.starting_at DESC
				LIMIT 1
			), 0) AS prev_round_id,
			COALESCE((
				SELECT pr_next.id
				FROM provider_rounds pr_next
				WHERE pr_next.season_id = c.season_id
				  AND pr_next.id <> c.round_id
				  AND (
				      (c.round_num IS NOT NULL AND NULLIF(regexp_replace(pr_next.name, '[^0-9]', '', 'g'), '')::int > c.round_num)
				      OR (c.round_num IS NULL AND pr_next.starting_at > c.starting_at)
				  )
				ORDER BY NULLIF(regexp_replace(pr_next.name, '[^0-9]', '', 'g'), '')::int ASC NULLS LAST, pr_next.starting_at ASC
				LIMIT 1
			), 0) AS next_round_id
		FROM cur c
	`

	var info LeagueSeasonInfo
	if err := pool.QueryRow(ctx, infoQuery, roundID).Scan(
		&info.CurrRoundID,
		&info.CurrRoundName,
		&info.SeasonID,
		&info.SeasonName,
		&info.LeagueID,
		&info.LeagueName,
		&info.PrevRoundID,
		&info.NextRoundID,
	); err != nil {
		return nil, nil, fmt.Errorf("round metadata not found for round %d: %w", roundID, err)
	}
	info.Locale = locale

	matches, err := queryFixturesDetailed(ctx, pool, "WHERE pf.round_id = $1 ORDER BY pf.starting_at ASC;", roundID, locale)
	if err != nil {
		return nil, nil, fmt.Errorf("GetRoundResultsByRoundID query: %w", err)
	}

	// Обогащаем предматчевой аналитикой и H2H несыгранные матчи тура (с кэшированием профилей команд)
	profileCache := make(map[int64]TeamPreMatchStats)
	for i := range matches {
		m := &matches[i]
		isUnplayed := m.StateShort == nil || *m.StateShort == "NS" || *m.StateShort == "TBD" || *m.StateShort == "POSTP"
		if !isUnplayed || m.TeamHomeID <= 0 || m.TeamAwayID <= 0 {
			continue
		}
		if h2h, hErr := GetH2HMatches(ctx, pool, m.TeamHomeID, m.TeamAwayID, locale, 10); hErr == nil {
			m.H2H = &h2h
		}
		hp, ok := profileCache[m.TeamHomeID]
		if !ok {
			hp, _ = GetTeamPreMatchProfile(ctx, pool, m.TeamHomeID, info.SeasonID)
			profileCache[m.TeamHomeID] = hp
		}
		ap, ok := profileCache[m.TeamAwayID]
		if !ok {
			ap, _ = GetTeamPreMatchProfile(ctx, pool, m.TeamAwayID, info.SeasonID)
			profileCache[m.TeamAwayID] = ap
		}
		m.PreMatchStats = &PreMatchStatsComparison{
			Home: hp,
			Away: ap,
		}
	}

	return matches, &info, nil
}

// GetFixtureLiveDetails возвращает оперативные live-данные конкретного матча.
func GetFixtureLiveDetails(ctx context.Context, pool *pgxpool.Pool, fixtureID int64, locale string) (*RoundResultMatch, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	matches, err := queryFixturesDetailed(ctx, pool, "WHERE pf.id = $1 LIMIT 1;", fixtureID, locale)
	if err != nil {
		return nil, fmt.Errorf("GetFixtureLiveDetails: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("fixture %d not found", fixtureID)
	}
	return &matches[0], nil
}

// GetLiveFixtures возвращает список всех матчей, идущих прямо сейчас или запланированных на сегодня.
func GetLiveFixtures(ctx context.Context, pool *pgxpool.Pool, leagueID int, locale string) ([]RoundResultMatch, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	tail := `
WHERE (pf.state_id IN (2, 3, 4, 6, 9, 21, 22, 23, 25) 
   OR (pf.starting_at::date = CURRENT_DATE AND pf.state_id NOT IN (10, 12, 14, 15)))
  AND ($1 = 0 OR pf.league_id = $1)
ORDER BY pf.starting_at ASC, pf.id ASC
LIMIT 50;`

	matches, err := queryFixturesDetailed(ctx, pool, tail, leagueID, locale)
	if err != nil {
		return nil, fmt.Errorf("GetLiveFixtures: %w", err)
	}
	if matches == nil {
		return []RoundResultMatch{}, nil
	}
	return matches, nil
}

func queryFixturesDetailed(ctx context.Context, pool *pgxpool.Pool, tailClause string, arg any, locale string) ([]RoundResultMatch, error) {
	query := `
SELECT pl.id AS league_id
     , pf.id AS fixture_id
     , pl.name AS league_name
     , ps.id AS season_id
     , ps.name AS season_name
     , pf.round_id
     , COALESCE(pr.name, '') AS round_name
     , pf.starting_at AS fixture_start
     , pf.venue_id
     , pv.name AS venue_name
     , COALESCE(pl.country_id, 0) AS country_id
     , pc.name AS country_name
     
     , COALESCE(fh.participant_id, 0) AS team_home_id
     , COALESCE(
           NULLIF(pth.locale->$2->>'short', ''),
           NULLIF(pth.locale->$2->>'long', ''),
           pth.name,
           'TBD'
       ) AS team_home_name
     , COALESCE(pth.image_path, '') AS team_home_logo
     , fh.winner AS team_home_winner
     , fh.result AS team_home_result
     
     , COALESCE(fa.participant_id, 0) AS team_away_id
     , COALESCE(
           NULLIF(pta.locale->$2->>'short', ''),
           NULLIF(pta.locale->$2->>'long', ''),
           pta.name,
           'TBD'
       ) AS team_away_name
     , COALESCE(pta.image_path, '') AS team_away_logo
     , fa.winner AS team_away_winner
     , fa.result AS team_away_result
     
     , COALESCE(pst.short_name, 'NS') AS state_short
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
LEFT JOIN provider_rounds AS pr ON (pr.id = pf.round_id)
JOIN provider_seasons AS ps ON (pf.season_id = ps.id)
LEFT JOIN provider_leagues AS pl ON (pf.league_id = pl.id)
LEFT JOIN provider_states AS pst ON (pst.id = pf.state_id)
LEFT JOIN provider_fixtures_participants AS fh ON (fh.fixture_id = pf.id AND fh.location = 'home')
LEFT JOIN provider_teams AS pth ON (pth.id = fh.participant_id)
LEFT JOIN provider_fixtures_participants AS fa ON (fa.fixture_id = pf.id AND fa.location = 'away')
LEFT JOIN provider_teams AS pta ON (pta.id = fa.participant_id)
LEFT JOIN provider_venues AS pv ON (pf.venue_id = pv.id)
LEFT JOIN provider_countries AS pc ON (pl.country_id = pc.id)

LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_build_object(
            '1X2',
            jsonb_build_object(
                '1', MAX(CASE WHEN po.market_id = 1 AND po.label = '1' THEN po.value END),
                'X', MAX(CASE WHEN po.market_id = 1 AND po.label = 'X' THEN po.value END),
                '2', MAX(CASE WHEN po.market_id = 1 AND po.label = '2' THEN po.value END)
            )
        ),
        '{}'::jsonb
    ) AS odds_data,
    COALESCE(MAX(pb.name), '') AS bookmaker_name
    FROM (
        SELECT DISTINCT ON (po_sub.market_id, po_sub.label) po_sub.*
        FROM provider_odds po_sub
        WHERE po_sub.fixture_id = pf.id AND po_sub.market_id = 1
        ORDER BY po_sub.market_id, po_sub.label,
                 CASE WHEN po_sub.bookmaker_id = 2 THEN 0 ELSE 1 END,
                 po_sub.created_at DESC
    ) po
    LEFT JOIN provider_bookmakers pb ON (po.bookmaker_id = pb.id)
) AS odds ON TRUE

LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_build_object(
            'home', jsonb_build_object(
                'current', MAX(CASE WHEN score_type.developer_name = 'CURRENT' AND pfs.participant_id = fh.participant_id THEN pfs.goals END),
                '1ST_HALF', MAX(CASE WHEN score_type.developer_name = '1ST_HALF' AND pfs.participant_id = fh.participant_id THEN pfs.goals END),
                '2ND_HALF', MAX(CASE WHEN score_type.developer_name = '2ND_HALF' AND pfs.participant_id = fh.participant_id THEN pfs.goals END)
            ),
            'away', jsonb_build_object(
                'current', MAX(CASE WHEN score_type.developer_name = 'CURRENT' AND pfs.participant_id = fa.participant_id THEN pfs.goals END),
                '1ST_HALF', MAX(CASE WHEN score_type.developer_name = '1ST_HALF' AND pfs.participant_id = fa.participant_id THEN pfs.goals END),
                '2ND_HALF', MAX(CASE WHEN score_type.developer_name = '2ND_HALF' AND pfs.participant_id = fa.participant_id THEN pfs.goals END)
            )
        ),
        '{}'::jsonb
    ) AS scores
    FROM provider_fixtures_scores pfs
    LEFT JOIN provider_types score_type ON (pfs.type_id = score_type.id)
    WHERE pfs.fixture_id = pf.id
) AS scores ON TRUE

LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'minute', pe.minute,
                'extra_minute', pe.extra_minute,
                'team_id', pe.participant_id,
                'type', COALESCE(pt.code, pt.developer_name, pt.id::text),
                'type_name', pt.name,
                'player', COALESCE(pp.display_name, pp.common_name, pp.name, ''),
                'related_player', COALESCE(prp.display_name, prp.common_name, prp.name, ''),
                'result', COALESCE(pe.result, ''),
                'addition', COALESCE(pe.addition, '')
            ) ORDER BY pe.minute ASC, COALESCE(pe.extra_minute, 0) ASC, pe.id ASC
        ),
        '[]'::jsonb
    ) AS events
    FROM provider_events pe
    LEFT JOIN provider_types pt ON (pe.type_id = pt.id)
    LEFT JOIN provider_players pp ON (pe.player_id = pp.id)
    LEFT JOIN provider_players prp ON (pe.related_player_id = prp.id)
    WHERE pe.fixture_id = pf.id
) AS events ON TRUE

LEFT JOIN LATERAL (
    SELECT COALESCE(
        jsonb_object_agg(
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
        ),
        '{}'::jsonb
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
    SELECT COALESCE(
        (SELECT COALESCE(pp.display_name, pp.common_name, pp.name, '') FROM provider_fixtures_referees pfr 
         JOIN provider_referees pp ON (pfr.referee_id = pp.id) 
         WHERE pfr.fixture_id = pf.id LIMIT 1),
        ''
    ) AS referee_name
) AS ref ON TRUE

LEFT JOIN LATERAL (
    SELECT value FROM provider_expected_fixtures 
    WHERE fixture_id = pf.id AND type_id = 5304 AND participant_id = fh.participant_id
    LIMIT 1
) AS xgh ON TRUE

LEFT JOIN LATERAL (
    SELECT value FROM provider_expected_fixtures 
    WHERE fixture_id = pf.id AND type_id = 5304 AND participant_id = fa.participant_id
    LIMIT 1
) AS xga ON TRUE

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

` + tailClause

	rows, err := pool.Query(ctx, query, arg, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []RoundResultMatch
	for rows.Next() {
		var m RoundResultMatch
		var (
			rawScores, rawEvents, rawStats, rawLineups, rawCoaches, rawOdds []byte
			bookmakerName                                                   *string
			homeXG, awayXG                                                   *float64
		)

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

			&rawScores,
			&rawEvents,
			&rawStats,
			&rawLineups,
			&rawCoaches,
			&rawOdds,
			&bookmakerName,

			&homeXG,
			&awayXG,
			&m.IsRival,
		); err != nil {
			return nil, fmt.Errorf("scan match error: %w", err)
		}

		if bookmakerName != nil {
			m.Odds.BookmakerName = *bookmakerName
		}

		if homeXG != nil {
			v := math.Round(*homeXG*100) / 100
			m.HomeXG = &v
		}
		if awayXG != nil {
			v := math.Round(*awayXG*100) / 100
			m.AwayXG = &v
		}

		parseMatchJSONPayload(&m, rawScores, rawEvents, rawStats, rawLineups, rawCoaches, rawOdds)
		matches = append(matches, m)
	}

	return matches, nil
}

// parseMatchJSONPayload разбирает вспомогательные jsonb поля в поля структуры RoundResultMatch.
func parseMatchJSONPayload(m *RoundResultMatch, rawScores, rawEvents, rawStats, rawLineups, rawCoaches, rawOdds []byte) {
	if len(rawScores) > 0 {
		var sc struct {
			Home struct {
				Current *int `json:"current"`
				HT      *int `json:"1ST_HALF"`
				FT      *int `json:"2ND_HALF"`
			} `json:"home"`
			Away struct {
				Current *int `json:"current"`
				HT      *int `json:"1ST_HALF"`
				FT      *int `json:"2ND_HALF"`
			} `json:"away"`
		}
		if err := json.Unmarshal(rawScores, &sc); err == nil {
			m.Scores.HomeCurrent = sc.Home.Current
			m.Scores.AwayCurrent = sc.Away.Current
			m.Scores.HomeHT = sc.Home.HT
			m.Scores.AwayHT = sc.Away.HT
		}
	}

	if len(rawEvents) > 0 {
		_ = json.Unmarshal(rawEvents, &m.Events)
	}

	if len(rawStats) > 0 {
		_ = json.Unmarshal(rawStats, &m.Stats)
	}

	if len(rawLineups) > 0 {
		_ = json.Unmarshal(rawLineups, &m.Lineups)
	}

	if len(rawCoaches) > 0 {
		_ = json.Unmarshal(rawCoaches, &m.Coaches)
	}

	if len(rawOdds) > 0 {
		var od struct {
			Odds1X2 struct {
				One  string `json:"1"`
				Draw string `json:"X"`
				Two  string `json:"2"`
			} `json:"1X2"`
		}
		if err := json.Unmarshal(rawOdds, &od); err == nil {
			m.Odds.HomeOdds = od.Odds1X2.One
			m.Odds.DrawOdds = od.Odds1X2.Draw
			m.Odds.AwayOdds = od.Odds1X2.Two
		}
	}
}
