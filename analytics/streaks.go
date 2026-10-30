package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Константы типов сущностей
const (
	EntityTypeTeam   = "team"
	EntityTypeCoach  = "coach"
	EntityTypePlayer = "player"
	EntityTypeH2H    = "h2h"
)

// Константы типов серий
const (
	StreakTypeTeamWin        = "team_win"
	StreakTypeTeamUnbeaten   = "team_unbeaten"
	StreakTypeTeamNoWin      = "team_no_win"
	StreakTypeTeamScoring    = "team_scoring"
	StreakTypeTeamCleanSheet = "team_clean_sheet"
	StreakTypeCoachWin       = "coach_win"
	StreakTypeCoachUnbeaten  = "coach_unbeaten"
	StreakTypePlayerGoal     = "player_goal"
	StreakTypePlayerGA       = "player_g_a"
	StreakTypeH2HUnbeaten    = "h2h_unbeaten"
	StreakTypeH2HNoWin       = "h2h_no_win"
)

// Константы областей действия (Scope)
const (
	StreakScopeLeague = "league"
	StreakScopeAll    = "all"
	StreakScopeHome   = "home"
	StreakScopeAway   = "away"
)

// StreakMatchHistoryItem описывает один матч внутри цепочки серии
type StreakMatchHistoryItem struct {
	FixtureID     int64     `json:"fixture_id"`
	Date          time.Time `json:"date"`
	DateStr       string    `json:"date_str"`
	OpponentID    int64     `json:"opponent_id"`
	OpponentName  string    `json:"opponent_name"`
	OpponentLogo  string    `json:"opponent_logo"`
	HomeScore     int       `json:"home_score"`
	AwayScore     int       `json:"away_score"`
	IsHome        bool      `json:"is_home"`
	Result        string    `json:"result"` // "W", "D", "L"
	TeamGoals     int       `json:"team_goals"`
	OpponentGoals int       `json:"opponent_goals"`
}

// StreakRecord представляет сохранённую или рассчитанную запись серии
type StreakRecord struct {
	ID                  int64                    `json:"id"`
	LeagueID            int                      `json:"league_id"`
	LeagueName          string                   `json:"league_name"`
	EntityType          string                   `json:"entity_type"`
	EntityID            int64                    `json:"entity_id"`
	EntityName          string                   `json:"entity_name"`
	EntityLogo          string                   `json:"entity_logo"`
	SecondaryEntityID   *int64                   `json:"secondary_entity_id,omitempty"`
	SecondaryEntityName string                   `json:"secondary_entity_name,omitempty"`
	SecondaryEntityLogo string                   `json:"secondary_entity_logo,omitempty"`
	CoachID             *int                     `json:"coach_id,omitempty"`
	CoachName           string                   `json:"coach_name,omitempty"`
	CoachImage          string                   `json:"coach_image,omitempty"`
	StreakType          string                   `json:"streak_type"`
	StreakTypeName      string                   `json:"streak_type_name"`
	StreakTypeLongName  string                   `json:"streak_type_long_name"`
	StreakTypeIcon      string                   `json:"streak_type_icon"`
	Scope               string                   `json:"scope"`
	ScopeName           string                   `json:"scope_name"`
	Length              int                      `json:"length"`
	IsActive            bool                     `json:"is_active"`
	IsRecordClub        bool                     `json:"is_record_club"`
	IsRecordLeague      bool                     `json:"is_record_league"`
	PrevClubRecord      int                      `json:"prev_club_record,omitempty"`
	PrevLeagueRecord    int                      `json:"prev_league_record,omitempty"`
	StartFixtureID      *int64                   `json:"start_fixture_id,omitempty"`
	StartSeasonID       *int                     `json:"start_season_id,omitempty"`
	StartSeasonName     string                   `json:"start_season_name,omitempty"`
	StartRoundID        *int                     `json:"start_round_id,omitempty"`
	StartDate           time.Time                `json:"start_date"`
	EndFixtureID        *int64                   `json:"end_fixture_id,omitempty"`
	EndSeasonID         *int                     `json:"end_season_id,omitempty"`
	EndSeasonName       string                   `json:"end_season_name,omitempty"`
	EndRoundID          *int                     `json:"end_round_id,omitempty"`
	EndDate             time.Time                `json:"end_date"`
	BreakingFixtureID   *int64                   `json:"breaking_fixture_id,omitempty"`
	FixturesHistory     []StreakMatchHistoryItem `json:"fixtures_history"`
	Metrics             map[string]interface{}   `json:"metrics"`
	InterestScore       float64                  `json:"interest_score"`
}

