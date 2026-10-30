package copywriter

import (
	"fmt"
	"strings"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/config"
)

// FactPostConfig задает параметры формирования Telegram-поста фактов и серий.
type FactPostConfig struct {
	LeagueName   string
	LeagueID     int
	BotUsername  string
	BotShortName string
	BaseURL      string
}

// GenerateFactHTMLPost формирует форматированный HTML-текст Telegram-поста для факта или серии.
func GenerateFactHTMLPost(rec analytics.StreakRecord, cfg FactPostConfig) string {
	var b strings.Builder

	leagueFlag := getLeagueFlag(rec.LeagueID)
	icon := rec.StreakTypeIcon
	if icon == "" {
		icon = "🔥"
	}

	scopeName := rec.ScopeName
	if scopeName == "" {
		scopeName = "Чемпионат"
	}

	// 1. Главный привлекающий заголовок
	var titleText string
	switch rec.StreakType {
	case analytics.StreakTypeTeamWin, analytics.StreakTypeCoachWin:
		titleText = fmt.Sprintf("%d ПОБЕД ПОДРЯД", rec.Length)
	case analytics.StreakTypeTeamUnbeaten, analytics.StreakTypeCoachUnbeaten:
		titleText = fmt.Sprintf("%d МАТЧЕЙ БЕЗ ПОРАЖЕНИЙ", rec.Length)
	case analytics.StreakTypeTeamNoWin:
		titleText = fmt.Sprintf("%d МАТЧЕЙ БЕЗ ПОБЕД", rec.Length)
	case analytics.StreakTypeTeamScoring:
		titleText = fmt.Sprintf("%d МАТЧЕЙ С ЗАБИТЫМИ ГОЛАМИ", rec.Length)
	case analytics.StreakTypeTeamCleanSheet:
		titleText = fmt.Sprintf("%d СУХИХ МАТЧЕЙ ПОДРЯД", rec.Length)
	default:
		titleText = fmt.Sprintf("СЕРИЯ: %d МАТЧЕЙ", rec.Length)
	}

	leagueHeader := strings.ToUpper(rec.LeagueName)
	if leagueHeader == "" {
		leagueHeader = strings.ToUpper(cfg.LeagueName)
	}

	b.WriteString(fmt.Sprintf("%s %s <b>%s: %s</b>\n", leagueFlag, icon, strings.ToUpper(rec.EntityName), titleText))
	if leagueHeader != "" {
		b.WriteString(fmt.Sprintf("<i>%s · %s</i>\n\n", leagueHeader, scopeName))
	} else {
		b.WriteString(fmt.Sprintf("<i>%s</i>\n\n", scopeName))
	}

	// 2. Рекордные отметки
	if rec.IsRecordLeague {
		b.WriteString("👑 <b>АБСОЛЮТНЫЙ РЕКОРД ЛИГИ!</b>\n")
		if rec.PrevLeagueRecord > 0 {
			b.WriteString(fmt.Sprintf("Предыдущий рекорд (%d) успешно превзойдён!\n\n", rec.PrevLeagueRecord))
		} else {
			b.WriteString("Историческое достижение в турнире!\n\n")
		}
	} else if rec.IsRecordClub {
		b.WriteString("⭐ <b>НОВЫЙ КЛУБНЫЙ РЕКОРД!</b>\n")
		if rec.PrevClubRecord > 0 {
			b.WriteString(fmt.Sprintf("Предыдущий рекорд клуба (%d) остался позади.\n\n", rec.PrevClubRecord))
		} else {
			b.WriteString("Лучший показатель в истории команды!\n\n")
		}
	}

	// 3. Главный тренер
	if rec.CoachName != "" {
		b.WriteString(fmt.Sprintf("👔 Главный тренер: <b>%s</b>\n\n", rec.CoachName))
	}

	// 4. Хронология матчей серии
	if len(rec.FixturesHistory) > 0 {
		b.WriteString("🗓 <b>Матчи серии:</b>\n")
		history := rec.FixturesHistory
		// Берем до 7 последних матчей для компактного Telegram-сообщения
		startIdx := 0
		if len(history) > 7 {
			startIdx = len(history) - 7
			b.WriteString("<i>... (ранее ещё матчи)</i>\n")
		}

		for i := startIdx; i < len(history); i++ {
			h := history[i]
			venue := "Д"
			if !h.IsHome {
				venue = "Г"
			}
			resEmoji := "🟢"
			if h.Result == "D" {
				resEmoji = "🟡"
			} else if h.Result == "L" {
				resEmoji = "🔴"
			}

			b.WriteString(fmt.Sprintf("%s <code>%s</code> · <b>%d:%d</b> vs %s (%s)\n",
				resEmoji, h.DateStr, h.TeamGoals, h.OpponentGoals, h.OpponentName, venue))
		}
		b.WriteString("\n")
	}

	// 5. Агрегированные метрики серии
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

	b.WriteString("📊 <b>Статистика серии:</b>\n")
	b.WriteString(fmt.Sprintf("⚽ Забито голов: <b>%d</b> (в ср. %.1f)\n", scored, avgScored))
	b.WriteString(fmt.Sprintf("🛡 Пропущено: <b>%d</b> (в ср. %.1f)\n", conceded, avgConceded))
	b.WriteString(fmt.Sprintf("🧤 «Сухих» матчей: <b>%d из %d</b>\n", cleanSheets, rec.Length))

	if rec.StreakType == analytics.StreakTypeTeamWin || rec.StreakType == analytics.StreakTypeCoachWin {
		maxPts := rec.Length * 3
		b.WriteString(fmt.Sprintf("🎯 Набрано очков: <b>%d из %d</b> (100%%)\n", maxPts, maxPts))
	}
	b.WriteString("\n")

	// 6. Подсказка для аудитории
	b.WriteString("💡 <i>Следите за всеми актуальными сериями и рекордами в Mini App</i>")

	return b.String()
}

// BuildFactMiniAppURL генерирует deep link ссылку на серию / команду в Telegram Mini App.
func BuildFactMiniAppURL(cfg FactPostConfig, rec analytics.StreakRecord) string {
	botUser := cfg.BotUsername
	if botUser == "" {
		if config.Config.WebApp != nil && config.Config.WebApp.BotUsername != "" {
			botUser = config.Config.WebApp.BotUsername
		} else {
			botUser = "football_data_lab_bot"
		}
	}

	botShort := cfg.BotShortName
	if botShort == "" {
		if config.Config.WebApp != nil && config.Config.WebApp.BotShortName != "" {
			botShort = config.Config.WebApp.BotShortName
		} else {
			botShort = "FDL"
		}
	}

	if rec.ID > 0 {
		return fmt.Sprintf("https://t.me/%s/%s?startapp=streak_%d", botUser, botShort, rec.ID)
	}
	return fmt.Sprintf("https://t.me/%s/%s?startapp=team_%d", botUser, botShort, rec.EntityID)
}
