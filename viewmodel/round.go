package viewmodel

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Pastir/footballdata/analytics"
)

// MatchScoreItem — отдельный матч с результатом для прошедшего тура.
type MatchScoreItem struct {
	ID           int64
	HomeTeamName string
	HomeTeamLogo string
	HomeScore    int
	HomeHTScore  int
	HomeRedCards int      // кол-во красных карточек у хозяев (0 если нет)
	HomeScorers  []string // e.g. ["Сака 24'", "Хаверц 68'"]

	AwayTeamName string
	AwayTeamLogo string
	AwayScore    int
	AwayHTScore  int
	AwayRedCards int      // кол-во красных карточек у гостей (0 если нет)
	AwayScorers  []string // e.g. ["Сон 82'"]

	Status     string // "FT", "AET", "PEN"
	DateStr    string // "Суббота, 15 марта • 15:30"
	Venue      string // "Эмирейтс Стэдиум"
	XGHome     string // "2.15" (если доступно)
	XGAway     string // "0.85" (если доступно)
	IsFeatured bool   // Матч тура / центральный

	// Котировки закрытия (Closing Odds)
	BookmakerName    string // "Pinnacle"
	ClosingOddsHome  string // "1.85"
	ClosingOddsDraw  string // "3.70"
	ClosingOddsAway  string // "4.20"
	WinningOutcome   string // "1", "X", "2"
	WinningOdds      string // "1.85"
	IsUpset          bool   // Сенсация тура (кэф >= 3.50)
	UpsetText        string // "🔥 Сенсация: кэф 4.20"
}

// HasOdds возвращает true, если у завершённого матча заполнены котировки закрытия.
func (m MatchScoreItem) HasOdds() bool {
	return m.ClosingOddsHome != "" && m.ClosingOddsHome != "0" && m.ClosingOddsHome != "0.00" && m.ClosingOddsHome != "-" &&
		m.ClosingOddsAway != "" && m.ClosingOddsAway != "0" && m.ClosingOddsAway != "0.00" && m.ClosingOddsAway != "-"
}

// RoundDayResultGroup — матчи прошедшего тура, сгруппированные по дням или колонкам.
type RoundDayResultGroup struct {
	DayLabel string // "СУББОТА, 15 МАРТА"
	Matches  []MatchScoreItem
}

// RoundStatsSummary — сводная статистика прошедшего тура.
type RoundStatsSummary struct {
	TotalMatches int
	TotalGoals   int
	AvgGoals     string // "2.9"
	HomeWins     int
	Draws        int
	AwayWins     int
	TotalReds    int
	TopMatch     string // "Манчестер Сити 2 : 2 Ливерпуль"
}

// RoundResultsViewModel — модель для рендера прошедшего тура.
type RoundResultsViewModel struct {
	LeagueID         int
	LeagueName       string
	LeagueLogo       string
	Season           string
	RoundTitle       string // "ТУР 28"
	DateRange        string // "15 – 17 МАРТА 2025"
	Timezone         string // "MSK", "UTC", "CET" (таймзона для отображения)
	BookmakerName    string // e.g. "Pinnacle"
	ShowClosingOdds  bool   // показывать ли предматчевые кэфы закрытия
	Stats            RoundStatsSummary
	DayGroups        []RoundDayResultGroup
	LeftCol          []MatchScoreItem // Для раскладки в 2 колонки
	RightCol         []MatchScoreItem
	UpdatedAt        string
}

// FixtureMatchItem — отдельный матч для анонса следующего тура.
type FixtureMatchItem struct {
	ID           int64
	HomeTeamID   int64
	HomeTeamName string
	HomeTeamLogo string
	HomeRank     int      // Позиция в таблице, e.g. 1
	HomePoints   int      // Очки в таблице
	HomePlayed   int      // Сыграно матчей
	HomeForm     []string // ["W", "W", "W", "D", "W"]

	AwayTeamID   int64
	AwayTeamName string
	AwayTeamLogo string
	AwayRank     int      // Позиция в таблице, e.g. 4
	AwayPoints   int      // Очки в таблице
	AwayPlayed   int      // Сыграно матчей
	AwayForm     []string // ["W", "L", "W", "W", "D"]

	KickoffTime      string    // "15:30"
	KickoffDate      string    // "Сб, 10 октября"
	KickoffTimestamp time.Time // Точное время начала матча
	Timezone         string    // "MSK", "UTC", "CET" (опционально для матча)
	Venue         string   // "Эмирейтс"
	BookmakerName string   // "Pinnacle" или "Winline"
	OddsHome      string   // "1.85"
	OddsDraw      string   // "3.60"
	OddsAway      string   // "4.20"
	OddsOver25    string   // "1.75"
	OddsUnder25   string   // "2.10"
	ProbHome      string   // "52%"
	ProbDraw      string   // "26%"
	ProbAway      string   // "22%"
	IsFeatured    bool     // Матч недели
	IsDerby       bool     // Принципиальное дерби / соперничество
}

