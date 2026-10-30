package viewmodel

import (
	"fmt"
	"strings"

	"github.com/Pastir/footballdata/analytics"
)

// ExpressSlipMatchItem описывает один матч экспресса в визуальном шаблоне карточки.
type ExpressSlipMatchItem struct {
	Order        int    `json:"order"`
	KickoffDate  string `json:"kickoff_date"`
	KickoffTime  string `json:"kickoff_time"`
	HomeTeamName string `json:"home_team_name"`
	HomeTeamLogo string `json:"home_team_logo"`
	AwayTeamName string `json:"away_team_name"`
	AwayTeamLogo string `json:"away_team_logo"`
	MarketName   string `json:"market_name"`
	Selection    string `json:"selection"`
	Odds         string `json:"odds"`
	Reason       string `json:"reason"`
	Badge        string `json:"badge"`
}

// ExpressSlipViewModel содержит подготовленные данные для рендеринга визуального билета экспресса.
type ExpressSlipViewModel struct {
	LeagueName       string                 `json:"league_name"`
	RoundTitle       string                 `json:"round_title"`
	TotalOdds        string                 `json:"total_odds"`
	ValueEdge        string                 `json:"value_edge"`
	Confidence       string                 `json:"confidence"`
	RecommendedStake string                 `json:"recommended_stake"`
	Matches          []ExpressSlipMatchItem `json:"matches"`
	DateRange        string                 `json:"date_range"`
	BookmakerPromo   string                 `json:"bookmaker_promo"`
	SlipID           string                 `json:"slip_id"`
}

// BuildExpressSlipViewModel преобразует аналитический план в ViewModel постера.
func BuildExpressSlipViewModel(leagueName, roundName string, plan *analytics.SmartExpressPlan, promoText string) ExpressSlipViewModel {
	var items []ExpressSlipMatchItem

	for i, sel := range plan.Selections {
		m := sel.Match
		badge := "TOP PICK"
		if i == 0 {
			badge = "МАТЕМАТИЧЕСКИЙ ПЕРЕВЕС"
		} else if strings.Contains(strings.ToLower(sel.Reason), "дерби") {
			badge = "ДЕРБИ"
		} else if strings.Contains(strings.ToLower(sel.MarketName), "обе забьют") {
			badge = "АТАКУЮЩИЙ ТЕМП"
		}

		timeStr := m.StartingAt.UTC().Format("15:04 UTC")
		dateStr := m.StartingAt.UTC().Format("02.01")

		items = append(items, ExpressSlipMatchItem{
			Order:        i + 1,
			KickoffDate:  dateStr,
			KickoffTime:  timeStr,
			HomeTeamName: m.HomeDisplayName(),
			HomeTeamLogo: m.HomeTeamLogo,
			AwayTeamName: m.AwayDisplayName(),
			AwayTeamLogo: m.AwayTeamLogo,
			MarketName:   sel.MarketName,
			Selection:    sel.OutcomeShortName(),
			Odds:         fmt.Sprintf("%.2f", sel.Odds),
			Reason:       sel.Reason,
			Badge:        badge,
		})
	}

	roundTitle := roundName
	if roundTitle != "" && !strings.Contains(strings.ToLower(roundTitle), "тур") {
		roundTitle = fmt.Sprintf("ТУР %s", roundTitle)
	}

	return ExpressSlipViewModel{
		LeagueName:       strings.ToUpper(leagueName),
		RoundTitle:       strings.ToUpper(roundTitle),
		TotalOdds:        fmt.Sprintf("%.2f", plan.TotalOdds),
		ValueEdge:        fmt.Sprintf("+%.1f%%", plan.ValueEdge),
		Confidence:       fmt.Sprintf("%.1f / 10", plan.Confidence),
		RecommendedStake: fmt.Sprintf("%.1f юнит", plan.RecommendedStake),
		Matches:          items,
		BookmakerPromo:   promoText,
		SlipID:           fmt.Sprintf("#FDL-EXP-%s", strings.TrimPrefix(roundName, "ТУР ")),
	}
}
