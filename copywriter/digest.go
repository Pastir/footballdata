package copywriter

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/viewmodel"
)

// MatchMetricRow описывает одну строку сравнительной таблицы метрик.
type MatchMetricRow struct {
	Label   string
	HomeVal string
	AwayVal string
}

// FormatForm объединяет элементы формы с заданным разделителем и лимитом.
func FormatForm(form []string, maxItems int, sep string) string {
	if len(form) == 0 {
		return "-"
	}
	if len(form) > maxItems {
		form = form[len(form)-maxItems:]
	}
	return strings.Join(form, sep)
}

// PadCenter центрирует строку в поле заданной ширины.
func PadCenter(s string, width int) string {
	runeLen := utf8.RuneCountInString(s)
	if runeLen >= width {
		return s
	}
	left := (width - runeLen) / 2
	right := width - runeLen - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// FormatDynamicComparisonTable форматирует моноширинную сравнительную таблицу матча.
func FormatDynamicComparisonTable(
	homeCode, awayCode string,
	oddsHome, oddsDraw, oddsAway string,
	metrics []MatchMetricRow,
) string {
	var b strings.Builder
	b.WriteString("<code>")

	hasOdds := oddsHome != "" && oddsAway != "" && oddsHome != "-" && oddsAway != "-" && oddsHome != "0" && oddsAway != "0"

	if hasOdds {
		col1 := PadCenter(homeCode, 10)
		col2 := PadCenter("1X2", 13)
		col3 := PadCenter(awayCode, 10)
		b.WriteString(col1 + col2 + col3 + "\n")

		oddsH := PadCenter(oddsHome, 10)
		oddsD := PadCenter(oddsDraw, 13)
		if oddsDraw == "" {
			oddsD = PadCenter("-", 13)
		}
		oddsA := PadCenter(oddsAway, 10)
		b.WriteString(oddsH + oddsD + oddsA + "\n")
	} else {
		col1 := PadCenter(homeCode, 10)
		col2 := PadCenter("vs", 13)
		col3 := PadCenter(awayCode, 10)
		b.WriteString(col1 + col2 + col3 + "\n")
	}

	// Короткая аккуратная разделительная линия по центру (17 символов)
	b.WriteString(PadCenter(strings.Repeat("─", 17), 33) + "\n")

	for i, m := range metrics {
		valH := PadCenter(m.HomeVal, 10)
		lbl := PadCenter(m.Label, 13)
		valA := PadCenter(m.AwayVal, 10)
		b.WriteString(valH + lbl + valA)
		if i < len(metrics)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("</code>")
	return b.String()
}

// GenerateMatchIntrigue генерирует интригу матча на основе турнирного положения.
func GenerateMatchIntrigue(fm viewmodel.FixtureMatchItem, reasons []string) string {
	if fm.IsDerby {
		return fmt.Sprintf("Принципиальное дерби: бескомпромиссная битва на «%s». Смогут ли хозяева сломить сопротивление соперника?", fm.Venue)
	}
	if len(reasons) > 0 {
		for _, r := range reasons {
			if strings.Contains(strings.ToLower(r), "title race") {
				return fmt.Sprintf("Ключевое сражение в чемпионской гонке за 6 очков. Очная дуэль на «%s» определит расклад на вершине таблицы.", fm.Venue)
			}
		}
	}
	if fm.HomeRank > 0 && fm.AwayRank > 0 {
		if fm.HomeRank <= 4 && fm.AwayRank <= 6 {
			return fmt.Sprintf("Битва за еврокубковую зону: соседей по таблице разделяет минимум очков. На «%s» ожидается матч высочайшего накала.", fm.Venue)
		}
		if fm.AwayRank <= 3 && fm.HomeRank > fm.AwayRank {
			return fmt.Sprintf("Один из лидеров чемпионата едет на непростой выезд. Сможет ли оборона «%s» сдержать атаку гостей?", fm.HomeTeamName)
		}
	}
	return fmt.Sprintf("Центральное противостояние уик-энда. Сможет ли «%s» на домашнем «%s» взять максимум очков против «%s»?", fm.HomeTeamName, fm.Venue, fm.AwayTeamName)
}

// BuildInteractiveDigestCaption формирует подпись к шапке матча в дайджесте.
func BuildInteractiveDigestCaption(
	metricsTable string,
	intrigue string,
) string {
	var b strings.Builder
	b.WriteString(metricsTable)
	b.WriteString("\n\n")
	b.WriteString("🔎 <b>Интрига:</b> ")
	b.WriteString(intrigue)
	return b.String()
}

// TruncateStr усекает строку до maxLen с добавлением точки.
func TruncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "."
}

// FormatDayRussian форматирует дату на русском языке (День недели, число месяц).
func FormatDayRussian(t time.Time) string {
	days := []string{"Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота"}
	months := []string{
		"", "января", "февраля", "марта", "апреля", "мая", "июня",
		"июля", "августа", "сентября", "октября", "ноября", "декабря",
	}
	weekday := days[int(t.Weekday())]
	m := int(t.Month())
	if m >= 1 && m <= 12 {
		return fmt.Sprintf("%s, %d %s", weekday, t.Day(), months[m])
	}
	return t.Format("02.01.2006")
}

// BuildInteractivePostText формирует полный текст поста дайджеста тура.
func BuildInteractivePostText(
	info analytics.LeagueSeasonInfo,
	vm viewmodel.RoundFixturesViewModel,
	matches []analytics.NextRoundMatch,
	leagueName, roundNum string,
) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("🏴󠁧󠁢󠁥󠁮󠁧󠁿 <b>%s: ПРЕМЬЕР-ЛИГА — %s ТУР</b>\n", strings.ToUpper(leagueName), roundNum))
	if vm.DateRange != "" {
		b.WriteString(fmt.Sprintf("📅 %s\n\n", vm.DateRange))
	} else {
		b.WriteString("\n")
	}

	// 1. Блок МАТЧ ТУРА в табличной форме
	if vm.FeaturedMatch != nil {
		fm := *vm.FeaturedMatch
		b.WriteString("<blockquote>")
		b.WriteString(fmt.Sprintf("⭐ <b>МАТЧ ТУРА: %s — %s</b>\n", strings.ToUpper(fm.HomeTeamName), strings.ToUpper(fm.AwayTeamName)))

		meta := ""
		if fm.Venue != "" {
			meta += "📍 " + fm.Venue
		}
		if fm.KickoffTime != "" {
			if meta != "" {
				meta += " • "
			}
			meta += fm.KickoffDate + ", " + fm.KickoffTime
			if vm.Timezone != "" {
				meta += " " + vm.Timezone
			}
		}
		if meta != "" {
			b.WriteString(meta + "\n\n")
		}

		// Табличное сравнение параметров команд и котировок
		hTeam := TruncateStr(fm.HomeTeamName, 12)
		aTeam := TruncateStr(fm.AwayTeamName, 12)
		hRank := fmt.Sprintf("#%d", fm.HomeRank)
		if fm.HomeRank == 0 {
			hRank = "-"
		}
		aRank := fmt.Sprintf("#%d", fm.AwayRank)
		if fm.AwayRank == 0 {
			aRank = "-"
		}
		hForm := FormatForm(fm.HomeForm, 4, " ")
		aForm := FormatForm(fm.AwayForm, 4, " ")

		b.WriteString("<code>ПАРАМЕТР     ХОЗЯЕВА      ГОСТИ\n")
		b.WriteString("─────────────────────────────────\n")
		b.WriteString(fmt.Sprintf("Клуб         %-12s %-12s\n", hTeam, aTeam))
		b.WriteString(fmt.Sprintf("Позиция      %-12s %-12s\n", hRank, aRank))
		b.WriteString(fmt.Sprintf("Форма        %-12s %-12s\n", hForm, aForm))
		b.WriteString("─────────────────────────────────\n")
		if fm.OddsHome != "" && fm.OddsAway != "" {
			b.WriteString(fmt.Sprintf("1X2 Пари     1: %-9s 2: %s\n", fm.OddsHome, fm.OddsAway))
			if fm.OddsDraw != "" {
				b.WriteString(fmt.Sprintf("Ничья (X)    %s\n", fm.OddsDraw))
			}
		}
		b.WriteString("</code>")

		if fm.ProbHome != "" && fm.ProbAway != "" {
			b.WriteString(fmt.Sprintf("\n<i>Модель: %s (П1) · %s (X) · %s (П2)</i>", fm.ProbHome, fm.ProbDraw, fm.ProbAway))
		}
		b.WriteString("</blockquote>\n\n")
	}

	// 2. Расписание матчей тура
	b.WriteString("<b>Расписание встреч тура:</b>\n")
	if len(matches) > 0 {
		type dayGroup struct {
			label   string
			matches []analytics.NextRoundMatch
		}
		var groups []dayGroup
		dateMap := make(map[string]int)

		for _, m := range matches {
			dayStr := FormatDayRussian(m.FixtureStart)
			idx, exists := dateMap[dayStr]
			if !exists {
				dateMap[dayStr] = len(groups)
				groups = append(groups, dayGroup{label: dayStr})
				idx = dateMap[dayStr]
			}
			groups[idx].matches = append(groups[idx].matches, m)
		}

		for _, g := range groups {
			b.WriteString(fmt.Sprintf("\n<b>%s</b>\n", g.label))
			for _, m := range g.matches {
				tStr := m.FixtureStart.Format("15:04")
				oddsStr := ""
				if m.OddsHome != "" && m.OddsAway != "" {
					if m.OddsDraw != "" {
						oddsStr = fmt.Sprintf(" <i>(%s · %s · %s)</i>", m.OddsHome, m.OddsDraw, m.OddsAway)
					} else {
						oddsStr = fmt.Sprintf(" <i>(%s · %s)</i>", m.OddsHome, m.OddsAway)
					}
				}
				derby := ""
				if m.IsRival {
					derby = " ⚔️"
				}
				b.WriteString(fmt.Sprintf("• %s <b>%s — %s</b>%s%s\n", tStr, m.TeamHomeName, m.TeamAwayName, derby, oddsStr))
			}
		}
	} else {
		b.WriteString("<i>Матчи тура уточняются.</i>\n")
	}

	b.WriteString("\n<i>💡 Нажмите интерактивные кнопки ниже для просмотра инфографики турнирной таблицы и расписания:</i>")
	return b.String()
}
