package copywriter

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Pastir/footballdata/analytics"
)

// ExpressPostConfig определяет параметры генерации поста экспресса.
type ExpressPostConfig struct {
	LeagueName    string
	RoundName     string
	BookmakerName string
	PromoCode     string
	BonusText     string
	BookmakerURL  string
	MiniAppURL    string
}

// GenerateExpressPostStyle1 формирует стиль [1 ТЕСТ]: The Athletic / Financial Times (Интеллектуальная колонка с <blockquote>).
func GenerateExpressPostStyle1(cfg ExpressPostConfig, plan *analytics.SmartExpressPlan) string {
	if plan == nil || len(plan.Selections) == 0 {
		return ""
	}

	var b strings.Builder

	leagueTitle := cfg.LeagueName
	if leagueTitle == "" {
		leagueTitle = "Премьер-лига"
	}
	roundTitle := cfg.RoundName
	if roundTitle != "" && !strings.Contains(strings.ToLower(roundTitle), "тур") {
		roundTitle = fmt.Sprintf("Тур %s", roundTitle)
	}

	b.WriteString("[1 ТЕСТ] <b>DATA LAB | АНАЛИТИЧЕСКИЙ ЭКСПРЕСС</b>\n")
	b.WriteString(fmt.Sprintf("<i>%s · %s · Алгоритмический отбор</i>\n\n", leagueTitle, roundTitle))

	b.WriteString(fmt.Sprintf("<b>ИТОГОВЫЙ КОЭФФИЦИЕНТ:</b> <code>%.2f</code>  •  <b>VALUE EDGE:</b> <code>+%.1f%%</code>\n", plan.TotalOdds, plan.ValueEdge))
	b.WriteString("<b>Математическое ожидание:</b> Положительное (+EV)\n")
	b.WriteString(fmt.Sprintf("<b>Рекомендуемый банк-менеджмент:</b> %.1f unit (1%% флэт)\n\n", plan.RecommendedStake))

	for _, sel := range plan.Selections {
		m := sel.Match
		hName := m.HomeDisplayName()
		aName := m.AwayDisplayName()
		timeStr := m.StartingAt.UTC().Format("02.01 · 15:04 UTC")

		badge := ""
		if m.CountryFlag != "" && m.LeagueName != "" {
			badge = fmt.Sprintf("%s %s · ", m.CountryFlag, m.LeagueName)
		} else if m.CountryFlag != "" {
			badge = fmt.Sprintf("%s ", m.CountryFlag)
		}

		b.WriteString("<blockquote>")
		b.WriteString(fmt.Sprintf("<b>%s%s — %s</b>\n", badge, hName, aName))
		b.WriteString(fmt.Sprintf("• Время: %s\n", timeStr))
		b.WriteString(fmt.Sprintf("• Выбор модели: <b>%s</b>  (кэф <b>%.2f</b>)\n", sel.OutcomeDisplayName(), sel.Odds))
		if sel.Reason != "" {
			b.WriteString(fmt.Sprintf("• <i>Метрика:</i> %s\n", sel.Reason))
		}
		b.WriteString("</blockquote>\n")
	}

	b.WriteString("\n<blockquote>Все прогнозы верифицируются и сохраняются в открытом реестре Data Lab. Решение о ставке принимайте ответственно на основе банк-ролл менеджмента.</blockquote>")

	var footers1 []string
	if cfg.BookmakerURL != "" {
		bk := cfg.BookmakerName
		if bk == "" {
			bk = "FONBET"
		}
		footers1 = append(footers1, fmt.Sprintf("<a href=\"%s\">Фрибет %s</a>", cfg.BookmakerURL, bk))
	}
	if cfg.MiniAppURL != "" {
		footers1 = append(footers1, fmt.Sprintf("<a href=\"%s\">Все матчи</a>", cfg.MiniAppURL))
	}
	if len(footers1) > 0 {
		b.WriteString("\n\n" + strings.Join(footers1, " · "))
	}

	return b.String()
}

