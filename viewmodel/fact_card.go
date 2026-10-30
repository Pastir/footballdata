package viewmodel

import (
	"fmt"
	"strings"

	"github.com/Pastir/footballdata/analytics"
)

// FactCardMatchItem представляет отдельный матч в цепочке серии на карточке
type FactCardMatchItem struct {
	DateStr      string
	Result       string // "W", "D", "L"
	ResultClass  string // "res-win", "res-draw", "res-loss"
	Score        string // "3 : 1"
	OpponentName string
	OpponentLogo string
	VenueText    string // "Д" (дома) или "Г" (в гостях)
}

// FactCardMetricItem представляет блок агрегированной статистики серии
type FactCardMetricItem struct {
	Label   string
	Value   string
	SubText string
}

// FactCardViewModel содержит полностью подготовленные данные для инфографики факта/серии
type FactCardViewModel struct {
	LeagueID            int
	LeagueName          string
	LeagueLogo          string
	CategoryBadge       string
	RecordBadge         string
	IsRecord            bool
	StreakTypeIcon      string
	HeroNumber          int
	HeroLabel           string
	HeroSubLabel        string
	EntityName          string
	EntityLogo          string
	CoachName           string
	CoachImage          string
	HasCoach            bool
	SecondaryEntityName string
	SecondaryEntityLogo string
	HasSecondary        bool
	Matches             []FactCardMatchItem
	Metrics             []FactCardMetricItem
	Watermark           string
}

