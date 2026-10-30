package viewmodel

import (
	"fmt"
	"time"

	"github.com/Pastir/footballdata/analytics"
)

// LeaderCardItem представляет карточку одного игрока для рендеринга и Mini App.
type LeaderCardItem struct {
	Rank         int    `json:"rank"`
	PlayerID     int64  `json:"player_id"`
	PlayerName   string `json:"player_name"`
	PlayerImage  string `json:"player_image"`
	TeamID       int    `json:"team_id"`
	TeamName     string `json:"team_name"`
	TeamCode     string `json:"team_code"`
	TeamLogo     string `json:"team_logo"`
	Matches      int    `json:"matches"`
	PrimaryVal   int    `json:"primary_val"`
	SecondaryVal int    `json:"secondary_val"`
	ValStr       string `json:"val_str"`       // e.g. "4"
	SubValStr    string `json:"sub_val_str"`   // e.g. "0 пен" или "2 гола"
	MetricLabel  string `json:"metric_label"`
}

// LeaderCategoryGroup представляет группу лидеров одной категории (топ-5 или топ-10).
type LeaderCategoryGroup struct {
	Key     string           `json:"key"`   // "scorers", "assists", "points", "cleansheets", "discipline"
	Title   string           `json:"title"` // "Бомбардиры", "Ассистенты"
	Icon    string           `json:"icon"`  // "⚽", "🎯", "⚡", "🧤", "🟨"
	Metric  string           `json:"metric"`
	TopItem *LeaderCardItem  `json:"top_item"` // Игрок №1 (крупная Hero-карточка)
	Items   []LeaderCardItem `json:"items"`    // Остальные игроки
}

// PlayerLeadersViewModel объединяет все данные для рендеринга инфографики и Mini App.
type PlayerLeadersViewModel struct {
	LeagueName   string                `json:"league_name"`
	SeasonName   string                `json:"season_name"`
	UpdatedAtStr string                `json:"updated_at_str"`
	Categories   []LeaderCategoryGroup `json:"categories"`
}

// BuildLeadersViewModel формирует готовую модель представления лидеров турнира.
func BuildLeadersViewModel(data *analytics.SeasonLeadersData) PlayerLeadersViewModel {
	vm := PlayerLeadersViewModel{
		LeagueName:   data.LeagueName,
		SeasonName:   data.SeasonName,
		UpdatedAtStr: time.Now().Format("02.01.2006"),
	}

	// 1. Бомбардиры
	if len(data.TopScorers) > 0 {
		group := mapCategoryGroup("scorers", "Бомбардиры", "⚽", "голы", data.TopScorers, func(item analytics.PlayerLeaderItem) string {
			if item.SecondaryVal > 0 {
				return fmt.Sprintf("%d пен", item.SecondaryVal)
			}
			return fmt.Sprintf("%d матчей", item.Matches)
		})
		vm.Categories = append(vm.Categories, group)
	}

	// 2. Ассистенты
	if len(data.TopAssists) > 0 {
		group := mapCategoryGroup("assists", "Ассистенты", "🎯", "пасы", data.TopAssists, func(item analytics.PlayerLeaderItem) string {
			return fmt.Sprintf("%d матчей", item.Matches)
		})
		vm.Categories = append(vm.Categories, group)
	}

	// 3. Гол + Пас
	if len(data.TopPoints) > 0 {
		group := mapCategoryGroup("points", "Гол + Пас", "⚡", "очки", data.TopPoints, func(item analytics.PlayerLeaderItem) string {
			return fmt.Sprintf("%d г + %d пас", item.SecondaryVal, item.PrimaryVal-item.SecondaryVal)
		})
		vm.Categories = append(vm.Categories, group)
	}

	// 4. «Сухие» матчи
	if len(data.CleanSheets) > 0 {
		group := mapCategoryGroup("cleansheets", "Сухие матчи", "🧤", "сухих игр", data.CleanSheets, func(item analytics.PlayerLeaderItem) string {
			return fmt.Sprintf("проп: %d / %d м", item.SecondaryVal, item.Matches)
		})
		vm.Categories = append(vm.Categories, group)
	}

	// 5. Дисциплина
	if len(data.Discipline) > 0 {
		group := mapCategoryGroup("discipline", "Карточки", "🟨", "карточки", data.Discipline, func(item analytics.PlayerLeaderItem) string {
			if item.SecondaryVal > 0 {
				return fmt.Sprintf("%d КК • %d ЖК", item.SecondaryVal, item.PrimaryVal)
			}
			return fmt.Sprintf("%d ЖК • %d м", item.PrimaryVal, item.Matches)
		})
		vm.Categories = append(vm.Categories, group)
	}

	return vm
}

func mapCategoryGroup(
	key, title, icon, metric string,
	rawItems []analytics.PlayerLeaderItem,
	subValFn func(analytics.PlayerLeaderItem) string,
) LeaderCategoryGroup {
	group := LeaderCategoryGroup{
		Key:    key,
		Title:  title,
		Icon:   icon,
		Metric: metric,
	}

	for i, r := range rawItems {
		item := LeaderCardItem{
			Rank:         r.Rank,
			PlayerID:     r.PlayerID,
			PlayerName:   r.PlayerName,
			PlayerImage:  r.PlayerImage,
			TeamID:       r.TeamID,
			TeamName:     r.TeamName,
			TeamCode:     r.TeamCode,
			TeamLogo:     r.TeamLogo,
			Matches:      r.Matches,
			PrimaryVal:   r.PrimaryVal,
			SecondaryVal: r.SecondaryVal,
			ValStr:       fmt.Sprintf("%d", r.PrimaryVal),
			SubValStr:    subValFn(r),
			MetricLabel:  r.MetricLabel,
		}

		if i == 0 {
			top := item
			group.TopItem = &top
		}
		group.Items = append(group.Items, item)
	}

	return group
}