// HasOdds возвращает true, если у предстоящего матча заполнены основные котировки 1X2.
func (f FixtureMatchItem) HasOdds() bool {
	return f.OddsHome != "" && f.OddsHome != "0" && f.OddsHome != "0.00" && f.OddsHome != "-" &&
		f.OddsAway != "" && f.OddsAway != "0" && f.OddsAway != "0.00" && f.OddsAway != "-"
}

// HasTotals возвращает true, если заполнены котировки тоталов.
func (f FixtureMatchItem) HasTotals() bool {
	return f.OddsOver25 != "" && f.OddsOver25 != "0" && f.OddsOver25 != "0.00" && f.OddsOver25 != "-"
}

// HasProbabilities возвращает true, если заполнены процентные вероятности модели.
func (f FixtureMatchItem) HasProbabilities() bool {
	return f.ProbHome != "" && f.ProbHome != "0" && f.ProbHome != "-"
}

// FixtureDayGroup — матчи предстоящего тура, сгруппированные по дням.
type FixtureDayGroup struct {
	DayLabel string // "СУББОТА, 22 МАРТА"
	Matches  []FixtureMatchItem
}

// MatchHeaderViewModel — модель для рендера PNG-шапки центрального матча тура.
type MatchHeaderViewModel struct {
	LeagueName   string // "АПЛ"
	RoundTitle   string // "ТУР 6"
	BadgeTitle   string // "МАТЧ ТУРА"
	HomeTeamName string // "Ливерпуль"
	HomeTeamCode string // "LIV"
	HomeTeamLogo string // URL или dataURI
	AwayTeamName string // "Манчестер Сити"
	AwayTeamCode string // "MCI"
	AwayTeamLogo string // URL или dataURI
	KickoffTime  string // "15:30"
	KickoffDate  string // "11 Октября"
	Venue        string // "Anfield"
	Timezone     string // "MSK" (если задано)
	HomeRank     int    // Место хозяев в таблице, e.g. 1
	AwayRank     int    // Место гостей в таблице, e.g. 2
	OddsHome     string // "1.53"
	OddsDraw     string // "5.25"
	OddsAway     string // "4.75"
	ProbHome     string // "62%"
	ProbDraw     string // "18%"
	ProbAway     string // "20%"
}

// RoundFixturesViewModel — модель для рендера расписания следующего тура.
type RoundFixturesViewModel struct {
	LeagueID      int
	LeagueName    string
	LeagueLogo    string
	Season        string
	RoundTitle    string // "ТУР 29"
	DateRange     string // "22 – 24 МАРТА 2025"
	Timezone      string // "MSK", "UTC", "CET" (таймзона для отображения времени)
	BookmakerName string // "Pinnacle" / "Winline"
	FeaturedMatch *FixtureMatchItem
	DayGroups     []FixtureDayGroup
	LeftCol       []FixtureMatchItem // Сетка в 2 колонки
	RightCol      []FixtureMatchItem
	UpdatedAt     string
}

