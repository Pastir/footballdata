package analytics

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

// LeagueSeasonInfo — данные лиги, текущего сезона и контекста туров.
// Заполняется одним SQL-запросом; передаётся во все analytics-функции
// вместо россыпи leagueID/seasonID/roundID параметров.
type LeagueSeasonInfo struct {
	// Лига
	LeagueID   int
	LeagueName string
	LeagueLogo string

	// Сезон
	SeasonID   int
	SeasonName string

	// Последний завершённый тур (для standings / trend)
	CurrRoundID   int
	CurrRoundName string

	// Предыдущий завершённый тур (для Trend); 0 = нет
	PrevRoundID int

	// Ближайший незавершённый тур (для fixtures / match-of-round)
	NextRoundID   int
	NextRoundName string

	// Контекст лиги — нужен для алгоритма MatchOfRound
	TeamCount        int // количество команд в лиге (например, 20 для PL)
	TotalTeamMatches int // всего матчей в сезоне на команду (например, 38 для PL)

	// Локаль для названий команд (например, "en", "ru", "ua")
	Locale string
}

// GetLeagueSeasonInfo возвращает полный контекст лиги и сезона одним запросом:
// данные лиги, сезона, curr/prev/next round ID и лиговый контекст для MatchOfRound.
func GetLeagueSeasonInfo(ctx context.Context, pool *pgxpool.Pool, leagueID int) (LeagueSeasonInfo, error) {
	query := `
WITH season AS (
    SELECT pl.id                         AS league_id,
           pl.name                       AS league_name,
           COALESCE(pl.image_path, '')   AS league_logo,
           ps.id                         AS season_id,
           ps.name                       AS season_name
    FROM provider_leagues pl
    JOIN provider_seasons ps ON ps.league_id = pl.id
    WHERE pl.id      = $1
      AND ps.is_current = true
    LIMIT 1
),
round_count AS (
    SELECT MAX(pr.name::INT) AS total_matches
    FROM provider_rounds pr
    JOIN season s ON pr.season_id = s.season_id
),
team_count AS (
    SELECT COUNT(DISTINCT ps2.participant_id) AS cnt
    FROM provider_standings ps2
    JOIN season s ON true
    JOIN LATERAL get_neighbor_rounds(s.season_id) nr ON ps2.round_id = nr.last_round_id
)
SELECT s.league_id,
       s.league_name,
       s.league_logo,
       s.season_id,
       s.season_name,
       COALESCE(nr.last_round_id, 0)   AS curr_round_id,
       COALESCE(nr.last_round_name, '') AS curr_round_name,
       COALESCE(nr.prev_round_id, 0)   AS prev_round_id,
       COALESCE(nr.next_round_id, 0)   AS next_round_id,
       COALESCE(nr.next_round_name, '') AS next_round_name,
       COALESCE((SELECT total_matches FROM round_count), 0)   AS total_team_matches,
       COALESCE((SELECT cnt FROM team_count), 0)              AS team_count
FROM season s
LEFT JOIN LATERAL get_neighbor_rounds(s.season_id) nr ON true`

	var info LeagueSeasonInfo
	row := pool.QueryRow(ctx, query, leagueID)
	if err := row.Scan(
		&info.LeagueID, &info.LeagueName, &info.LeagueLogo,
		&info.SeasonID, &info.SeasonName,
		&info.CurrRoundID, &info.CurrRoundName,
		&info.PrevRoundID,
		&info.NextRoundID, &info.NextRoundName,
		&info.TotalTeamMatches,
		&info.TeamCount,
	); err != nil {
		return LeagueSeasonInfo{}, fmt.Errorf("GetLeagueSeasonInfo: %w", err)
	}
	return info, nil
}