// BuildFactCardViewModel преобразует аналитическую запись серии в ViewModel карточки 1200x1200px
func BuildFactCardViewModel(rec analytics.StreakRecord) FactCardViewModel {
	vm := FactCardViewModel{
		LeagueID:            rec.LeagueID,
		LeagueName:          rec.LeagueName,
		EntityName:          strings.ToUpper(rec.EntityName),
		EntityLogo:          rec.EntityLogo,
		SecondaryEntityName: strings.ToUpper(rec.SecondaryEntityName),
		SecondaryEntityLogo: rec.SecondaryEntityLogo,
		HasSecondary:        rec.SecondaryEntityID != nil && *rec.SecondaryEntityID > 0,
		StreakTypeIcon:      rec.StreakTypeIcon,
		HeroNumber:          rec.Length,
		Watermark:           "FOOTBALL DATA LAB",
	}

	if vm.StreakTypeIcon == "" {
		vm.StreakTypeIcon = "🔥"
	}

	// Бейдж турнира и области действия
	scopeLabel := rec.ScopeName
	if scopeLabel == "" {
		scopeLabel = "Чемпионат"
	}
	if rec.LeagueName != "" {
		vm.CategoryBadge = strings.ToUpper(fmt.Sprintf("%s · %s", rec.LeagueName, scopeLabel))
	} else {
		vm.CategoryBadge = strings.ToUpper(scopeLabel)
	}

	// Главный заголовок счетчика серии
	switch rec.StreakType {
	case analytics.StreakTypeTeamWin, analytics.StreakTypeCoachWin:
		vm.HeroLabel = "ПОБЕД ПОДРЯД"
		vm.HeroSubLabel = "Победная серия"
	case analytics.StreakTypeTeamUnbeaten, analytics.StreakTypeCoachUnbeaten:
		vm.HeroLabel = "МАТЧЕЙ БЕЗ ПОРАЖЕНИЙ"
		vm.HeroSubLabel = "Беспроигрышная серия"
	case analytics.StreakTypeTeamNoWin:
		vm.HeroLabel = "МАТЧЕЙ БЕЗ ПОБЕД"
		vm.HeroSubLabel = "Серия без побед"
	case analytics.StreakTypeTeamScoring:
		vm.HeroLabel = "МАТЧЕЙ С ГОЛАМИ"
		vm.HeroSubLabel = "Голевая серия"
	case analytics.StreakTypeTeamCleanSheet:
		vm.HeroLabel = "МАТЧЕЙ НА НОЛЬ"
		vm.HeroSubLabel = "Сухая серия"
	default:
		if rec.StreakTypeLongName != "" {
			vm.HeroLabel = strings.ToUpper(rec.StreakTypeLongName)
		} else {
			vm.HeroLabel = "МАТЧЕЙ В СЕРИИ"
		}
		vm.HeroSubLabel = rec.StreakTypeName
	}

	// Рекордные бейджи
	if rec.IsRecordLeague {
		vm.IsRecord = true
		vm.RecordBadge = "🏆 РЕКОРД ЛИГИ"
	} else if rec.IsRecordClub {
		vm.IsRecord = true
		vm.RecordBadge = "⭐ РЕКОРД КЛУБА"
	} else if rec.Length >= 5 {
		vm.RecordBadge = "🔥 ТОП СЕРИЯ СЕЗОНА"
	}

	// Тренер
	if rec.CoachName != "" {
		vm.CoachName = rec.CoachName
		vm.CoachImage = rec.CoachImage
		vm.HasCoach = true
	}

	// Матчи серии (показываем до 6 последних матчей в истории)
	history := rec.FixturesHistory
	maxMatches := 6
	startIndex := 0
	if len(history) > maxMatches {
		startIndex = len(history) - maxMatches
	}

	for i := startIndex; i < len(history); i++ {
		h := history[i]
		resClass := "res-draw"
		if h.Result == "W" {
			resClass = "res-win"
		} else if h.Result == "L" {
			resClass = "res-loss"
		}

		venue := "Д"
		if !h.IsHome {
			venue = "Г"
		}

		vm.Matches = append(vm.Matches, FactCardMatchItem{
			DateStr:      h.DateStr,
			Result:       h.Result,
			ResultClass:  resClass,
			Score:        fmt.Sprintf("%d : %d", h.TeamGoals, h.OpponentGoals),
			OpponentName: h.OpponentName,
			OpponentLogo: h.OpponentLogo,
			VenueText:    venue,
		})
	}

	// Метрики серии
	scored := 0
	conceded := 0
	cleanSheets := 0

	if val, ok := rec.Metrics["goals_scored"].(float64); ok {
		scored = int(val)
	} else if val, ok := rec.Metrics["goals_scored"].(int); ok {
		scored = val
	}

	if val, ok := rec.Metrics["goals_conceded"].(float64); ok {
		conceded = int(val)
	} else if val, ok := rec.Metrics["goals_conceded"].(int); ok {
		conceded = val
	}

	if val, ok := rec.Metrics["clean_sheets"].(float64); ok {
		cleanSheets = int(val)
	} else if val, ok := rec.Metrics["clean_sheets"].(int); ok {
		cleanSheets = val
	}

	avgScored := 0.0
	avgConceded := 0.0
	if rec.Length > 0 {
		avgScored = float64(scored) / float64(rec.Length)
		avgConceded = float64(conceded) / float64(rec.Length)
	}

	vm.Metrics = []FactCardMetricItem{
		{
			Label:   "Забито голов",
			Value:   fmt.Sprintf("%d", scored),
			SubText: fmt.Sprintf("%.1f в среднем", avgScored),
		},
		{
			Label:   "Пропущено",
			Value:   fmt.Sprintf("%d", conceded),
			SubText: fmt.Sprintf("%.1f в среднем", avgConceded),
		},
		{
			Label:   "Сухих матчей",
			Value:   fmt.Sprintf("%d", cleanSheets),
			SubText: fmt.Sprintf("из %d игр", rec.Length),
		},
	}

	// Если победная серия — добавляем 100% очков
	if rec.StreakType == analytics.StreakTypeTeamWin || rec.StreakType == analytics.StreakTypeCoachWin {
		maxPts := rec.Length * 3
		vm.Metrics = append(vm.Metrics, FactCardMetricItem{
			Label:   "Очки в серии",
			Value:   fmt.Sprintf("%d / %d", maxPts, maxPts),
			SubText: "100% побед",
		})
	}

	return vm
}
