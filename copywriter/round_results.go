package copywriter

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/config"
)

// RoundResultsPostConfig задает параметры форматирования поста результатов тура.
type RoundResultsPostConfig struct {
	LeagueName   string
	RoundName    string
	LeagueID     int
	SeasonID     int
	RoundID      int
	BotUsername  string
	BotShortName string
	BaseURL      string
}

// GenerateRoundResultsHTMLPost формирует форматированный HTML-текст Telegram-поста с результатами тура.
func GenerateRoundResultsHTMLPost(
	cfg RoundResultsPostConfig,
	matches []analytics.RoundResultMatch,
	summary analytics.RoundSummary,
) string {
	var b strings.Builder

	// Заголовок турнира и тура
	leagueFlag := getLeagueFlag(cfg.LeagueID)
	roundTitle := cfg.RoundName
	if !strings.Contains(strings.ToLower(roundTitle), "тур") && !strings.Contains(strings.ToLower(roundTitle), "round") {
		roundTitle = fmt.Sprintf("Тур %s", roundTitle)
	}
	b.WriteString(fmt.Sprintf("%s <b>%s · %s — РЕЗУЛЬТАТЫ</b>\n\n", leagueFlag, strings.ToUpper(cfg.LeagueName), strings.ToUpper(roundTitle)))

	// Определение максимальной длины названия команды хозяев для выравнивания колонок
	maxHomeLen := 0
	for _, m := range matches {
		rCount := utf8.RuneCountInString(m.TeamHomeName)
		if rCount > maxHomeLen {
			maxHomeLen = rCount
		}
	}

	// Группировка матчей по датам (дню начала)
	dateGroups := groupMatchesByDay(matches)
	for _, dg := range dateGroups {
		b.WriteString(fmt.Sprintf("<b>%s</b>\n", dg.DayHeader))
		for _, m := range dg.Matches {
			scoreText := m.Scores.FullScoreString()
			htText := m.Scores.HTScoreString()
			if htText != "" {
				scoreText = fmt.Sprintf("%s %s", scoreText, htText)
			}

			// Ссылка на матч в Telegram Mini App
			matchURL := buildMatchMiniAppURL(cfg, m.FixtureID)

			// Выравнивание: точное пиксельное выравнивание колонок через monospace <code>
			pad := (maxHomeLen + 2) - utf8.RuneCountInString(m.TeamHomeName)
			if pad < 1 {
				pad = 1
			}
			spaces := strings.Repeat(" ", pad)

			line := fmt.Sprintf("<code>%s%s</code><a href=\"%s\"><b>%s</b></a><code>  %s</code>",
				m.TeamHomeName, spaces, matchURL, scoreText, m.TeamAwayName)

			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	// Блок экспресс-статистики тура (минималистичный стиль, выделенные жирным заголовки)
	if summary.FinishedMatches > 0 {
		b.WriteString("📊 <b>ЭКСПРЕСС-СТАТИСТИКА ТУРА:</b>\n")
		b.WriteString(fmt.Sprintf("<b>Забито голов:</b> %d (в среднем %.1f за матч)\n", summary.TotalGoals, summary.AvgGoals))
		b.WriteString(fmt.Sprintf("<b>%d</b> побед хозяев  |  <b>%d</b> ничьих  |  <b>%d</b> победы гостей\n", summary.HomeWins, summary.Draws, summary.AwayWins))

		if summary.FavoritesWon+summary.FavoritesFailed > 0 {
			b.WriteString(fmt.Sprintf("<b>Букмекеры:</b> фавориты сыграли %d / %d\n", summary.FavoritesWon, summary.FavoritesWon+summary.FavoritesFailed))
		}

		if summary.BTTSCount > 0 {
			b.WriteString(fmt.Sprintf("<b>Обе забьют (ОЗ):</b> %d из %d матчей\n", summary.BTTSCount, summary.FinishedMatches))
		}

		if summary.TopUpsetMatch != "" {
			b.WriteString(fmt.Sprintf("<b>Главная сенсация:</b> %s\n", summary.TopUpsetMatch))
		}

		if summary.LateDramaEvent != "" {
			b.WriteString(fmt.Sprintf("<b>Драма тура:</b> <i>%s</i>\n", summary.LateDramaEvent))
		}

		if summary.TotalYellowCards > 0 || summary.TotalRedCards > 0 {
			b.WriteString(fmt.Sprintf("<b>Дисциплина:</b> %d 🟨 · %d 🟥\n", summary.TotalYellowCards, summary.TotalRedCards))
		}
		b.WriteString("\n")
	}

	// Подсказка для читателей
	b.WriteString("💡 <i>Нажмите на счёт матча, чтобы открыть xG, события и составы</i>")

	return b.String()
}

// buildMatchMiniAppURL генерирует прямую deep link ссылку на матч в Telegram Mini App.
func buildMatchMiniAppURL(cfg RoundResultsPostConfig, fixtureID int64) string {
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

	if cfg.LeagueID > 0 && cfg.SeasonID > 0 && cfg.RoundID > 0 {
		return fmt.Sprintf("https://t.me/%s/%s?startapp=match_%d_%d_%d_%d", botUser, botShort, cfg.LeagueID, cfg.SeasonID, cfg.RoundID, fixtureID)
	}
	return fmt.Sprintf("https://t.me/%s/%s?startapp=match_%d", botUser, botShort, fixtureID)
}

// matchDayGroup группирует матчи одного игрового дня.
type matchDayGroup struct {
	DateKey   string
	DayHeader string
	Matches   []analytics.RoundResultMatch
}

func groupMatchesByDay(matches []analytics.RoundResultMatch) []matchDayGroup {
	groupsMap := make(map[string]*matchDayGroup)
	var keys []string

	for _, m := range matches {
		d := m.FixtureStart.UTC()
		dateKey := d.Format("2006-01-02")

		if _, exists := groupsMap[dateKey]; !exists {
			groupsMap[dateKey] = &matchDayGroup{
				DateKey:   dateKey,
				DayHeader: formatRussianDate(d),
				Matches:   nil,
			}
			keys = append(keys, dateKey)
		}
		groupsMap[dateKey].Matches = append(groupsMap[dateKey].Matches, m)
	}

	sort.Strings(keys)
	var result []matchDayGroup
	for _, k := range keys {
		result = append(result, *groupsMap[k])
	}
	return result
}

// formatRussianDate форматирует дату на русском: "10 Окт. 2026, Суббота"
func formatRussianDate(t time.Time) string {
	months := map[time.Month]string{
		time.January:   "Янв.",
		time.February:  "Фев.",
		time.March:     "Мар.",
		time.April:     "Апр.",
		time.May:       "Мая",
		time.June:      "Июн.",
		time.July:      "Июл.",
		time.August:    "Авг.",
		time.September: "Сен.",
		time.October:   "Окт.",
		time.November:  "Ноя.",
		time.December:  "Дек.",
	}
	weekdays := map[time.Weekday]string{
		time.Monday:    "Понедельник",
		time.Tuesday:   "Вторник",
		time.Wednesday: "Среда",
		time.Thursday:  "Четверг",
		time.Friday:    "Пятница",
		time.Saturday:  "Суббота",
		time.Sunday:    "Воскресенье",
	}

	return fmt.Sprintf("%02d %s %d, %s",
		t.Day(),
		months[t.Month()],
		t.Year(),
		weekdays[t.Weekday()],
	)
}

// getLeagueFlag возвращает эмодзи флага страны по LeagueID.
func getLeagueFlag(leagueID int) string {
	switch leagueID {
	case 8, 9:
		return "🏴󠁧󠁢󠁥󠁮󠁧󠁿" // Англия
	case 564:
		return "🇪🇸" // Испания
	case 82:
		return "🇩🇪" // Германия
	case 384:
		return "🇮🇹" // Италия
	case 301:
		return "🇫🇷" // Франция
	case 2, 5:
		return "🏆" // Еврокубки
	default:
		return "⚽"
	}
}