// ActiveLeagueItem описывает одну лигу для селектора и каталога.
type ActiveLeagueItem struct {
	LeagueID     int    `json:"league_id"`
	LeagueName   string `json:"league_name"`
	ShortName    string `json:"short_name"`
	DisplayName  string `json:"display_name"`
	CountryID    int    `json:"country_id"`
	CountryName  string `json:"country_name"`
	CountryFlag  string `json:"country_flag"`
	LeagueLogo   string `json:"league_logo"`
	SeasonID     int    `json:"season_id"`
	SeasonName   string `json:"season_name"`
	CurrRoundID  int    `json:"curr_round_id"`
	CurrRound    string `json:"curr_round"`
	NextRoundID  int    `json:"next_round_id"`
	NextRound    string `json:"next_round"`
	Category     string `json:"category"`      // "eurocups", "domestic"
	CategoryName string `json:"category_name"` // "Еврокубки", "Европейские чемпионаты"
	IsTop        bool              `json:"is_top"`
	Active       bool              `json:"active"`
	Order        int               `json:"order"`
	Rounds       []LeagueRoundItem `json:"rounds,omitempty"`
}

// LeagueRoundItem представляет метаданные тура сезона для навигатора туров.
type LeagueRoundItem struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	StartingAt    string `json:"starting_at,omitempty"`
	EndingAt      string `json:"ending_at,omitempty"`
	Finished      bool   `json:"finished"`
	FinishedCount int    `json:"finished_count"`
	TotalMatches  int    `json:"total_matches"`
}

// LeagueCategory описывает раздел турниров в селекторе.
type LeagueCategory struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	Count int    `json:"count"`
}

// LeagueCatalogResponse представляет структуру active_leagues.json.
type LeagueCatalogResponse struct {
	UpdatedAt  string             `json:"updated_at"`
	Total      int                `json:"total"`
	Categories []LeagueCategory   `json:"categories"`
	Leagues    []ActiveLeagueItem `json:"leagues"`
}