// GenerateExpressPostStyle2 формирует стиль [2 ТЕСТ]: Bloomberg Terminal (Ультра-лаконичный ASCII купон в <pre> + факты).
func GenerateExpressPostStyle2(cfg ExpressPostConfig, plan *analytics.SmartExpressPlan) string {
	if plan == nil || len(plan.Selections) == 0 {
		return ""
	}

	var b strings.Builder

	leagueTitle := cfg.LeagueName
	if leagueTitle == "" {
		leagueTitle = "Премьер-лига"
	}
	roundTitle := cfg.RoundName
	if roundTitle != "" && !strings.Contains(strings.ToLower(roundTitle), "тур") {
		roundTitle = fmt.Sprintf("Тур %s", roundTitle)
	}

	b.WriteString("[2 ТЕСТ] <b>DATA LAB | СМАРТ-КУПОН</b>\n")
	b.WriteString(fmt.Sprintf("<i>%s · %s · Количественный скринер</i>\n\n", leagueTitle, roundTitle))

	b.WriteString("<pre>┌────────────────────────────────────────────┐\n")
	roundNum := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(cfg.RoundName, "ТУР"), "Тур"))
	headerTitle := fmt.Sprintf(" АНАЛИТИЧЕСКИЙ КУПОН #%s • %s", roundNum, leagueTitle)
	if strings.Contains(strings.ToLower(leagueTitle), "европ") {
		headerTitle = " АНАЛИТИЧЕСКИЙ КУПОН • ЕВРОПА"
	}
	headerPad := 42 - utf8.RuneCountInString(headerTitle)
	if headerPad < 0 {
		headerPad = 0
	}
	b.WriteString(fmt.Sprintf("│%s%s│\n", headerTitle, strings.Repeat(" ", headerPad)))
	b.WriteString("├────────────────────────────────────────────┤\n")

	for _, sel := range plan.Selections {
		m := sel.Match
		hName := m.HomeDisplayName()
		aName := m.AwayDisplayName()
		timeStr := m.StartingAt.UTC().Format("15:04")
		pair := fmt.Sprintf("%s—%s", hName, aName)
		if utf8.RuneCountInString(pair) > 18 {
			pair = string([]rune(pair)[:17]) + "…"
		}
		pairPad := 18 - utf8.RuneCountInString(pair)
		if pairPad < 0 {
			pairPad = 0
		}
		pairPadded := pair + strings.Repeat(" ", pairPad)

		pick := sel.OutcomeShortName()
		pickPad := 7 - utf8.RuneCountInString(pick)
		if pickPad < 0 {
			pickPad = 0
		}
		pickPadded := pick + strings.Repeat(" ", pickPad)

		oddsStr := fmt.Sprintf("%.2f", sel.Odds)

		// Формат: "│ 14:00  Арсенал—Эвертон    П1       1.36 │\n"
		rowContent := fmt.Sprintf(" %s  %s %s %5s ", timeStr, pairPadded, pickPadded, oddsStr)
		rowPad := 44 - utf8.RuneCountInString(rowContent)
		if rowPad < 0 {
			rowPad = 0
		}
		b.WriteString(fmt.Sprintf("│%s%s│\n", rowContent, strings.Repeat(" ", rowPad)))
	}

	b.WriteString("├────────────────────────────────────────────┤\n")
	lineOdds := fmt.Sprintf(" ИТОГОВЫЙ КОЭФФИЦИЕНТ:               %5.2f ", plan.TotalOdds)
	b.WriteString(fmt.Sprintf("│%s%s│\n", lineOdds, strings.Repeat(" ", 44-utf8.RuneCountInString(lineOdds))))

	lineEdge := fmt.Sprintf(" МАТЕМАТИЧЕСКИЙ VALUE EDGE:          +%.1f%% ", plan.ValueEdge)
	b.WriteString(fmt.Sprintf("│%s%s│\n", lineEdge, strings.Repeat(" ", 44-utf8.RuneCountInString(lineEdge))))

	lineStake := " СТРАТЕГИЯ СТАВКИ:                1.0 UNIT "
	b.WriteString(fmt.Sprintf("│%s%s│\n", lineStake, strings.Repeat(" ", 44-utf8.RuneCountInString(lineStake))))
	b.WriteString("└────────────────────────────────────────────┘</pre>\n")

	b.WriteString("<b>Факторы математического перевеса (+EV):</b>\n")
	for _, sel := range plan.Selections {
		m := sel.Match
		hName := m.HomeDisplayName()
		aName := m.AwayDisplayName()
		flag := m.CountryFlag
		if flag != "" {
			flag += " "
		}
		b.WriteString(fmt.Sprintf("• <b>%s%s — %s</b>: %s\n", flag, hName, aName, sel.Reason))
	}
	b.WriteString("\n<i>Верификация исходов фиксируется в системе. Открытая история в Mini App.</i>")

	var footers2 []string
	if cfg.BookmakerURL != "" {
		bk := cfg.BookmakerName
		if bk == "" {
			bk = "FONBET"
		}
		footers2 = append(footers2, fmt.Sprintf("<a href=\"%s\">Фрибет %s</a>", cfg.BookmakerURL, bk))
	}
	if cfg.MiniAppURL != "" {
		footers2 = append(footers2, fmt.Sprintf("<a href=\"%s\">Все матчи</a>", cfg.MiniAppURL))
	}
	if len(footers2) > 0 {
		b.WriteString("\n\n" + strings.Join(footers2, " · "))
	}

	return b.String()
}