// BuildRoundFixturesViewModel трансформирует данные аналитики в модель отображения расписания тура.
// Если передан опциональный featuredFixtureID > 0, именно этот матч выбирается в качестве FeaturedMatch.
// Иначе используется эвристика наименьшей суммы мест соперников в таблице.
func BuildRoundFixturesViewModel(
	leagueInfo analytics.LeagueSeasonInfo,
	matches []analytics.NextRoundMatch,
	timezone string,
	featuredFixtureID ...int64,
) RoundFixturesViewModel {
	if timezone == "" {
		timezone = "UTC"
	}

	vm := RoundFixturesViewModel{
		LeagueID:      leagueInfo.LeagueID,
		LeagueName:    leagueInfo.LeagueName,
		LeagueLogo:    leagueInfo.LeagueLogo,
		Season:        leagueInfo.SeasonName,
		Timezone:      timezone,
		BookmakerName: "BET365",
		UpdatedAt:     fmt.Sprintf("%s UTC", time.Now().UTC().Format("02 Jan 2006, 15:04")),
	}

	if len(matches) == 0 {
		vm.RoundTitle = "СЛЕДУЮЩИЙ ТУР"
		vm.DateRange = "СКОРО"
		return vm
	}

	first := matches[0]
	vm.RoundTitle = fmt.Sprintf("ТУР %s", first.RoundName)
	vm.DateRange = fmt.Sprintf("%s – %s",
		formatRussianDate(first.RoundStart),
		formatRussianDate(first.RoundEnd),
	)

	// Находим центральный матч тура:
	// Сначала проверяем, задан ли явно ID матча тура (например, рассчитанный analytics.GetOrComputeMatchOfRound)
	featuredIdx := 0
	foundExplicitFeatured := false
	if len(featuredFixtureID) > 0 && featuredFixtureID[0] > 0 {
		for i, m := range matches {
			if m.FixtureID == featuredFixtureID[0] {
				featuredIdx = i
				foundExplicitFeatured = true
				break
			}
		}
	}

	// Если явный матч не передан или не найден среди матчей тура — берем наименьшую сумму мест
	if !foundExplicitFeatured {
		bestRankSum := 999
		for i, m := range matches {
			rankH := 20
			if m.PositionHome != nil && *m.PositionHome > 0 {
				rankH = *m.PositionHome
			}
			rankA := 20
			if m.PositionAway != nil && *m.PositionAway > 0 {
				rankA = *m.PositionAway
			}
			sum := rankH + rankA
			if sum < bestRankSum {
				bestRankSum = sum
				featuredIdx = i
			}
		}
	}

	var leftCol []FixtureMatchItem
	var rightCol []FixtureMatchItem
	var allItems []FixtureMatchItem

	for i, m := range matches {
		venue := "Стадион"
		if m.VenueName != nil && *m.VenueName != "" {
			venue = *m.VenueName
		}

		rankH := 0
		if m.PositionHome != nil {
			rankH = *m.PositionHome
		}
		rankA := 0
		if m.PositionAway != nil {
			rankA = *m.PositionAway
		}

		bmName := m.BookmakerName
		if bmName == "" {
			bmName = "bet365"
		}

		pointsH := 0
		if m.PointsHome != nil {
			pointsH = *m.PointsHome
		}
		pointsA := 0
		if m.PointsAway != nil {
			pointsA = *m.PointsAway
		}
		playedH := 0
		if m.OverallMatchesHome != nil {
			playedH = *m.OverallMatchesHome
		}
		playedA := 0
		if m.MatchesAway != nil {
			playedA = *m.MatchesAway
		}

		item := FixtureMatchItem{
			ID:               m.FixtureID,
			HomeTeamID:       m.TeamHomeID,
			HomeTeamName:     m.TeamHomeName,
			HomeTeamLogo:     m.TeamHomeLogo,
			HomeRank:         rankH,
			HomePoints:       pointsH,
			HomePlayed:       playedH,
			HomeForm:         m.FormHome,
			AwayTeamID:       m.TeamAwayID,
			AwayTeamName:     m.TeamAwayName,
			AwayTeamLogo:     m.TeamAwayLogo,
			AwayRank:         rankA,
			AwayPoints:       pointsA,
			AwayPlayed:       playedA,
			AwayForm:         m.FormAway,
			KickoffTime:      m.FixtureStart.Format("15:04"),
			KickoffDate:      formatDayHeader(m.FixtureStart),
			KickoffTimestamp: m.FixtureStart,
			Venue:            venue,
			BookmakerName:    strings.ToUpper(bmName),
			OddsHome:         m.OddsHome,
			OddsDraw:         m.OddsDraw,
			OddsAway:         m.OddsAway,
			IsDerby:          m.IsRival,
		}

		// Если заданы котировки 1X2, рассчитываем подразумеваемые вероятности модели (в %)
		if m.OddsHome != "" && m.OddsDraw != "" && m.OddsAway != "" {
			if oH, err1 := strconv.ParseFloat(m.OddsHome, 64); err1 == nil && oH > 1.0 {
				if oD, err2 := strconv.ParseFloat(m.OddsDraw, 64); err2 == nil && oD > 1.0 {
					if oA, err3 := strconv.ParseFloat(m.OddsAway, 64); err3 == nil && oA > 1.0 {
						invH := 1.0 / oH
						invD := 1.0 / oD
						invA := 1.0 / oA
						margin := invH + invD + invA
						if margin > 0 {
							item.ProbHome = fmt.Sprintf("%.0f%%", math.Round((invH/margin)*100))
							item.ProbDraw = fmt.Sprintf("%.0f%%", math.Round((invD/margin)*100))
							item.ProbAway = fmt.Sprintf("%.0f%%", math.Round((invA/margin)*100))
						}
					}
				}
			}
		}

		if i == featuredIdx {
			item.IsFeatured = true
			vm.FeaturedMatch = &item
		} else {
			if len(leftCol) <= len(rightCol) {
				leftCol = append(leftCol, item)
			} else {
				rightCol = append(rightCol, item)
			}
		}

		allItems = append(allItems, item)
	}

	vm.LeftCol = leftCol
	vm.RightCol = rightCol
	vm.DayGroups = groupFixtureItemsByDay(allItems)
	return vm
}