// GetActiveLeagues извлекает каталог турниров с активным флагом (pl.active = true) и активными сезонами (ps.is_current = true).
func GetActiveLeagues(ctx context.Context, pool *pgxpool.Pool) (LeagueCatalogResponse, error) {
	query := `
SELECT 
    pl.id,
    pl.name,
    COALESCE(pl.short_code, ''),
    COALESCE(pl.image_path, ''),
    COALESCE(pl.sub_type, ''),
    COALESCE(pl.type, ''),
    COALESCE(pl.country_id, 0),
    COALESCE(pc.name, ''),
    COALESCE(pc.image_path, ''),
    ps.id AS season_id,
    ps.name AS season_name,
    COALESCE(nr.last_round_id, 0) AS curr_round_id,
    COALESCE(nr.last_round_name, '') AS curr_round_name,
    COALESCE(nr.next_round_id, 0) AS next_round_id,
    COALESCE(nr.next_round_name, '') AS next_round_name,
    pl.active
FROM provider_leagues pl
JOIN provider_seasons ps ON ps.league_id = pl.id AND ps.is_current = true
LEFT JOIN provider_countries pc ON pc.id = pl.country_id
LEFT JOIN LATERAL get_neighbor_rounds(ps.id) nr ON true
WHERE pl.active = true
ORDER BY pl.id;`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return LeagueCatalogResponse{}, fmt.Errorf("GetActiveLeagues query: %w", err)
	}
	defer rows.Close()

	var leagues []ActiveLeagueItem
	catCounts := make(map[string]int)

	for rows.Next() {
		var (
			lid, cid, sid, currRid, nextRid int
			lname, shortCode, llogo, subType, ltype, cname, cflag, sname, currRname, nextRname string
			active bool
		)

		if err := rows.Scan(
			&lid, &lname, &shortCode, &llogo, &subType, &ltype, &cid,
			&cname, &cflag, &sid, &sname, &currRid, &currRname, &nextRid, &nextRname, &active,
		); err != nil {
			return LeagueCatalogResponse{}, fmt.Errorf("scan active league: %w", err)
		}

		item := resolveLeagueMetadata(lid, lname, shortCode, llogo, cid, cname, cflag, sid, sname, currRid, currRname, nextRid, nextRname, subType, ltype, active)
		leagues = append(leagues, item)
		catCounts[item.Category]++
	}

	// Подгружаем полный список туров текущих сезонов для навигатора ‹ Тур ›
	seasonIDs := make([]int, len(leagues))
	for i, l := range leagues {
		seasonIDs[i] = l.SeasonID
	}
	if len(seasonIDs) > 0 {
		roundsQuery := `
SELECT 
    pr.season_id, 
    pr.id, 
    pr.name, 
    COALESCE(pr.starting_at::text, ''), 
    COALESCE(pr.ending_at::text, ''),
    COALESCE(pr.finished, false),
    COALESCE(COUNT(pf.id) FILTER (WHERE pf.state_id = 5), 0),
    COALESCE(COUNT(pf.id), 0)
FROM provider_rounds pr
LEFT JOIN provider_fixtures pf ON pf.round_id = pr.id
WHERE pr.season_id = ANY($1)
GROUP BY pr.season_id, pr.id, pr.name, pr.starting_at, pr.ending_at, pr.finished
ORDER BY pr.season_id, pr.starting_at ASC, pr.id ASC;`
		rRows, rErr := pool.Query(ctx, roundsQuery, seasonIDs)
		if rErr == nil {
			defer rRows.Close()
			roundsMap := make(map[int][]LeagueRoundItem)
			for rRows.Next() {
				var sid, rid, finishedCount, totalMatches int
				var rname, startAt, endAt string
				var finished bool
				if scanErr := rRows.Scan(&sid, &rid, &rname, &startAt, &endAt, &finished, &finishedCount, &totalMatches); scanErr == nil {
					roundsMap[sid] = append(roundsMap[sid], LeagueRoundItem{
						ID:            rid,
						Name:          rname,
						StartingAt:    startAt,
						EndingAt:      endAt,
						Finished:      finished,
						FinishedCount: finishedCount,
						TotalMatches:  totalMatches,
					})
				}
			}
			for i := range leagues {
				if rList, ok := roundsMap[leagues[i].SeasonID]; ok {
					leagues[i].Rounds = rList
				}
			}
		}
	}

	// Формируем список категорий только если в них есть хотя бы одна лига
	allCats := []LeagueCategory{
		{Key: "domestic", Title: "Европейские чемпионаты", Icon: "🌍"},
		{Key: "eurocups", Title: "Еврокубки", Icon: "⭐"},
	}

	var activeCategories []LeagueCategory
	for _, c := range allCats {
		cnt := catCounts[c.Key]
		if cnt > 0 {
			c.Count = cnt
			activeCategories = append(activeCategories, c)
		}
	}

	resp := LeagueCatalogResponse{
		UpdatedAt:  "",
		Total:      len(leagues),
		Categories: activeCategories,
		Leagues:    leagues,
	}

	return resp, nil
}

// GetLeagueNameAndFlag возвращает краткое название турнира и эмодзи флага.
func GetLeagueNameAndFlag(lid int) (string, string) {
	item := resolveLeagueMetadata(lid, "", "", "", 0, "", "", 0, "", 0, "", 0, "", "", "", true)
	return item.DisplayName, item.CountryFlag
}

