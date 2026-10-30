package copywriter

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/viewmodel"
)

// RoundFixturesPostConfig задает параметры форматирования поста расписания тура.
type RoundFixturesPostConfig struct {
	LeagueName   string
	RoundName    string
	LeagueID     int
	SeasonID     int
	RoundID      int
	BotUsername  string
	BotShortName string
	BaseURL      string
}

// GenerateRoundFixturesHTMLPost формирует форматированный HTML-текст Telegram-поста с расписанием тура.
func GenerateRoundFixturesHTMLPost(
	cfg RoundFixturesPostConfig,
	vm viewmodel.RoundFixturesViewModel,
) string {
	var b strings.Builder

	// Заголовок турнира и тура
	leagueFlag := getLeagueFlag(cfg.LeagueID)
	roundTitle := vm.RoundTitle
	if roundTitle == "" {
		roundTitle = cfg.RoundName
	}
	if !strings.Contains(strings.ToLower(roundTitle), "тур") && !strings.Contains(strings.ToLower(roundTitle), "round") {
		roundTitle = fmt.Sprintf("Тур %s", roundTitle)
	}
	b.WriteString(fmt.Sprintf("%s <b>%s · %s — РАСПИСАНИЕ</b>\n\n",
		leagueFlag, strings.ToUpper(cfg.LeagueName), strings.ToUpper(roundTitle)))

	// Сбор всех матчей для вычисления максимальной ширины левой колонки (Время + Команда хозяев)
	var allMatches []viewmodel.FixtureMatchItem
	for _, dg := range vm.DayGroups {
		allMatches = append(allMatches, dg.Matches...)
	}

	maxPrefixLen := 0
	for _, m := range allMatches {
		kTime := m.KickoffTime
		if kTime == "" {
			kTime = "  -  "
		}
		prefix := fmt.Sprintf("%s  %s", kTime, m.HomeTeamName)
		rCount := utf8.RuneCountInString(prefix)
		if rCount > maxPrefixLen {
			maxPrefixLen = rCount
		}
	}

	// Вывод матчей по группам дней
	for _, dg := range vm.DayGroups {
		b.WriteString(fmt.Sprintf("<b>%s</b>\n", dg.DayLabel))
		for _, m := range dg.Matches {
			kTime := m.KickoffTime
			if kTime == "" {
				kTime = "  -  "
			}
			prefix := fmt.Sprintf("%s  %s", kTime, m.HomeTeamName)

			// Котировки 1X2 или дефолтная плашка
			oddsText := "[ VS ]"
			if m.HasOdds() {
				oddsText = fmt.Sprintf("[ %s · %s · %s ]", m.OddsHome, m.OddsDraw, m.OddsAway)
			} else if m.KickoffTime != "" {
				oddsText = fmt.Sprintf("[ %s ]", m.KickoffTime)
			}

			// Ссылка на матч в Telegram Mini App
			matchURL := buildFixturesMatchMiniAppURL(cfg, m.ID)

			pad := (maxPrefixLen + 2) - utf8.RuneCountInString(prefix)
			if pad < 1 {
				pad = 1
			}
			spaces := strings.Repeat(" ", pad)

			line := fmt.Sprintf("<code>%s%s</code><a href=\"%s\"><b>%s</b></a><code>  %s</code>",
				prefix, spaces, matchURL, oddsText, m.AwayTeamName)

			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	// Блок «Матч тура»
	feat := vm.FeaturedMatch
	if feat != nil {
		b.WriteString(fmt.Sprintf("🔥 <b>МАТЧ ТУРА: %s — %s</b>\n", feat.HomeTeamName, feat.AwayTeamName))
		if feat.HasProbabilities() {
			probText := fmt.Sprintf("<b>Прогноз модели:</b> П1 %s | Х %s | П2 %s", feat.ProbHome, feat.ProbDraw, feat.ProbAway)
			b.WriteString(probText)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Подсказка для подписчиков
	b.WriteString("💡 <i>Нажмите на котировки матча, чтобы открыть H2H, xG и составы</i>")

	return b.String()
}

// buildFixturesMatchMiniAppURL генерирует ссылку на матч в Telegram Mini App для предстоящего тура.
func buildFixturesMatchMiniAppURL(cfg RoundFixturesPostConfig, fixtureID int64) string {
	botUser := cfg.BotUsername
	if botUser == "" {
		if config.Config.WebApp != nil && config.Config.WebApp.BotUsername != "" {
			botUser = config.Config.WebApp.BotUsername
		} else {
			botUser = "football_data_lab_bot"
		}
	}
	shortName := cfg.BotShortName
	if shortName == "" {
		if config.Config.WebApp != nil && config.Config.WebApp.BotShortName != "" {
			shortName = config.Config.WebApp.BotShortName
		} else {
			shortName = "FDL"
		}
	}

	// Формат deep link: fixture_{leagueId}_{seasonId}_{roundId}_{fixtureId}
	startAppParam := fmt.Sprintf("fixture_%d_%d_%d_%d", cfg.LeagueID, cfg.SeasonID, cfg.RoundID, fixtureID)
	return fmt.Sprintf("https://t.me/%s/%s?startapp=%s", botUser, shortName, startAppParam)
}