func groupFixtureItemsByDay(items []FixtureMatchItem) []FixtureDayGroup {
	daysRu := []string{"Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота"}
	monthsRu := []string{
		"", "Янв.", "Фев.", "Мар.", "Апр.", "Мая", "Июн.",
		"Июл.", "Авг.", "Сен.", "Окт.", "Ноя.", "Дек.",
	}

	groupsMap := make(map[string]*FixtureDayGroup)
	var keys []string

	for _, item := range items {
		d := item.KickoffTimestamp
		dateKey := d.Format("2006-01-02")

		if _, exists := groupsMap[dateKey]; !exists {
			mIdx := int(d.Month())
			monthStr := ""
			if mIdx >= 1 && mIdx <= 12 {
				monthStr = monthsRu[mIdx]
			}
			dayLabel := fmt.Sprintf("%d %s %d, %s", d.Day(), monthStr, d.Year(), daysRu[int(d.Weekday())])

			groupsMap[dateKey] = &FixtureDayGroup{
				DayLabel: dayLabel,
				Matches:  nil,
			}
			keys = append(keys, dateKey)
		}
		groupsMap[dateKey].Matches = append(groupsMap[dateKey].Matches, item)
	}

	sort.Strings(keys)
	var result []FixtureDayGroup
	for _, k := range keys {
		result = append(result, *groupsMap[k])
	}
	return result
}


func formatRussianDate(t time.Time) string {
	months := []string{
		"", "ЯНВАРЯ", "ФЕВРАЛЯ", "МАРТА", "АПРЕЛЯ", "МАЯ", "ИЮНЯ",
		"ИЮЛЯ", "АВГУСТА", "СЕНТЯБРЯ", "ОКТЯБРЯ", "НОЯБРЯ", "ДЕКАБРЯ",
	}
	m := int(t.Month())
	if m >= 1 && m <= 12 {
		return fmt.Sprintf("%d %s %d", t.Day(), months[m], t.Year())
	}
	return t.Format("02.01.2006")
}

func formatDayHeader(t time.Time) string {
	days := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	months := []string{
		"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}
	weekday := days[int(t.Weekday())]
	m := int(t.Month())
	if m >= 1 && m <= 12 {
		return fmt.Sprintf("%s, %d %s", weekday, t.Day(), months[m])
	}
	return t.Format("02.01.2006")
}

func formatShortRussianDate(t time.Time) string {
	days := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	months := []string{
		"", "янв", "фев", "мар", "апр", "мая", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек",
	}
	weekday := days[int(t.Weekday())]
	m := int(t.Month())
	if m >= 1 && m <= 12 {
		return fmt.Sprintf("%s, %d %s", weekday, t.Day(), months[m])
	}
	return strings.ToLower(t.Format("Mon, 02 Jan"))
}
