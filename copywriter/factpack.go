package copywriter

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/Pastir/footballdata/viewmodel"
)

// FactPack — структурированный пакет детерминированных фактов из БД/ViewModel для передачи в LLM.
type FactPack struct {
	PostType     string                 `json:"post_type"`
	League       string                 `json:"league"`
	Season       string                 `json:"season"`
	Round        string                 `json:"round,omitempty"`
	Stats        map[string]interface{} `json:"stats,omitempty"`
	TopMatch     *TopMatchFact          `json:"top_match,omitempty"`
	KeyMatches   []KeyMatchFact         `json:"key_matches,omitempty"`
	Dramas       []string               `json:"dramas,omitempty"`
	TableContext *TableContextFact      `json:"table_context,omitempty"`
	Fixtures     []FixtureFact          `json:"fixtures,omitempty"`
}

// TopMatchFact описывает центральный матч тура.
type TopMatchFact struct {
	Fixture   string   `json:"fixture"`
	Score     string   `json:"score"`
	HTScore   string   `json:"ht_score,omitempty"`
	XG        string   `json:"xg,omitempty"`
	KeyEvents []string `json:"key_events,omitempty"`
}

// KeyMatchFact описывает примечательный результат матча.
type KeyMatchFact struct {
	HomeTeam  string   `json:"home_team"`
	AwayTeam  string   `json:"away_team"`
	Score     string   `json:"score"`
	XG        string   `json:"xg,omitempty"`
	IsUpset   bool     `json:"is_upset,omitempty"`
	UpsetText string   `json:"upset_text,omitempty"`
	Scorers   []string `json:"scorers,omitempty"`
}

// TableContextFact описывает контекст турнирной таблицы.
type TableContextFact struct {
	Leader   string `json:"leader,omitempty"`
	Pursuer  string `json:"pursuer,omitempty"`
	TopStats string `json:"top_stats,omitempty"`
}

// FixtureFact описывает предстоящий матч анонса тура.
type FixtureFact struct {
	HomeTeam string    `json:"home_team"`
	AwayTeam string    `json:"away_team"`
	Kickoff  string    `json:"kickoff"`
	Date     string    `json:"date,omitempty"`
	Time     string    `json:"time,omitempty"`
	RawTime  time.Time `json:"-"`
	IsDerby  bool      `json:"is_derby,omitempty"`
	OddsHome string    `json:"odds_home,omitempty"`
	OddsDraw string    `json:"odds_draw,omitempty"`
	OddsAway string    `json:"odds_away,omitempty"`
}

// ToJSON сериализует FactPack в форматированный JSON для LLM.
func (f FactPack) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal fact pack: %w", err)
	}
	return string(bytes), nil
}

// BuildRoundResultsFactPack формирует FactPack на основе результатов прошедшего тура.
func BuildRoundResultsFactPack(vm viewmodel.RoundResultsViewModel) FactPack {
	pack := FactPack{
		PostType: "round_results",
		League:   vm.LeagueName,
		Season:   vm.Season,
		Round:    vm.RoundTitle,
		Stats: map[string]interface{}{
			"total_matches": vm.Stats.TotalMatches,
			"total_goals":   vm.Stats.TotalGoals,
			"avg_goals":     vm.Stats.AvgGoals,
			"home_wins":     vm.Stats.HomeWins,
			"draws":         vm.Stats.Draws,
			"away_wins":     vm.Stats.AwayWins,
			"red_cards":     vm.Stats.TotalReds,
		},
	}

	if vm.Stats.TopMatch != "" {
		pack.TopMatch = &TopMatchFact{
			Fixture: vm.Stats.TopMatch,
		}
	}

	allMatches := append([]viewmodel.MatchScoreItem{}, vm.LeftCol...)
	allMatches = append(allMatches, vm.RightCol...)

	for _, m := range allMatches {
		fact := KeyMatchFact{
			HomeTeam:  m.HomeTeamName,
			AwayTeam:  m.AwayTeamName,
			Score:     fmt.Sprintf("%d:%d", m.HomeScore, m.AwayScore),
			IsUpset:   m.IsUpset,
			UpsetText: m.UpsetText,
		}
		if m.XGHome != "" && m.XGAway != "" {
			fact.XG = fmt.Sprintf("%s - %s", m.XGHome, m.XGAway)
		}
		var scorers []string
		scorers = append(scorers, m.HomeScorers...)
		scorers = append(scorers, m.AwayScorers...)
		fact.Scorers = scorers

		pack.KeyMatches = append(pack.KeyMatches, fact)

		if m.IsUpset && m.UpsetText != "" {
			pack.Dramas = append(pack.Dramas, fmt.Sprintf("%s: победа %s над %s (%s)", m.UpsetText, m.HomeTeamName, m.AwayTeamName, fact.Score))
		}
		if m.HomeRedCards > 0 {
			pack.Dramas = append(pack.Dramas, fmt.Sprintf("Красная карточка у %s", m.HomeTeamName))
		}
		if m.AwayRedCards > 0 {
			pack.Dramas = append(pack.Dramas, fmt.Sprintf("Красная карточка у %s", m.AwayTeamName))
		}
	}

	return pack
}

// BuildRoundFixturesFactPack формирует FactPack для анонса предстоящего тура.
func BuildRoundFixturesFactPack(vm viewmodel.RoundFixturesViewModel) FactPack {
	pack := FactPack{
		PostType: "round_fixtures",
		League:   vm.LeagueName,
		Season:   vm.Season,
		Round:    vm.RoundTitle,
		Stats: map[string]interface{}{
			"date_range": vm.DateRange,
			"bookmaker":  vm.BookmakerName,
			"timezone":   vm.Timezone,
		},
	}

	if vm.FeaturedMatch != nil {
		fm := vm.FeaturedMatch
		events := []string{
			fmt.Sprintf("%s • %s %s • %s • #%d vs #%d", fm.KickoffDate, fm.KickoffTime, vm.Timezone, fm.Venue, fm.HomeRank, fm.AwayRank),
		}
		if fm.HasOdds() {
			events = append(events, fmt.Sprintf("1X2: <i>[%s · %s · %s]</i>", fm.OddsHome, fm.OddsDraw, fm.OddsAway))
		}
		if fm.IsDerby {
			events = append(events, "⚔️ Принципиальное дерби")
		}
		pack.TopMatch = &TopMatchFact{
			Fixture:   fmt.Sprintf("%s — %s", fm.HomeTeamName, fm.AwayTeamName),
			KeyEvents: events,
		}
	}

	var allFixtures []viewmodel.FixtureMatchItem
	allFixtures = append(allFixtures, vm.LeftCol...)
	allFixtures = append(allFixtures, vm.RightCol...)

	// Сортируем матчи хронологически
	sort.SliceStable(allFixtures, func(i, j int) bool {
		return allFixtures[i].KickoffTimestamp.Before(allFixtures[j].KickoffTimestamp)
	})

	for _, f := range allFixtures {
		item := FixtureFact{
			HomeTeam: f.HomeTeamName,
			AwayTeam: f.AwayTeamName,
			Kickoff:  fmt.Sprintf("%s %s", f.KickoffDate, f.KickoffTime),
			Date:     f.KickoffDate,
			Time:     f.KickoffTime,
			RawTime:  f.KickoffTimestamp,
			IsDerby:  f.IsDerby,
		}
		if f.HasOdds() {
			item.OddsHome = f.OddsHome
			item.OddsDraw = f.OddsDraw
			item.OddsAway = f.OddsAway
		}
		pack.Fixtures = append(pack.Fixtures, item)
	}

	return pack
}