// StreaksFilter задает параметры фильтрации для перерасчёта серий
type StreaksFilter struct {
	LeagueID   int
	SeasonID   int
	TeamID     int64
	CoachID    int
	PlayerID   int64
	StreakType string
	Scope      string
	MinLength  int
	AllHistory bool
	Locale     string
}

type rawTeamMatchRow struct {
	fixtureID     int64
	leagueID      int
	seasonID      int
	seasonName    string
	roundID       *int
	startingAt    time.Time
	homeID        int64
	homeName      string
	homeLogo      string
	awayID        int64
	awayName      string
	awayLogo      string
	homeGoals     int
	awayGoals     int
	coachID       *int
	coachName     string
	coachImage    string
	teamIsHome    bool
	teamGoals     int
	opponentGoals int
	opponentID    int64
	opponentName  string
	opponentLogo  string
	result        string // "W", "D", "L"
}

// ComputeAndSaveTeamStreaks выполняет детерминированный расчёт серий команд
// с сохранением в таблицу analytics_streaks и поддержкой гранулярной фильтрации.
func ComputeAndSaveTeamStreaks(ctx context.Context, pool *pgxpool.Pool, filter StreaksFilter) ([]StreakRecord, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}

	locale := filter.Locale
	if locale == "" {
		locale = "ru"
	}

	if filter.MinLength <= 0 {
		filter.MinLength = 3 // минимальная длина серии для фиксации
	}

	// 1. Получаем список команд для расчёта
	teamIDs, err := getTargetTeamIDs(ctx, pool, filter)
	if err != nil {
		return nil, fmt.Errorf("getTargetTeamIDs: %w", err)
	}

	// 2. Загружаем метаданные справочников типов и областей
	typesMap, err := LoadStreakTypesMetadata(ctx, pool, locale)
	if err != nil {
		return nil, fmt.Errorf("LoadStreakTypesMetadata: %w", err)
	}
	scopesMap, err := LoadStreakScopesMetadata(ctx, pool, locale)
	if err != nil {
		return nil, fmt.Errorf("LoadStreakScopesMetadata: %w", err)
	}

	var allCalculatedRecords []StreakRecord

	// 3. Расчёт серий для каждого клуба
	for _, teamID := range teamIDs {
		matches, mErr := fetchTeamChronologicalMatches(ctx, pool, teamID, filter, locale)
		if mErr != nil {
			return nil, fmt.Errorf("fetchTeamChronologicalMatches for team %d: %w", teamID, mErr)
		}
		if len(matches) < filter.MinLength {
			continue
		}

		teamRecords := detectTeamStreaksFromMatches(matches, filter, typesMap, scopesMap)
		for _, rec := range teamRecords {
			if rec.Length >= filter.MinLength {
				allCalculatedRecords = append(allCalculatedRecords, rec)
			}
		}
	}

	// 4. Определение клубных и лиговых рекордов
	evaluateRecordsAndScores(ctx, pool, allCalculatedRecords)

	// 5. Очищаем предыдущие активные серии для обрабатываемых команд
	if len(teamIDs) > 0 {
		_, _ = pool.Exec(ctx, "DELETE FROM analytics_streaks WHERE entity_type = 'team' AND entity_id = ANY($1) AND is_active = true;", teamIDs)
	}

	for i := range allCalculatedRecords {
		if err := saveStreakRecord(ctx, pool, &allCalculatedRecords[i]); err != nil {
			return nil, fmt.Errorf("saveStreakRecord: %w", err)
		}
	}

	return allCalculatedRecords, nil
}

