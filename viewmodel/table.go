package viewmodel

import (
	"fmt"
	"time"

	"github.com/Pastir/footballdata/analytics"
)

// TableViewModel — полностью подготовленные данные для renderer.
// Renderer не выполняет никаких расчётов.
type TableViewModel struct {
	LeagueID   int
	LeagueName string
	LeagueLogo string
	Season     string
	Teams      []TeamRow
	Insight    string
	Stats      []analytics.StatCard
	UpdatedAt  string
}

// TeamRow — строка таблицы, готовая для отображения.
type TeamRow struct {
	Position int
	Name     string
	Logo     string
	Played   int
	Won      int
	Drawn    int
	Lost     int
	GF       int
	GA       int
	GD       int
	Points   int
	Form     []string // ["W","L","D","W","W"] — от старого к новому
	Trend    int      // > 0 рост, < 0 падение, 0 без изменений
	Zone     string   // "ucl" / "uel" / "uecl" / "relegation" / ""
}

// BuildTableViewModel собирает ViewModel из данных analytics.
func BuildTableViewModel(
	info analytics.LeagueSeasonInfo,
	rows []analytics.Standing,
	details map[int64]analytics.StandingDetails,
	forms map[int64][]string,
	trends map[int64]int, // teamID → trend
	insight string,
	stats []analytics.StatCard,
) TableViewModel {
	teams := make([]TeamRow, 0, len(rows))

	for _, r := range rows {
		played := r.Played
		won := r.Won
		drawn := r.Drawn
		lost := r.Lost
		gf := r.GF
		ga := r.GA
		gd := r.GD
		if d, ok := details[r.ID]; ok {
			played = d.Played
			won = d.Won
			drawn = d.Drawn
			lost = d.Lost
			gf = d.GF
			ga = d.GA
			gd = d.GD
		}

		form := r.Form
		if f, ok := forms[r.ID]; ok && len(f) > 0 {
			form = f
		}

		trend := r.Trend
		if t, ok := trends[r.TeamID]; ok {
			trend = t
		}

		zone := analytics.ZoneFromDeveloperName(r.RuleDevName)
		if zone == "" && r.Rule != "" {
			zone = analytics.ZoneFromDeveloperName(r.Rule)
		}

		teams = append(teams, TeamRow{
			Position: r.Position,
			Name:     r.TeamName,
			Logo:     r.TeamLogoURL,
			Played:   played,
			Won:      won,
			Drawn:    drawn,
			Lost:     lost,
			GF:       gf,
			GA:       ga,
			GD:       gd,
			Points:   r.Points,
			Form:     form,
			Trend:    trend,
			Zone:     zone,
		})
	}

	return TableViewModel{
		LeagueID:   info.LeagueID,
		LeagueName: info.LeagueName,
		LeagueLogo: info.LeagueLogo,
		Season:     info.SeasonName,
		Teams:      teams,
		Insight:    insight,
		Stats:      stats,
		UpdatedAt:  fmt.Sprintf("%s UTC", time.Now().UTC().Format("02 Jan 2006, 15:04")),
	}
}

// Top5TableViewModel — подготовленные данные для постера «Топ-5 лидеров + Зона вылета».
type Top5TableViewModel struct {
	LeagueID        int
	LeagueName      string
	LeagueLogo      string
	Season          string
	RoundName       string
	TopTeams        []TeamRow // 1st to 5th
	RelegationTeams []TeamRow // 18th to 20th
	Insight         string
	UpdatedAt       string
}

// BuildTop5TableViewModel собирает данные для компактного постера Топ-5 лидеров и зоны вылета.
func BuildTop5TableViewModel(
	info analytics.LeagueSeasonInfo,
	roundName string,
	rows []analytics.Standing,
	insight string,
) Top5TableViewModel {
	fullVM := BuildTableViewModel(info, rows, nil, nil, nil, insight, nil)

	var topTeams []TeamRow
	if len(fullVM.Teams) >= 5 {
		topTeams = fullVM.Teams[:5]
	} else {
		topTeams = fullVM.Teams
	}

	var relegationTeams []TeamRow
	if len(fullVM.Teams) > 5 {
		startIdx := len(fullVM.Teams) - 3
		if startIdx < 5 {
			startIdx = 5
		}
		relegationTeams = fullVM.Teams[startIdx:]
	}

	return Top5TableViewModel{
		LeagueID:        info.LeagueID,
		LeagueName:      info.LeagueName,
		LeagueLogo:      info.LeagueLogo,
		Season:          info.SeasonName,
		RoundName:       roundName,
		TopTeams:        topTeams,
		RelegationTeams: relegationTeams,
		Insight:         insight,
		UpdatedAt:       fmt.Sprintf("%s UTC", time.Now().UTC().Format("02 Jan 2006, 15:04")),
	}
}

