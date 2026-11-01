package analytics

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

// PlayerLeaderItem представляет одного игрока в рейтинге лидеров сезона.
type PlayerLeaderItem struct {
	Rank         int    `json:"rank"`
	PlayerID     int64  `json:"player_id"`
	PlayerName   string `json:"player_name"`
	PlayerImage  string `json:"player_image"`
	TeamID       int    `json:"team_id"`
	TeamName     string `json:"team_name"`
	TeamCode     string `json:"team_code"`
	TeamLogo     string `json:"team_logo"`
	Matches      int    `json:"matches"`
	PrimaryVal   int    `json:"primary_val"`   // Основной показатель (голы / ассисты / G+A / сухие матчи / карточки)
	SecondaryVal int    `json:"secondary_val"` // Дополнительный показатель (пенальти / ассисты / пропущено)
	MetricLabel  string `json:"metric_label"`  // Подпись показателя (например, "голов", "ассистов")
}

// SeasonLeadersData объединяет топ-10 по всем ключевым категориям турнира.
type SeasonLeadersData struct {
	LeagueID    int                `json:"league_id"`
	LeagueName  string             `json:"league_name"`
	SeasonID    int                `json:"season_id"`
	SeasonName  string             `json:"season_name"`
	Locale      string             `json:"locale"`
	TopScorers  []PlayerLeaderItem `json:"top_scorers"`  // Бомбардиры
	TopAssists  []PlayerLeaderItem `json:"top_assists"`  // Ассистенты
	TopPoints   []PlayerLeaderItem `json:"top_points"`   // Гол + Пас
	CleanSheets []PlayerLeaderItem `json:"clean_sheets"` // «Сухие» матчи вратарей
	Discipline  []PlayerLeaderItem `json:"discipline"`   // Карточки (ЖК + КК)
}

// GetSeasonPlayerLeaders рассчитывает всех лидеров сезона на основе матчевой статистики.
func GetSeasonPlayerLeaders(ctx context.Context, pool *pgxpool.Pool, info *LeagueSeasonInfo, limit int) (*SeasonLeadersData, error) {
	if limit <= 0 {
		limit = 10
	}

	data := &SeasonLeadersData{
		LeagueID:   info.LeagueID,
		LeagueName: info.LeagueName,
		SeasonID:   info.SeasonID,
		SeasonName: info.SeasonName,
		Locale:     info.Locale,
	}

	// 1. Топ бомбардиров
	scorers, err := queryTopScorers(ctx, pool, info.SeasonID, info.LeagueID, info.Locale, limit)
	if err != nil {
		return nil, fmt.Errorf("queryTopScorers: %w", err)
	}
	data.TopScorers = scorers

	// 2. Топ ассистентов
	assists, err := queryTopAssists(ctx, pool, info.SeasonID, info.LeagueID, info.Locale, limit)
	if err != nil {
		return nil, fmt.Errorf("queryTopAssists: %w", err)
	}
	data.TopAssists = assists

	// 3. Топ Гол + Пас
	points, err := queryTopPoints(ctx, pool, info.SeasonID, info.LeagueID, info.Locale, limit)
	if err != nil {
		return nil, fmt.Errorf("queryTopPoints: %w", err)
	}
	data.TopPoints = points

	// 4. «Сухие» матчи вратарей
	cleanSheets, err := queryCleanSheets(ctx, pool, info.SeasonID, info.LeagueID, info.Locale, limit)
	if err != nil {
		return nil, fmt.Errorf("queryCleanSheets: %w", err)
	}
	data.CleanSheets = cleanSheets

	// 5. Дисциплина (карточки)
	discipline, err := queryDiscipline(ctx, pool, info.SeasonID, info.LeagueID, info.Locale, limit)
	if err != nil {
		return nil, fmt.Errorf("queryDiscipline: %w", err)
	}
	data.Discipline = discipline

	return data, nil
}

