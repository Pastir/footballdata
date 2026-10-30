package copywriter

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// GenerateFallbackPost формирует детерминированный структурированный текст поста
// без обращения к внешним нейросетям (используется при тайм-аутах или сбоях LLM).
func GenerateFallbackPost(pack FactPack) string {
	var b strings.Builder

	switch pack.PostType {
	case "round_results":
		b.WriteString(fmt.Sprintf("📊 <b>%s — %s: Итоги игрового уик-энда</b>\n\n", pack.League, pack.Round))

		if pack.Stats != nil {
			goals, _ := pack.Stats["total_goals"]
			avg, _ := pack.Stats["avg_goals"]
			reds, _ := pack.Stats["red_cards"]
			b.WriteString(fmt.Sprintf("⚽ Всего голов: <b>%v</b> (в среднем <b>%v</b> за игру)\n", goals, avg))
			if redsInt, ok := reds.(int); ok && redsInt > 0 {
				b.WriteString(fmt.Sprintf("🟥 Красных карточек: <b>%d</b>\n", redsInt))
			}
			b.WriteString("\n")
		}

		if len(pack.KeyMatches) > 0 {
			b.WriteString("<b>Ключевые результаты тура:</b>\n")
			limit := 5
			if len(pack.KeyMatches) < limit {
				limit = len(pack.KeyMatches)
			}
			for i := 0; i < limit; i++ {
				m := pack.KeyMatches[i]
				line := fmt.Sprintf("• <b>%s %s %s</b>", m.HomeTeam, m.Score, m.AwayTeam)
				if m.XG != "" {
					line += fmt.Sprintf(" <i>(xG %s)</i>", m.XG)
				}
				if m.IsUpset && m.UpsetText != "" {
					line += fmt.Sprintf(" — %s", m.UpsetText)
				}
				b.WriteString(line + "\n")
			}
			b.WriteString("\n")
		}

		if len(pack.Dramas) > 0 {
			b.WriteString("🔥 <b>События тура:</b>\n")
			for _, d := range pack.Dramas {
				b.WriteString(fmt.Sprintf("• %s\n", d))
			}
			b.WriteString("\n")
		}

		cta := "💬 <i>Какой матч тура запомнился вам больше всего? Делитесь в комментариях!</i>"
		if utf8.RuneCountInString(b.String())+utf8.RuneCountInString(cta) <= MaxCaptionLength {
			b.WriteString(cta)
		}

	case "round_fixtures":
		b.WriteString(fmt.Sprintf("⚡ <b>%s — %s: Расписание матчей</b>\n\n", pack.League, pack.Round))

		if pack.TopMatch != nil {
			b.WriteString(fmt.Sprintf("⭐ <b>Матч тура: %s</b>\n", pack.TopMatch.Fixture))
			for _, ev := range pack.TopMatch.KeyEvents {
				b.WriteString(fmt.Sprintf("• %s\n", ev))
			}
			b.WriteString("\n")
		}

		if len(pack.Fixtures) > 0 {
			b.WriteString("<b>Расписание встреч:</b>\n")
			currentDate := ""
			for _, f := range pack.Fixtures {
				if f.Date != "" && f.Date != currentDate {
					currentDate = f.Date
					b.WriteString(fmt.Sprintf("\n<b>%s</b>\n", currentDate))
				}
				timeStr := f.Time
				if timeStr == "" {
					timeStr = f.Kickoff
				}
				derbyMark := ""
				if f.IsDerby {
					derbyMark = " ⚔"
				}
				b.WriteString(fmt.Sprintf("• %s <b>%s — %s</b>%s\n", timeStr, f.HomeTeam, f.AwayTeam, derbyMark))
				if f.OddsHome != "" && f.OddsAway != "" {
					if f.OddsDraw != "" {
						b.WriteString(fmt.Sprintf("            %s · %s · %s\n", f.OddsHome, f.OddsDraw, f.OddsAway))
					} else {
						b.WriteString(fmt.Sprintf("            %s · %s\n", f.OddsHome, f.OddsAway))
					}
				}
			}
			b.WriteString("\n")
		}

		cta := "💬 <i>На кого ставите в центральном матче? Ждём прогнозы!</i>"
		if utf8.RuneCountInString(b.String())+utf8.RuneCountInString(cta) <= MaxCaptionLength {
			b.WriteString(cta)
		}

	default:
		b.WriteString(fmt.Sprintf("📊 <b>%s — %s</b>\n\nСвежая аналитика и актуальная статистика в инфографике выше.", pack.League, pack.Season))
	}

	return b.String()
}