// resolveLeagueMetadata сопоставляет турниру русскоязычное имя, флаг и категорию.
func resolveLeagueMetadata(
	lid int, lname, shortCode, llogo string, cid int, cname, cflag string,
	sid int, sname string, currRid int, currRname string, nextRid int, nextRname string,
	subType, ltype string, active bool,
) ActiveLeagueItem {
	var (
		displayName = lname
		catKey      = "domestic"
		catName     = "Европейские чемпионаты"
		flagEmoji   = "⚽"
		isTop       = false
		order       = 100
	)

	switch lid {
	case 8:
		displayName = "АПЛ"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🏴󠁧󠁢󠁥󠁮󠁧󠁿"
		isTop = true
		order = 1
	case 564:
		displayName = "Ла Лига"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🇪🇸"
		isTop = true
		order = 2
	case 384:
		displayName = "Серия А"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🇮🇹"
		isTop = true
		order = 3
	case 82:
		displayName = "Бундеслига"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🇩🇪"
		isTop = true
		order = 4
	case 301:
		displayName = "Лига 1"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🇫🇷"
		isTop = true
		order = 5
	case 2:
		displayName = "Лига чемпионов"
		catKey = "eurocups"
		catName = "Еврокубки"
		flagEmoji = "🇪🇺"
		isTop = true
		order = 6
	case 5:
		displayName = "Лига Европы"
		catKey = "eurocups"
		catName = "Еврокубки"
		flagEmoji = "🇪🇺"
		isTop = true
		order = 7
	case 2286:
		displayName = "Лига конференций"
		catKey = "eurocups"
		catName = "Еврокубки"
		flagEmoji = "🇪🇺"
		isTop = true
		order = 8
	case 9:
		displayName = "Чемпионшип"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🏴󠁧󠁢󠁥󠁮󠁧󠁿"
		order = 10
	case 72:
		displayName = "Эредивизи"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🇳🇱"
		order = 11
	case 462:
		displayName = "Примейра"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🇵🇹"
		order = 12
	case 501:
		displayName = "Премьершип"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🏴󠁧󠁢󠁳󠁣󠁴󠁿"
		order = 13
	case 600:
		displayName = "Суперлига"
		catKey = "domestic"
		catName = "Европейские чемпионаты"
		flagEmoji = "🇹🇷"
		order = 14
	default:
		// Автоматическая классификация
		if subType == "cup_international" || subType == "international" || (cid == 0 && ltype == "cup") || cid == 41 {
			catKey = "eurocups"
			catName = "Еврокубки"
			flagEmoji = "⭐"
		} else {
			catKey = "domestic"
			catName = "Европейские чемпионаты"
			switch cname {
			case "England", "United Kingdom":
				flagEmoji = "🏴󠁧󠁢󠁥󠁮󠁧󠁿"
			case "Spain":
				flagEmoji = "🇪🇸"
			case "Italy":
				flagEmoji = "🇮🇹"
			case "Germany":
				flagEmoji = "🇩🇪"
			case "France":
				flagEmoji = "🇫🇷"
			case "Netherlands":
				flagEmoji = "🇳🇱"
			case "Portugal":
				flagEmoji = "🇵🇹"
			case "Scotland":
				flagEmoji = "🏴󠁧󠁢󠁳󠁣󠁴󠁿"
			case "Turkey":
				flagEmoji = "🇹🇷"
			case "Belgium":
				flagEmoji = "🇧🇪"
			case "Austria":
				flagEmoji = "🇦🇹"
			case "Croatia":
				flagEmoji = "🇭🇷"
			case "Denmark":
				flagEmoji = "🇩🇰"
			case "Switzerland":
				flagEmoji = "🇨🇭"
			case "Greece":
				flagEmoji = "🇬🇷"
			case "Poland":
				flagEmoji = "🇵🇱"
			}
		}
	}

	return ActiveLeagueItem{
		LeagueID:     lid,
		LeagueName:   lname,
		ShortName:    shortCode,
		DisplayName:  displayName,
		CountryID:    cid,
		CountryName:  cname,
		CountryFlag:  flagEmoji,
		LeagueLogo:   llogo,
		SeasonID:     sid,
		SeasonName:   sname,
		CurrRoundID:  currRid,
		CurrRound:    currRname,
		NextRoundID:  nextRid,
		NextRound:    nextRname,
		Category:     catKey,
		CategoryName: catName,
		IsTop:        isTop,
		Active:       active,
		Order:        order,
	}
}