func getTargetTeamIDs(ctx context.Context, pool *pgxpool.Pool, filter StreaksFilter) ([]int64, error) {
	if filter.TeamID > 0 {
		return []int64{filter.TeamID}, nil
	}

	var query string
	var args []interface{}

	if filter.SeasonID > 0 {
		query = `
			SELECT DISTINCT participant_id 
			FROM provider_fixtures_participants pfp
			JOIN provider_fixtures pf ON pf.id = pfp.fixture_id
			WHERE pf.season_id = $1
			ORDER BY participant_id;
		`
		args = append(args, filter.SeasonID)
	} else if filter.LeagueID > 0 {
		query = `
			SELECT DISTINCT participant_id 
			FROM provider_fixtures_participants pfp
			JOIN provider_fixtures pf ON pf.id = pfp.fixture_id
			JOIN provider_seasons ps ON ps.id = pf.season_id AND ps.is_current = true
			WHERE pf.league_id = $1
			ORDER BY participant_id;
		`
		args = append(args, filter.LeagueID)
	} else {
		query = `
			SELECT DISTINCT participant_id 
			FROM provider_fixtures_participants pfp
			JOIN provider_fixtures pf ON pf.id = pfp.fixture_id
			JOIN provider_seasons ps ON ps.id = pf.season_id AND ps.is_current = true
			JOIN provider_leagues pl ON pl.id = pf.league_id AND pl.active = true
			ORDER BY participant_id;
		`
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func fetchTeamChronologicalMatches(ctx context.Context, pool *pgxpool.Pool, teamID int64, filter StreaksFilter, locale string) ([]rawTeamMatchRow, error) {
	query := `
		SELECT pf.id,
		       pf.league_id,
		       pf.season_id,
		       ps.name AS season_name,
		       pf.round_id,
		       pf.starting_at,
		       fph.participant_id AS home_id,
		       COALESCE(NULLIF(pth.locale->$2->>'short', ''), NULLIF(pth.locale->$2->>'long', ''), pth.name) AS home_name,
		       COALESCE(pth.image_path, '') AS home_logo,
		       fpa.participant_id AS away_id,
		       COALESCE(NULLIF(pta.locale->$2->>'short', ''), NULLIF(pta.locale->$2->>'long', ''), pta.name) AS away_name,
		       COALESCE(pta.image_path, '') AS away_logo,
		       COALESCE(sh.goals, 0) AS home_goals,
		       COALESCE(sa.goals, 0) AS away_goals,
		       pfc.coaches_id AS coach_id,
		       COALESCE(pc.name, pc.common_name, '') AS coach_name,
		       COALESCE(pc.image_path, '') AS coach_image
		FROM provider_fixtures pf
		JOIN provider_fixtures_participants fph ON fph.fixture_id = pf.id AND fph.location = 'home'
		JOIN provider_teams pth ON pth.id = fph.participant_id
		JOIN provider_fixtures_participants fpa ON fpa.fixture_id = pf.id AND fpa.location = 'away'
		JOIN provider_teams pta ON pta.id = fpa.participant_id
		JOIN provider_seasons ps ON ps.id = pf.season_id
		LEFT JOIN provider_fixtures_scores sh ON sh.fixture_id = pf.id AND sh.participant_id = pth.id AND sh.type_id = 1525
		LEFT JOIN provider_fixtures_scores sa ON sa.fixture_id = pf.id AND sa.participant_id = pta.id AND sa.type_id = 1525
		LEFT JOIN provider_fixtures_coaches pfc ON pfc.fixture_id = pf.id AND pfc.participant_id = $1
		LEFT JOIN provider_coachs pc ON pc.id = pfc.coaches_id
		WHERE (fph.participant_id = $1 OR fpa.participant_id = $1)
		  AND pf.starting_at < NOW()
		  AND sh.goals IS NOT NULL
		  AND sa.goals IS NOT NULL
	`

	var args []interface{}
	args = append(args, teamID, locale)

	if filter.LeagueID > 0 {
		args = append(args, filter.LeagueID)
		query += fmt.Sprintf(" AND pf.league_id = $%d", len(args))
	}

	if filter.SeasonID > 0 && !filter.AllHistory {
		args = append(args, filter.SeasonID)
		query += fmt.Sprintf(" AND pf.season_id = $%d", len(args))
	}

	query += " ORDER BY pf.starting_at ASC;"

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []rawTeamMatchRow
	for rows.Next() {
		var m rawTeamMatchRow
		if err := rows.Scan(
			&m.fixtureID, &m.leagueID, &m.seasonID, &m.seasonName,
			&m.roundID, &m.startingAt,
			&m.homeID, &m.homeName, &m.homeLogo,
			&m.awayID, &m.awayName, &m.awayLogo,
			&m.homeGoals, &m.awayGoals,
			&m.coachID, &m.coachName, &m.coachImage,
		); err != nil {
			return nil, err
		}

		m.teamIsHome = (m.homeID == teamID)
		if m.teamIsHome {
			m.teamGoals = m.homeGoals
			m.opponentGoals = m.awayGoals
			m.opponentID = m.awayID
			m.opponentName = m.awayName
			m.opponentLogo = m.awayLogo
		} else {
			m.teamGoals = m.awayGoals
			m.opponentGoals = m.homeGoals
			m.opponentID = m.homeID
			m.opponentName = m.homeName
			m.opponentLogo = m.homeLogo
		}

		if m.teamGoals > m.opponentGoals {
			m.result = "W"
		} else if m.teamGoals < m.opponentGoals {
			m.result = "L"
		} else {
			m.result = "D"
		}

		matches = append(matches, m)
	}

	return matches, nil
}

// detectTeamStreaksFromMatches анализирует хронологическую последовательность матчей
func detectTeamStreaksFromMatches(
	matches []rawTeamMatchRow,
	filter StreaksFilter,
	typesMap map[string]StreakTypeMeta,
	scopesMap map[string]StreakScopeMeta,
) []StreakRecord {
	if len(matches) == 0 {
		return nil
	}

	var records []StreakRecord

	scopes := []string{StreakScopeLeague}
	if filter.Scope != "" {
		scopes = []string{filter.Scope}
	} else {
		scopes = append(scopes, StreakScopeAll, StreakScopeHome, StreakScopeAway)
	}

	teamID := matches[0].homeID
	teamName := matches[0].homeName
	teamLogo := matches[0].homeLogo
	if !matches[0].teamIsHome {
		teamID = matches[0].awayID
		teamName = matches[0].awayName
		teamLogo = matches[0].awayLogo
	}

	for _, scope := range scopes {
		var scopedMatches []rawTeamMatchRow
		for _, m := range matches {
			if scope == StreakScopeHome && !m.teamIsHome {
				continue
			}
			if scope == StreakScopeAway && m.teamIsHome {
				continue
			}
			scopedMatches = append(scopedMatches, m)
		}

		if len(scopedMatches) < filter.MinLength {
			continue
		}

		// 1. Победная серия (team_win)
		if r := calculateSingleStreak(scopedMatches, teamID, teamName, teamLogo, StreakTypeTeamWin, scope, typesMap, scopesMap, func(m rawTeamMatchRow) bool {
			return m.result == "W"
		}); r != nil {
			records = append(records, *r)
		}

		// 2. Беспроигрышная серия (team_unbeaten)
		if r := calculateSingleStreak(scopedMatches, teamID, teamName, teamLogo, StreakTypeTeamUnbeaten, scope, typesMap, scopesMap, func(m rawTeamMatchRow) bool {
			return m.result == "W" || m.result == "D"
		}); r != nil {
			records = append(records, *r)
		}

		// 3. Серия без побед (team_no_win)
		if r := calculateSingleStreak(scopedMatches, teamID, teamName, teamLogo, StreakTypeTeamNoWin, scope, typesMap, scopesMap, func(m rawTeamMatchRow) bool {
			return m.result == "D" || m.result == "L"
		}); r != nil {
			records = append(records, *r)
		}

		// 4. Голевая серия (team_scoring)
		if r := calculateSingleStreak(scopedMatches, teamID, teamName, teamLogo, StreakTypeTeamScoring, scope, typesMap, scopesMap, func(m rawTeamMatchRow) bool {
			return m.teamGoals > 0
		}); r != nil {
			records = append(records, *r)
		}

		// 5. Сухая серия обороны (team_clean_sheet)
		if r := calculateSingleStreak(scopedMatches, teamID, teamName, teamLogo, StreakTypeTeamCleanSheet, scope, typesMap, scopesMap, func(m rawTeamMatchRow) bool {
			return m.opponentGoals == 0
		}); r != nil {
			records = append(records, *r)
		}
	}

	return records
}

func calculateSingleStreak(
	matches []rawTeamMatchRow,
	teamID int64,
	teamName, teamLogo string,
	streakType, scope string,
	typesMap map[string]StreakTypeMeta,
	scopesMap map[string]StreakScopeMeta,
	predicate func(m rawTeamMatchRow) bool,
) *StreakRecord {
	n := len(matches)
	if n == 0 {
		return nil
	}

	// Ищем текущую активную серию (с конца массива к началу)
	streakLen := 0
	var streakMatches []rawTeamMatchRow

	for i := n - 1; i >= 0; i-- {
		if predicate(matches[i]) {
			streakLen++
			streakMatches = append([]rawTeamMatchRow{matches[i]}, streakMatches...)
		} else {
			break
		}
	}

	if streakLen == 0 {
		return nil
	}

	firstMatch := streakMatches[0]
	lastMatch := streakMatches[len(streakMatches)-1]

	var history []StreakMatchHistoryItem
	totalGF := 0
	totalGA := 0

	for _, sm := range streakMatches {
		totalGF += sm.teamGoals
		totalGA += sm.opponentGoals

		dStr := sm.startingAt.Format("02.01")
		history = append(history, StreakMatchHistoryItem{
			FixtureID:     sm.fixtureID,
			Date:          sm.startingAt,
			DateStr:       dStr,
			OpponentID:    sm.opponentID,
			OpponentName:  sm.opponentName,
			OpponentLogo:  sm.opponentLogo,
			HomeScore:     sm.homeGoals,
			AwayScore:     sm.awayGoals,
			IsHome:        sm.teamIsHome,
			Result:        sm.result,
			TeamGoals:     sm.teamGoals,
			OpponentGoals: sm.opponentGoals,
		})
	}

	tMeta := typesMap[streakType]
	sMeta := scopesMap[scope]

	rec := &StreakRecord{
		LeagueID:           lastMatch.leagueID,
		EntityType:         EntityTypeTeam,
		EntityID:           teamID,
		EntityName:         teamName,
		EntityLogo:         teamLogo,
		CoachID:            lastMatch.coachID,
		CoachName:          lastMatch.coachName,
		CoachImage:         lastMatch.coachImage,
		StreakType:         streakType,
		StreakTypeName:     tMeta.ShortName,
		StreakTypeLongName: tMeta.LongName,
		StreakTypeIcon:     tMeta.Icon,
		Scope:              scope,
		ScopeName:          sMeta.LongName,
		Length:             streakLen,
		IsActive:           true,
		StartFixtureID:     &firstMatch.fixtureID,
		StartSeasonID:      &firstMatch.seasonID,
		StartSeasonName:    firstMatch.seasonName,
		StartRoundID:       firstMatch.roundID,
		StartDate:          firstMatch.startingAt,
		EndFixtureID:       &lastMatch.fixtureID,
		EndSeasonID:        &lastMatch.seasonID,
		EndSeasonName:      lastMatch.seasonName,
		EndRoundID:         lastMatch.roundID,
		EndDate:            lastMatch.startingAt,
		FixturesHistory:    history,
		Metrics: map[string]interface{}{
			"goals_scored":   totalGF,
			"goals_conceded": totalGA,
			"clean_sheets":   countCleanSheets(history),
		},
	}

	return rec
}

func countCleanSheets(history []StreakMatchHistoryItem) int {
	cnt := 0
	for _, h := range history {
		if h.OpponentGoals == 0 {
			cnt++
		}
	}
	return cnt
}

// StreakTypeMeta содержит локализованные метаданные типа серии
type StreakTypeMeta struct {
	Code        string
	Category    string
	Icon        string
	ShortName   string
	LongName    string
	Description string
}

// StreakScopeMeta содержит локализованные метаданные области серии
type StreakScopeMeta struct {
	Code        string
	ShortName   string
	LongName    string
	Description string
}

// LoadStreakTypesMetadata читает справочник analytics_streak_types
func LoadStreakTypesMetadata(ctx context.Context, pool *pgxpool.Pool, locale string) (map[string]StreakTypeMeta, error) {
	query := `
		SELECT code, category, icon,
		       COALESCE(locale->$1->>'short', locale->'ru'->>'short', code) AS short_name,
		       COALESCE(locale->$1->>'long', locale->'ru'->>'long', code) AS long_name,
		       COALESCE(locale->$1->>'description', locale->'ru'->>'description', '') AS description
		FROM analytics_streak_types;
	`
	rows, err := pool.Query(ctx, query, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]StreakTypeMeta)
	for rows.Next() {
		var m StreakTypeMeta
		if err := rows.Scan(&m.Code, &m.Category, &m.Icon, &m.ShortName, &m.LongName, &m.Description); err == nil {
			res[m.Code] = m
		}
	}
	return res, nil
}

// LoadStreakScopesMetadata читает справочник analytics_streak_scopes
func LoadStreakScopesMetadata(ctx context.Context, pool *pgxpool.Pool, locale string) (map[string]StreakScopeMeta, error) {
	query := `
		SELECT code,
		       COALESCE(locale->$1->>'short', locale->'ru'->>'short', code) AS short_name,
		       COALESCE(locale->$1->>'long', locale->'ru'->>'long', code) AS long_name,
		       COALESCE(locale->$1->>'description', locale->'ru'->>'description', '') AS description
		FROM analytics_streak_scopes;
	`
	rows, err := pool.Query(ctx, query, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make(map[string]StreakScopeMeta)
	for rows.Next() {
		var m StreakScopeMeta
		if err := rows.Scan(&m.Code, &m.ShortName, &m.LongName, &m.Description); err == nil {
			res[m.Code] = m
		}
	}
	return res, nil
}

func evaluateRecordsAndScores(ctx context.Context, pool *pgxpool.Pool, records []StreakRecord) {
	for i := range records {
		rec := &records[i]

		// 1. Поиск предыдущего клубного рекорда
		var prevClubMax int
		_ = pool.QueryRow(ctx, `
			SELECT COALESCE(MAX(length), 0)
			FROM analytics_streaks
			WHERE entity_id = $1 AND streak_type = $2 AND scope = $3 AND id != $4;
		`, rec.EntityID, rec.StreakType, rec.Scope, rec.ID).Scan(&prevClubMax)

		rec.PrevClubRecord = prevClubMax
		if rec.Length > prevClubMax && prevClubMax > 0 {
			rec.IsRecordClub = true
		}

		// 2. Поиск предыдущего рекорда лиги
		var prevLeagueMax int
		_ = pool.QueryRow(ctx, `
			SELECT COALESCE(MAX(length), 0)
			FROM analytics_streaks
			WHERE league_id = $1 AND streak_type = $2 AND scope = $3 AND id != $4;
		`, rec.LeagueID, rec.StreakType, rec.Scope, rec.ID).Scan(&prevLeagueMax)

		rec.PrevLeagueRecord = prevLeagueMax
		if rec.Length > prevLeagueMax && prevLeagueMax > 0 {
			rec.IsRecordLeague = true
		}

		// 3. Расчёт коэффициента интересности (Interest Score)
		weight := 1.0
		switch rec.StreakType {
		case StreakTypeTeamWin, StreakTypeCoachWin:
			weight = 3.0
		case StreakTypeTeamUnbeaten, StreakTypeCoachUnbeaten:
			weight = 2.0
		case StreakTypePlayerGoal:
			weight = 2.5
		case StreakTypeTeamCleanSheet:
			weight = 1.8
		}

		score := float64(rec.Length) * weight
		if rec.Scope == StreakScopeLeague {
			score *= 1.2
		}
		if rec.IsRecordClub {
			score += 15.0
		}
		if rec.IsRecordLeague {
			score += 30.0
		}

		rec.InterestScore = score
		if rec.Metrics == nil {
			rec.Metrics = make(map[string]interface{})
		}
		rec.Metrics["interest_score"] = score
	}
}

func saveStreakRecord(ctx context.Context, pool *pgxpool.Pool, rec *StreakRecord) error {
	historyJSON, err := json.Marshal(rec.FixturesHistory)
	if err != nil {
		historyJSON = []byte("[]")
	}
	metricsJSON, err := json.Marshal(rec.Metrics)
	if err != nil {
		metricsJSON = []byte("{}")
	}

	query := `
		INSERT INTO analytics_streaks (
			league_id, entity_type, entity_id, secondary_entity_id, coach_id,
			streak_type, scope, length, is_active, is_record_club, is_record_league,
			start_fixture_id, start_season_id, start_round_id, start_date,
			end_fixture_id, end_season_id, end_round_id, end_date,
			breaking_fixture_id, fixtures_history, metrics, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19,
			$20, $21, $22, NOW()
		)
		RETURNING id;
	`

	return pool.QueryRow(ctx, query,
		rec.LeagueID, rec.EntityType, rec.EntityID, rec.SecondaryEntityID, rec.CoachID,
		rec.StreakType, rec.Scope, rec.Length, rec.IsActive, rec.IsRecordClub, rec.IsRecordLeague,
		rec.StartFixtureID, rec.StartSeasonID, rec.StartRoundID, rec.StartDate,
		rec.EndFixtureID, rec.EndSeasonID, rec.EndRoundID, rec.EndDate,
		rec.BreakingFixtureID, historyJSON, metricsJSON,
	).Scan(&rec.ID)
}

// GetTopActiveStreaks возвращает лучшие активные серии для формирования карточек и постов
func GetTopActiveStreaks(ctx context.Context, pool *pgxpool.Pool, filter StreaksFilter, limit int) ([]StreakRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	locale := filter.Locale
	if locale == "" {
		locale = "ru"
	}

	query := `
		SELECT s.id, s.league_id, pl.name AS league_name,
		       s.entity_type, s.entity_id,
		       COALESCE(NULLIF(pt.locale->$1->>'short', ''), NULLIF(pt.locale->$1->>'long', ''), pt.name, '') AS entity_name,
		       COALESCE(pt.image_path, '') AS entity_logo,
		       s.secondary_entity_id,
		       COALESCE(NULLIF(pt2.locale->$1->>'short', ''), NULLIF(pt2.locale->$1->>'long', ''), pt2.name, '') AS secondary_entity_name,
		       COALESCE(pt2.image_path, '') AS secondary_entity_logo,
		       s.coach_id,
		       COALESCE(pc.name, pc.common_name, '') AS coach_name,
		       COALESCE(pc.image_path, '') AS coach_image,
		       s.streak_type,
		       COALESCE(st.locale->$1->>'short', st.locale->'ru'->>'short', st.code) AS streak_type_name,
		       COALESCE(st.locale->$1->>'long', st.locale->'ru'->>'long', st.code) AS streak_type_long_name,
		       st.icon AS streak_type_icon,
		       s.scope,
		       COALESCE(sc.locale->$1->>'long', sc.locale->'ru'->>'long', sc.code) AS scope_name,
		       s.length, s.is_active, s.is_record_club, s.is_record_league,
		       s.start_fixture_id, s.start_season_id, ps1.name AS start_season_name, s.start_round_id, s.start_date,
		       s.end_fixture_id, s.end_season_id, ps2.name AS end_season_name, s.end_round_id, s.end_date,
		       s.breaking_fixture_id, s.fixtures_history, s.metrics
		FROM analytics_streaks s
		JOIN provider_leagues pl ON pl.id = s.league_id
		JOIN analytics_streak_types st ON st.code = s.streak_type
		JOIN analytics_streak_scopes sc ON sc.code = s.scope
		LEFT JOIN provider_teams pt ON pt.id = s.entity_id AND s.entity_type = 'team'
		LEFT JOIN provider_teams pt2 ON pt2.id = s.secondary_entity_id
		LEFT JOIN provider_coachs pc ON pc.id = s.coach_id
		LEFT JOIN provider_seasons ps1 ON ps1.id = s.start_season_id
		LEFT JOIN provider_seasons ps2 ON ps2.id = s.end_season_id
		WHERE s.is_active = true
	`

	var args []interface{}
	args = append(args, locale)

	if filter.LeagueID > 0 {
		args = append(args, filter.LeagueID)
		query += fmt.Sprintf(" AND s.league_id = $%d", len(args))
	}
	if filter.TeamID > 0 {
		args = append(args, filter.TeamID)
		query += fmt.Sprintf(" AND s.entity_id = $%d", len(args))
	}
	if filter.CoachID > 0 {
		args = append(args, filter.CoachID)
		query += fmt.Sprintf(" AND s.coach_id = $%d", len(args))
	}
	if filter.StreakType != "" {
		args = append(args, filter.StreakType)
		query += fmt.Sprintf(" AND s.streak_type = $%d", len(args))
	}
	if filter.Scope != "" {
		args = append(args, filter.Scope)
		query += fmt.Sprintf(" AND s.scope = $%d", len(args))
	}
	if filter.MinLength > 0 {
		args = append(args, filter.MinLength)
		query += fmt.Sprintf(" AND s.length >= $%d", len(args))
	}

	query += " ORDER BY COALESCE((s.metrics->>'interest_score')::numeric, 0) DESC, s.length DESC, s.end_date DESC LIMIT " + fmt.Sprintf("%d", limit)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []StreakRecord
	for rows.Next() {
		var rec StreakRecord
		var historyJSON, metricsJSON []byte
		if err := rows.Scan(
			&rec.ID, &rec.LeagueID, &rec.LeagueName,
			&rec.EntityType, &rec.EntityID, &rec.EntityName, &rec.EntityLogo,
			&rec.SecondaryEntityID, &rec.SecondaryEntityName, &rec.SecondaryEntityLogo,
			&rec.CoachID, &rec.CoachName, &rec.CoachImage,
			&rec.StreakType, &rec.StreakTypeName, &rec.StreakTypeLongName, &rec.StreakTypeIcon,
			&rec.Scope, &rec.ScopeName,
			&rec.Length, &rec.IsActive, &rec.IsRecordClub, &rec.IsRecordLeague,
			&rec.StartFixtureID, &rec.StartSeasonID, &rec.StartSeasonName, &rec.StartRoundID, &rec.StartDate,
			&rec.EndFixtureID, &rec.EndSeasonID, &rec.EndSeasonName, &rec.EndRoundID, &rec.EndDate,
			&rec.BreakingFixtureID, &historyJSON, &metricsJSON,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal(historyJSON, &rec.FixturesHistory)
		_ = json.Unmarshal(metricsJSON, &rec.Metrics)
		records = append(records, rec)
	}

	return records, nil
}