// GenerateExpressPostStyle3Caption формирует стиль [3 ТЕСТ]: Подпись к визуальному графическому билету-постеру.
func GenerateExpressPostStyle3Caption(cfg ExpressPostConfig, plan *analytics.SmartExpressPlan) string {
	if plan == nil || len(plan.Selections) == 0 {
		return ""
	}

	var b strings.Builder

	leagueTitle := cfg.LeagueName
	if leagueTitle == "" {
		leagueTitle = "Топ-чемпионаты Европы"
	}
	roundTitle := cfg.RoundName
	if roundTitle != "" && !strings.Contains(strings.ToLower(roundTitle), "тур") {
		roundTitle = fmt.Sprintf("Тур %s", roundTitle)
	}

	b.WriteString(fmt.Sprintf("[3 ТЕСТ] 🎫 <b>СМАРТ-КУПОН ТУРА: КОЭФФИЦИЕНТ %.2f</b>\n", plan.TotalOdds))
	b.WriteString(fmt.Sprintf("<i>Математический экспресс · %s · %s</i>\n\n", leagueTitle, roundTitle))

	b.WriteString(fmt.Sprintf("• Модельный перевес (EV): <b>+%.1f%%</b>\n", plan.ValueEdge))
	b.WriteString(fmt.Sprintf("• Рекомендуемый банк-менеджмент: <b>%.1f unit (флэт)</b>\n", plan.RecommendedStake))
	b.WriteString(fmt.Sprintf("• Оценка надежности алгоритма: <b>%.1f / 10</b>\n\n", plan.Confidence))

	bkName := cfg.BookmakerName
	if bkName == "" {
		bkName = "FONBET"
	}
	promo := cfg.PromoCode
	if promo == "" {
		promo = "FDL"
	}
	b.WriteString(fmt.Sprintf("🎁 <i>Фрибет по промокоду <code>%s</code> в %s.</i>\n", promo, bkName))
	b.WriteString("<i>Все исходы и коэффициенты верифицируются в реестре Data Lab.</i>")

	var footers3 []string
	if cfg.BookmakerURL != "" {
		footers3 = append(footers3, fmt.Sprintf("<a href=\"%s\">Фрибет %s</a>", cfg.BookmakerURL, bkName))
	}
	if cfg.MiniAppURL != "" {
		footers3 = append(footers3, fmt.Sprintf("<a href=\"%s\">Все матчи</a>", cfg.MiniAppURL))
	}
	if len(footers3) > 0 {
		b.WriteString("\n\n" + strings.Join(footers3, " · "))
	}

	return b.String()
}

// GenerateExpressPostHTML формирует HTML-текст по умолчанию (стиль 1).
func GenerateExpressPostHTML(cfg ExpressPostConfig, plan *analytics.SmartExpressPlan) string {
	return GenerateExpressPostStyle1(cfg, plan)
}
