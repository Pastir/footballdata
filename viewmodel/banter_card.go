package viewmodel

import (
	"fmt"
	"strings"

	"github.com/Pastir/footballdata/analytics"
)

// BanterStatBar описывает полосу сравнения метрик (xG, удары, владение, сейвы)
type BanterStatBar struct {
	Label  string
	ValA   string
	ValB   string
	PctA   int
	ColorA string
	ColorB string
}

// BanterCardViewModel подготавливает данные для рендеринга 1200x1200px постера аномалии
type BanterCardViewModel struct {
	BadgeText      string
	BadgeColor     string // "amber", "cyan", "rose"
	BadgeClass     string
	LeagueBadge    string
	TagText        string
	AnomalySummary string

	// Победитель матча (для динамической подсветки)
	TeamAIsWinner bool
	TeamBIsWinner bool

	// Карточка игрока (Джокер / Киборг / Вратарь)
	IsHeroPlayer    bool
	PlayerName      string
	PlayerTeam      string
	PlayerAvatar    string
	PlayerStatBig   string
	PlayerStatSmall string

	// Командный матч
	TeamAName  string
	TeamALogo  string
	TeamAScore int
	TeamAXG    string
	TeamAPoss  string

	TeamBName  string
	TeamBLogo  string
	TeamBScore int
	TeamBXG    string
	TeamBPoss  string

	// Статистические бары
	Stats []BanterStatBar

	// Подвал карточки
	FooterIcon  string
	FooterQuote string
	Watermark   string
	SiteURL     string
}

// BuildBanterCardViewModel преобразует аналитическую аномалию в готовую ViewModel
func BuildBanterCardViewModel(a analytics.MatchAnomaly) BanterCardViewModel {
	leagueText := strings.ToUpper(a.LeagueName)
	if a.RoundName != "" {
		leagueText = fmt.Sprintf("%s • %s", leagueText, strings.ToUpper(a.RoundName))
	}

	hXGStr := "-"
	if a.HomeXG != nil {
		hXGStr = fmt.Sprintf("%.2f", *a.HomeXG)
	}
	aXGStr := "-"
	if a.AwayXG != nil {
		aXGStr = fmt.Sprintf("%.2f", *a.AwayXG)
	}

	badgeColor := "rose"
	if a.BadgeText == "Камбэк тура" || a.BadgeText == "Сенсация тура" {
		badgeColor = "amber"
	} else if a.BadgeText == "Спасение 90+" {
		badgeColor = "cyan"
	}

	vm := BanterCardViewModel{
		BadgeText:       a.BadgeText,
		BadgeColor:      badgeColor,
		BadgeClass:      a.BadgeClass,
		LeagueBadge:     leagueText,
		TagText:         a.TagText,
		AnomalySummary:  a.AnomalySummary,
		TeamAIsWinner:   a.ScoreHome > a.ScoreAway,
		TeamBIsWinner:   a.ScoreAway > a.ScoreHome,
		IsHeroPlayer:    a.IsHeroPlayer,
		PlayerName:      a.HeroName,
		PlayerTeam:      a.HeroTeam,
		PlayerAvatar:    a.HeroAvatar,
		PlayerStatBig:   a.HeroStatBig,
		PlayerStatSmall: a.HeroStatSmall,
		TeamAName:       a.TeamHomeName,
		TeamALogo:       a.TeamHomeLogo,
		TeamAScore:      a.ScoreHome,
		TeamAXG:         hXGStr,
		TeamAPoss:       fmt.Sprintf("%d%%", a.HomePossession),
		TeamBName:       a.TeamAwayName,
		TeamBLogo:       a.TeamAwayLogo,
		TeamBScore:      a.ScoreAway,
		TeamBXG:         aXGStr,
		TeamBPoss:       fmt.Sprintf("%d%%", a.AwayPossession),
		FooterIcon:      a.FooterIcon,
		FooterQuote:     a.FooterQuote,
		Watermark:       "FOOTBALL DATA LAB",
		SiteURL:         "data-lab.live",
	}

	if vm.FooterIcon == "" {
		vm.FooterIcon = "⚖️"
	}

	// Формируем полосы сравнения статистики
	var stats []BanterStatBar

	// 1. Ожидаемые голы (xG)
	if a.HomeXG != nil && a.AwayXG != nil {
		totalXG := *a.HomeXG + *a.AwayXG
		pctA := 50
		if totalXG > 0 {
			pctA = int((*a.HomeXG / totalXG) * 100)
		}
		stats = append(stats, BanterStatBar{
			Label:  "Ожидаемые голы (xG)",
			ValA:   fmt.Sprintf("%.2f", *a.HomeXG),
			ValB:   fmt.Sprintf("%.2f", *a.AwayXG),
			PctA:   pctA,
			ColorA: "bg-sky-400",
			ColorB: "bg-rose-500",
		})
	}

	// 2. Удары по воротам
	if a.HomeShots > 0 || a.AwayShots > 0 {
		totalShots := a.HomeShots + a.AwayShots
		pctA := 50
		if totalShots > 0 {
			pctA = int((float64(a.HomeShots) / float64(totalShots)) * 100)
		}
		stats = append(stats, BanterStatBar{
			Label:  "Удары по воротам",
			ValA:   fmt.Sprintf("%d", a.HomeShots),
			ValB:   fmt.Sprintf("%d", a.AwayShots),
			PctA:   pctA,
			ColorA: "bg-sky-400",
			ColorB: "bg-rose-500",
		})
	}

	// 3. Владение мячом
	if a.HomePossession > 0 || a.AwayPossession > 0 {
		pctA := a.HomePossession
		if pctA <= 0 && a.AwayPossession > 0 {
			pctA = 100 - a.AwayPossession
		}
		stats = append(stats, BanterStatBar{
			Label:  "Владение мячом",
			ValA:   fmt.Sprintf("%d%%", a.HomePossession),
			ValB:   fmt.Sprintf("%d%%", a.AwayPossession),
			PctA:   pctA,
			ColorA: "bg-sky-400",
			ColorB: "bg-amber-400",
		})
	}

	// 4. Сейвы вратаря
	if a.HomeSaves > 0 || a.AwaySaves > 0 {
		totalSaves := a.HomeSaves + a.AwaySaves
		pctA := 50
		if totalSaves > 0 {
			pctA = int((float64(a.HomeSaves) / float64(totalSaves)) * 100)
		}
		stats = append(stats, BanterStatBar{
			Label:  "Сейвы голкиперов",
			ValA:   fmt.Sprintf("%d", a.HomeSaves),
			ValB:   fmt.Sprintf("%d", a.AwaySaves),
			PctA:   pctA,
			ColorA: "bg-cyan-400",
			ColorB: "bg-emerald-400",
		})
	}

	vm.Stats = stats
	return vm
}

