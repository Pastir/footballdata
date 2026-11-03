package copywriter

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Pastir/footballdata/analytics"
)

// LeagueLeadersPostConfig содержит параметры для формирования поста лидеров сезона.
type LeagueLeadersPostConfig struct {
	LeagueName   string
	SeasonName   string
	LeagueID     int
	SeasonID     int
	BotUsername  string
	BotShortName string
	BaseURL      string
}

// GenerateLeadersHTMLPost генерирует моноширинный аналитический пост для Telegram.
func GenerateLeadersHTMLPost(cfg LeagueLeadersPostConfig, data *analytics.SeasonLeadersData) string {
	var b strings.Builder

	leagueUpper := strings.ToUpper(cfg.LeagueName)
	b.WriteString(fmt.Sprintf("👑 <b>%s — ЛИДЕРЫ СЕЗОНА %s</b>\n", leagueUpper, cfg.SeasonName))
	b.WriteString("<i>Индивидуальная статистика ключевых игроков турнира</i>\n\n")

	// 1. Топ бомбардиров (до 5 игроков)
	if len(data.TopScorers) > 0 {
		b.WriteString("⚽ <b>БОМБАРДИРЫ:</b>\n")
		b.WriteString("<code>")
		b.WriteString(fmt.Sprintf("%-2s %-16s %-4s %4s\n", "#", "ИГРОК", "КЛУБ", "ГОЛЫ"))
		b.WriteString("─────────────────────────────\n")
		limit := 5
		if len(data.TopScorers) < limit {
			limit = len(data.TopScorers)
		}
		for _, p := range data.TopScorers[:limit] {
			pName := truncateName(p.PlayerName, 16)
			tCode := p.TeamCode
			if tCode == "" {
				tCode = truncateName(p.TeamName, 4)
			}
			valStr := fmt.Sprintf("%d", p.PrimaryVal)
			if p.SecondaryVal > 0 {
				valStr = fmt.Sprintf("%d(%d)", p.PrimaryVal, p.SecondaryVal)
			}
			b.WriteString(fmt.Sprintf("%-2d %-16s %-4s %4s\n", p.Rank, pName, tCode, valStr))
		}
		b.WriteString("</code>\n\n")
	}

	// 2. Топ ассистентов (до 5 игроков)
	if len(data.TopAssists) > 0 {
		b.WriteString("🎯 <b>АССИСТЕНТЫ:</b>\n")
		b.WriteString("<code>")
		b.WriteString(fmt.Sprintf("%-2s %-16s %-4s %4s\n", "#", "ИГРОК", "КЛУБ", "ПАСЫ"))
		b.WriteString("─────────────────────────────\n")
		limit := 5
		if len(data.TopAssists) < limit {
			limit = len(data.TopAssists)
		}
		for _, p := range data.TopAssists[:limit] {
			pName := truncateName(p.PlayerName, 16)
			tCode := p.TeamCode
			if tCode == "" {
				tCode = truncateName(p.TeamName, 4)
			}
			b.WriteString(fmt.Sprintf("%-2d %-16s %-4s %4d\n", p.Rank, pName, tCode, p.PrimaryVal))
		}
		b.WriteString("</code>\n\n")
	}

	// 3. «Сухие» матчи вратарей (до 4 голкиперов)
	if len(data.CleanSheets) > 0 {
		b.WriteString("🧤 <b>СУХИЕ МАТЧИ ВРАТАРЕЙ:</b>\n")
		b.WriteString("<code>")
		b.WriteString(fmt.Sprintf("%-2s %-16s %-4s %4s\n", "#", "ВРАТАРЬ", "КЛУБ", "СУХ"))
		b.WriteString("─────────────────────────────\n")
		limit := 4
		if len(data.CleanSheets) < limit {
			limit = len(data.CleanSheets)
		}
		for _, p := range data.CleanSheets[:limit] {
			pName := truncateName(p.PlayerName, 16)
			tCode := p.TeamCode
			if tCode == "" {
				tCode = truncateName(p.TeamName, 4)
			}
			b.WriteString(fmt.Sprintf("%-2d %-16s %-4s %4d\n", p.Rank, pName, tCode, p.PrimaryVal))
		}
		b.WriteString("</code>\n\n")
	}

	b.WriteString("<i>💡 Полные рейтинги, Гол+Пас, карточки и расширенная статистика доступны в Telegram Mini App:</i>")
	return b.String()
}

func truncateName(name string, maxLen int) string {
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) <= maxLen {
		return name + strings.Repeat(" ", maxLen-len(runes))
	}
	return string(runes[:maxLen-1]) + "."
}

func padRight(s string, width int) string {
	l := utf8.RuneCountInString(s)
	if l >= width {
		return s
	}
	return s + strings.Repeat(" ", width-l)
}