func hasSeasonsTopscores(ctx context.Context, pool *pgxpool.Pool, seasonID int) bool {
	var exists bool
	_ = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM provider_seasons_topscores WHERE season_id = $1)", seasonID).Scan(&exists)
	return exists
}

func queryTopScorers(ctx context.Context, pool *pgxpool.Pool, seasonID, leagueID int, locale string, limit int) ([]PlayerLeaderItem, error) {
	var query string
	if hasSeasonsTopscores(ctx, pool, seasonID) {
		query = fmt.Sprintf(`
			SELECT 
				pst.player_id,
				COALESCE(pp.display_name, pp.common_name, pp.name, '') AS player_name,
				COALESCE(pp.image_path, '') AS player_image,
				pt.id AS team_id,
				COALESCE(
					CASE 
						WHEN $3 = 'ru' THEN NULLIF(pt.locale->'ru'->>'long', '')
						WHEN $3 = 'en' THEN NULLIF(pt.locale->'en'->>'long', '')
						WHEN $3 = 'ua' THEN NULLIF(pt.locale->'ua'->>'long', '')
					END,
					pt.name, ''
				) AS team_name,
				COALESCE(pt.short_code, '') AS team_code,
				COALESCE(pt.image_path, '') AS team_logo,
				COALESCE(matches_sub.cnt, 0) AS matches,
				pst.total AS goals,
				COALESCE(pen.total, 0) AS penalties
			FROM provider_seasons_topscores pst
			JOIN provider_players pp ON pst.player_id = pp.id
			JOIN provider_teams pt ON pst.participant_id = pt.id
			LEFT JOIN provider_seasons_topscores pen 
				   ON pen.season_id = pst.season_id 
				  AND pen.player_id = pst.player_id 
				  AND pen.type_id = 1600
			LEFT JOIN LATERAL (
				SELECT COUNT(DISTINCT plu.fixture_id) AS cnt
				FROM provider_lineups_%d plu
				JOIN provider_fixtures pf ON pf.id = plu.fixture_id
				WHERE pf.season_id = $1 AND plu.player_id = pst.player_id AND plu.league_id = $2
			) matches_sub ON TRUE
			WHERE pst.season_id = $1 AND pst.type_id = 208
			ORDER BY pst.total DESC, penalties ASC, pst.position ASC
			LIMIT $4
		`, leagueID)
	} else {
		query = fmt.Sprintf(`
			SELECT 
				pp.id AS player_id,
				COALESCE(pp.display_name, pp.common_name, pp.name, '') AS player_name,
				COALESCE(pp.image_path, '') AS player_image,
				pt.id AS team_id,
				COALESCE(
					CASE 
						WHEN $3 = 'ru' THEN NULLIF(pt.locale->'ru'->>'long', '')
						WHEN $3 = 'en' THEN NULLIF(pt.locale->'en'->>'long', '')
						WHEN $3 = 'ua' THEN NULLIF(pt.locale->'ua'->>'long', '')
					END,
					pt.name, ''
				) AS team_name,
				COALESCE(pt.short_code, '') AS team_code,
				COALESCE(pt.image_path, '') AS team_logo,
				COUNT(DISTINCT pfps.fixture_id) AS matches,
				COALESCE(SUM(CASE WHEN ptype.developer_name = 'GOALS' THEN pfps.value ELSE 0 END), 0)::int AS goals,
				COALESCE(SUM(CASE WHEN ptype.developer_name = 'PENALTIES_SCORED' THEN pfps.value ELSE 0 END), 0)::int AS penalties
			FROM provider_fixtures_players_statistics_%d pfps
			JOIN provider_fixtures pf ON pfps.fixture_id = pf.id
			JOIN provider_players pp ON pfps.player_id = pp.id
			JOIN provider_teams pt ON pfps.team_id = pt.id
			JOIN provider_types ptype ON pfps.type_id = ptype.id
			WHERE pf.season_id = $1 AND pfps.league_id = $2
			GROUP BY pp.id, pp.display_name, pp.common_name, pp.name, pp.image_path, 
			         pt.id, pt.name, pt.short_code, pt.image_path, pt.locale
			HAVING SUM(CASE WHEN ptype.developer_name = 'GOALS' THEN pfps.value ELSE 0 END) > 0
			ORDER BY goals DESC, penalties ASC, matches ASC, pp.id ASC
			LIMIT $4
		`, leagueID)
	}

	rows, err := pool.Query(ctx, query, seasonID, leagueID, locale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PlayerLeaderItem
	rank := 1
	for rows.Next() {
		var item PlayerLeaderItem
		if err := rows.Scan(
			&item.PlayerID,
			&item.PlayerName,
			&item.PlayerImage,
			&item.TeamID,
			&item.TeamName,
			&item.TeamCode,
			&item.TeamLogo,
			&item.Matches,
			&item.PrimaryVal,
			&item.SecondaryVal,
		); err != nil {
			return nil, err
		}
		item.Rank = rank
		item.MetricLabel = "голов"
		items = append(items, item)
		rank++
	}
	return items, rows.Err()
}

func queryTopAssists(ctx context.Context, pool *pgxpool.Pool, seasonID, leagueID int, locale string, limit int) ([]PlayerLeaderItem, error) {
	var query string
	if hasSeasonsTopscores(ctx, pool, seasonID) {
		query = fmt.Sprintf(`
			SELECT 
				pst.player_id,
				COALESCE(pp.display_name, pp.common_name, pp.name, '') AS player_name,
				COALESCE(pp.image_path, '') AS player_image,
				pt.id AS team_id,
				COALESCE(
					CASE 
						WHEN $3 = 'ru' THEN NULLIF(pt.locale->'ru'->>'long', '')
						WHEN $3 = 'en' THEN NULLIF(pt.locale->'en'->>'long', '')
						WHEN $3 = 'ua' THEN NULLIF(pt.locale->'ua'->>'long', '')
					END,
					pt.name, ''
				) AS team_name,
				COALESCE(pt.short_code, '') AS team_code,
				COALESCE(pt.image_path, '') AS team_logo,
				COALESCE(matches_sub.cnt, 0) AS matches,
				pst.total AS assists,
				COALESCE(goals.total, 0) AS goals
			FROM provider_seasons_topscores pst
			JOIN provider_players pp ON pst.player_id = pp.id
			JOIN provider_teams pt ON pst.participant_id = pt.id
			LEFT JOIN provider_seasons_topscores goals 
				   ON goals.season_id = pst.season_id 
				  AND goals.player_id = pst.player_id 
				  AND goals.type_id = 208
			LEFT JOIN LATERAL (
				SELECT COUNT(DISTINCT plu.fixture_id) AS cnt
				FROM provider_lineups_%d plu
				JOIN provider_fixtures pf ON pf.id = plu.fixture_id
				WHERE pf.season_id = $1 AND plu.player_id = pst.player_id AND plu.league_id = $2
			) matches_sub ON TRUE
			WHERE pst.season_id = $1 AND pst.type_id = 209
			ORDER BY pst.total DESC, goals DESC, pst.position ASC
			LIMIT $4
		`, leagueID)
	} else {
		query = fmt.Sprintf(`
			SELECT 
				pp.id AS player_id,
				COALESCE(pp.display_name, pp.common_name, pp.name, '') AS player_name,
				COALESCE(pp.image_path, '') AS player_image,
				pt.id AS team_id,
				COALESCE(
					CASE 
						WHEN $3 = 'ru' THEN NULLIF(pt.locale->'ru'->>'long', '')
						WHEN $3 = 'en' THEN NULLIF(pt.locale->'en'->>'long', '')
						WHEN $3 = 'ua' THEN NULLIF(pt.locale->'ua'->>'long', '')
					END,
					pt.name, ''
				) AS team_name,
				COALESCE(pt.short_code, '') AS team_code,
				COALESCE(pt.image_path, '') AS team_logo,
				COUNT(DISTINCT pfps.fixture_id) AS matches,
				COALESCE(SUM(CASE WHEN ptype.developer_name = 'ASSISTS' THEN pfps.value ELSE 0 END), 0)::int AS assists,
				COALESCE(SUM(CASE WHEN ptype.developer_name = 'GOALS' THEN pfps.value ELSE 0 END), 0)::int AS goals
			FROM provider_fixtures_players_statistics_%d pfps
			JOIN provider_fixtures pf ON pfps.fixture_id = pf.id
			JOIN provider_players pp ON pfps.player_id = pp.id
			JOIN provider_teams pt ON pfps.team_id = pt.id
			JOIN provider_types ptype ON pfps.type_id = ptype.id
			WHERE pf.season_id = $1 AND pfps.league_id = $2
			GROUP BY pp.id, pp.display_name, pp.common_name, pp.name, pp.image_path, 
			         pt.id, pt.name, pt.short_code, pt.image_path, pt.locale
			HAVING SUM(CASE WHEN ptype.developer_name = 'ASSISTS' THEN pfps.value ELSE 0 END) > 0
			ORDER BY assists DESC, goals DESC, matches ASC, pp.id ASC
			LIMIT $4
		`, leagueID)
	}

	rows, err := pool.Query(ctx, query, seasonID, leagueID, locale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PlayerLeaderItem
	rank := 1
	for rows.Next() {
		var item PlayerLeaderItem
		if err := rows.Scan(
			&item.PlayerID,
			&item.PlayerName,
			&item.PlayerImage,
			&item.TeamID,
			&item.TeamName,
			&item.TeamCode,
			&item.TeamLogo,
			&item.Matches,
			&item.PrimaryVal,
			&item.SecondaryVal,
		); err != nil {
			return nil, err
		}
		item.Rank = rank
		item.MetricLabel = "ассистов"
		items = append(items, item)
		rank++
	}
	return items, rows.Err()
}

func queryTopPoints(ctx context.Context, pool *pgxpool.Pool, seasonID, leagueID int, locale string, limit int) ([]PlayerLeaderItem, error) {
	var query string
	if hasSeasonsTopscores(ctx, pool, seasonID) {
		query = fmt.Sprintf(`
			SELECT 
				sub.player_id,
				COALESCE(pp.display_name, pp.common_name, pp.name, '') AS player_name,
				COALESCE(pp.image_path, '') AS player_image,
				pt.id AS team_id,
				COALESCE(
					CASE 
						WHEN $3 = 'ru' THEN NULLIF(pt.locale->'ru'->>'long', '')
						WHEN $3 = 'en' THEN NULLIF(pt.locale->'en'->>'long', '')
						WHEN $3 = 'ua' THEN NULLIF(pt.locale->'ua'->>'long', '')
					END,
					pt.name, ''
				) AS team_name,
				COALESCE(pt.short_code, '') AS team_code,
				COALESCE(pt.image_path, '') AS team_logo,
				COALESCE(matches_sub.cnt, 0) AS matches,
				(sub.goals + sub.assists) AS points,
				sub.goals AS goals
			FROM (
				SELECT 
					player_id,
					MAX(participant_id) AS participant_id,
					COALESCE(SUM(CASE WHEN type_id = 208 THEN total ELSE 0 END), 0)::int AS goals,
					COALESCE(SUM(CASE WHEN type_id = 209 THEN total ELSE 0 END), 0)::int AS assists
				FROM provider_seasons_topscores
				WHERE season_id = $1 AND type_id IN (208, 209)
				GROUP BY player_id
				HAVING (COALESCE(SUM(CASE WHEN type_id = 208 THEN total ELSE 0 END), 0) +
				        COALESCE(SUM(CASE WHEN type_id = 209 THEN total ELSE 0 END), 0)) > 0
			) sub
			JOIN provider_players pp ON sub.player_id = pp.id
			JOIN provider_teams pt ON sub.participant_id = pt.id
			LEFT JOIN LATERAL (
				SELECT COUNT(DISTINCT plu.fixture_id) AS cnt
				FROM provider_lineups_%d plu
				JOIN provider_fixtures pf ON pf.id = plu.fixture_id
				WHERE pf.season_id = $1 AND plu.player_id = sub.player_id AND plu.league_id = $2
			) matches_sub ON TRUE
			ORDER BY points DESC, goals DESC, matches ASC, pp.id ASC
			LIMIT $4
		`, leagueID)
	} else {
		query = fmt.Sprintf(`
			SELECT 
				pp.id AS player_id,
				COALESCE(pp.display_name, pp.common_name, pp.name, '') AS player_name,
				COALESCE(pp.image_path, '') AS player_image,
				pt.id AS team_id,
				COALESCE(
					CASE 
						WHEN $3 = 'ru' THEN NULLIF(pt.locale->'ru'->>'long', '')
						WHEN $3 = 'en' THEN NULLIF(pt.locale->'en'->>'long', '')
						WHEN $3 = 'ua' THEN NULLIF(pt.locale->'ua'->>'long', '')
					END,
					pt.name, ''
				) AS team_name,
				COALESCE(pt.short_code, '') AS team_code,
				COALESCE(pt.image_path, '') AS team_logo,
				COUNT(DISTINCT pfps.fixture_id) AS matches,
				(COALESCE(SUM(CASE WHEN ptype.developer_name = 'GOALS' THEN pfps.value ELSE 0 END), 0) +
				 COALESCE(SUM(CASE WHEN ptype.developer_name = 'ASSISTS' THEN pfps.value ELSE 0 END), 0))::int AS points,
				COALESCE(SUM(CASE WHEN ptype.developer_name = 'GOALS' THEN pfps.value ELSE 0 END), 0)::int AS goals
			FROM provider_fixtures_players_statistics_%d pfps
			JOIN provider_fixtures pf ON pfps.fixture_id = pf.id
			JOIN provider_players pp ON pfps.player_id = pp.id
			JOIN provider_teams pt ON pfps.team_id = pt.id
			JOIN provider_types ptype ON pfps.type_id = ptype.id
			WHERE pf.season_id = $1 AND pfps.league_id = $2
			GROUP BY pp.id, pp.display_name, pp.common_name, pp.name, pp.image_path, 
			         pt.id, pt.name, pt.short_code, pt.image_path, pt.locale
			HAVING (COALESCE(SUM(CASE WHEN ptype.developer_name = 'GOALS' THEN pfps.value ELSE 0 END), 0) +
			        COALESCE(SUM(CASE WHEN ptype.developer_name = 'ASSISTS' THEN pfps.value ELSE 0 END), 0)) > 0
			ORDER BY points DESC, goals DESC, matches ASC, pp.id ASC
			LIMIT $4
		`, leagueID)
	}

	rows, err := pool.Query(ctx, query, seasonID, leagueID, locale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PlayerLeaderItem
	rank := 1
	for rows.Next() {
		var item PlayerLeaderItem
		if err := rows.Scan(
			&item.PlayerID,
			&item.PlayerName,
			&item.PlayerImage,
			&item.TeamID,
			&item.TeamName,
			&item.TeamCode,
			&item.TeamLogo,
			&item.Matches,
			&item.PrimaryVal,
			&item.SecondaryVal,
		); err != nil {
			return nil, err
		}
		item.Rank = rank
		item.MetricLabel = "очков (Г+П)"
		items = append(items, item)
		rank++
	}
	return items, rows.Err()
}

func queryCleanSheets(ctx context.Context, pool *pgxpool.Pool, seasonID, leagueID int, locale string, limit int) ([]PlayerLeaderItem, error) {
	query := fmt.Sprintf(`
		SELECT 
			pp.id AS player_id,
			COALESCE(pp.display_name, pp.common_name, pp.name, '') AS player_name,
			COALESCE(pp.image_path, '') AS player_image,
			pt.id AS team_id,
			COALESCE(
				CASE 
					WHEN $3 = 'ru' THEN NULLIF(pt.locale->'ru'->>'long', '')
					WHEN $3 = 'en' THEN NULLIF(pt.locale->'en'->>'long', '')
					WHEN $3 = 'ua' THEN NULLIF(pt.locale->'ua'->>'long', '')
				END,
				pt.name, ''
			) AS team_name,
			COALESCE(pt.short_code, '') AS team_code,
			COALESCE(pt.image_path, '') AS team_logo,
			COUNT(DISTINCT pf.id) AS matches,
			COUNT(DISTINCT CASE WHEN opp_score.goals = 0 THEN pf.id END) AS clean_sheets,
			COALESCE(SUM(opp_score.goals), 0)::int AS goals_conceded
		FROM provider_lineups_%d plu
		JOIN provider_players pp ON plu.player_id = pp.id
		JOIN provider_teams pt ON plu.team_id = pt.id
		JOIN provider_fixtures pf ON plu.fixture_id = pf.id
		JOIN provider_fixtures_participants pfp ON (pf.id = pfp.fixture_id AND pfp.participant_id = plu.team_id)
		JOIN provider_fixtures_scores opp_score ON (pf.id = opp_score.fixture_id AND opp_score.type_id = 1525 AND opp_score.participant != pfp.location)
		WHERE pf.season_id = $1 AND pf.state_id = 5 
		  AND plu.type_id = 11 AND plu.position_id = 24
		  AND plu.league_id = $2
		GROUP BY pp.id, pp.display_name, pp.common_name, pp.name, pp.image_path, 
		         pt.id, pt.name, pt.short_code, pt.image_path, pt.locale
		ORDER BY clean_sheets DESC, goals_conceded ASC, matches ASC, pp.id ASC
		LIMIT $4
	`, leagueID)

	rows, err := pool.Query(ctx, query, seasonID, leagueID, locale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PlayerLeaderItem
	rank := 1
	for rows.Next() {
		var item PlayerLeaderItem
		if err := rows.Scan(
			&item.PlayerID,
			&item.PlayerName,
			&item.PlayerImage,
			&item.TeamID,
			&item.TeamName,
			&item.TeamCode,
			&item.TeamLogo,
			&item.Matches,
			&item.PrimaryVal,
			&item.SecondaryVal,
		); err != nil {
			return nil, err
		}
		item.Rank = rank
		item.MetricLabel = "сухих матчей"
		items = append(items, item)
		rank++
	}
	return items, rows.Err()
}

func queryDiscipline(ctx context.Context, pool *pgxpool.Pool, seasonID, leagueID int, locale string, limit int) ([]PlayerLeaderItem, error) {
	var query string
	if hasSeasonsTopscores(ctx, pool, seasonID) {
		query = fmt.Sprintf(`
			SELECT 
				sub.player_id,
				COALESCE(pp.display_name, pp.common_name, pp.name, '') AS player_name,
				COALESCE(pp.image_path, '') AS player_image,
				pt.id AS team_id,
				COALESCE(
					CASE 
						WHEN $3 = 'ru' THEN NULLIF(pt.locale->'ru'->>'long', '')
						WHEN $3 = 'en' THEN NULLIF(pt.locale->'en'->>'long', '')
						WHEN $3 = 'ua' THEN NULLIF(pt.locale->'ua'->>'long', '')
					END,
					pt.name, ''
				) AS team_name,
				COALESCE(pt.short_code, '') AS team_code,
				COALESCE(pt.image_path, '') AS team_logo,
				COALESCE(matches_sub.cnt, 0) AS matches,
				sub.yellow_cards,
				sub.red_cards
			FROM (
				SELECT 
					player_id,
					MAX(participant_id) AS participant_id,
					COALESCE(SUM(CASE WHEN type_id = 84 THEN total ELSE 0 END), 0)::int AS yellow_cards,
					COALESCE(SUM(CASE WHEN type_id = 83 THEN total ELSE 0 END), 0)::int AS red_cards
				FROM provider_seasons_topscores
				WHERE season_id = $1 AND type_id IN (83, 84)
				GROUP BY player_id
				HAVING (COALESCE(SUM(CASE WHEN type_id = 84 THEN total ELSE 0 END), 0) +
				        COALESCE(SUM(CASE WHEN type_id = 83 THEN total ELSE 0 END), 0)) > 0
			) sub
			JOIN provider_players pp ON sub.player_id = pp.id
			JOIN provider_teams pt ON sub.participant_id = pt.id
			LEFT JOIN LATERAL (
				SELECT COUNT(DISTINCT plu.fixture_id) AS cnt
				FROM provider_lineups_%d plu
				JOIN provider_fixtures pf ON pf.id = plu.fixture_id
				WHERE pf.season_id = $1 AND plu.player_id = sub.player_id AND plu.league_id = $2
			) matches_sub ON TRUE
			ORDER BY sub.red_cards DESC, sub.yellow_cards DESC, matches ASC, pp.id ASC
			LIMIT $4
		`, leagueID)
	} else {
		query = fmt.Sprintf(`
			SELECT 
				pp.id AS player_id,
				COALESCE(pp.display_name, pp.common_name, pp.name, '') AS player_name,
				COALESCE(pp.image_path, '') AS player_image,
				pt.id AS team_id,
				COALESCE(
					CASE 
						WHEN $3 = 'ru' THEN NULLIF(pt.locale->'ru'->>'long', '')
						WHEN $3 = 'en' THEN NULLIF(pt.locale->'en'->>'long', '')
						WHEN $3 = 'ua' THEN NULLIF(pt.locale->'ua'->>'long', '')
					END,
					pt.name, ''
				) AS team_name,
				COALESCE(pt.short_code, '') AS team_code,
				COALESCE(pt.image_path, '') AS team_logo,
				COUNT(DISTINCT pfps.fixture_id) AS matches,
				COALESCE(SUM(CASE WHEN ptype.developer_name = 'YELLOWCARDS' THEN pfps.value ELSE 0 END), 0)::int AS yellow_cards,
				COALESCE(SUM(CASE WHEN ptype.developer_name = 'REDCARDS' THEN pfps.value ELSE 0 END), 0)::int AS red_cards
			FROM provider_fixtures_players_statistics_%d pfps
			JOIN provider_fixtures pf ON pfps.fixture_id = pf.id
			JOIN provider_players pp ON pfps.player_id = pp.id
			JOIN provider_teams pt ON pfps.team_id = pt.id
			JOIN provider_types ptype ON pfps.type_id = ptype.id
			WHERE pf.season_id = $1 AND pfps.league_id = $2
			GROUP BY pp.id, pp.display_name, pp.common_name, pp.name, pp.image_path, 
			         pt.id, pt.name, pt.short_code, pt.image_path, pt.locale
			HAVING (COALESCE(SUM(CASE WHEN ptype.developer_name = 'YELLOWCARDS' THEN pfps.value ELSE 0 END), 0) +
			        COALESCE(SUM(CASE WHEN ptype.developer_name = 'REDCARDS' THEN pfps.value ELSE 0 END), 0)) > 0
			ORDER BY red_cards DESC, yellow_cards DESC, matches ASC, pp.id ASC
			LIMIT $4
		`, leagueID)
	}

	rows, err := pool.Query(ctx, query, seasonID, leagueID, locale, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []PlayerLeaderItem
	rank := 1
	for rows.Next() {
		var item PlayerLeaderItem
		if err := rows.Scan(
			&item.PlayerID,
			&item.PlayerName,
			&item.PlayerImage,
			&item.TeamID,
			&item.TeamName,
			&item.TeamCode,
			&item.TeamLogo,
			&item.Matches,
			&item.PrimaryVal,
			&item.SecondaryVal,
		); err != nil {
			return nil, err
		}
		item.Rank = rank
		item.MetricLabel = "ЖК"
		items = append(items, item)
		rank++
	}
	return items, rows.Err()
}
