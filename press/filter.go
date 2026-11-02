package press

import (
	"regexp"
	"strings"
)

var (
	htmlTagRegex  = regexp.MustCompile(`<[^>]*>`)
	spaceCollapse = regexp.MustCompile(`\s+`)
)

// CleanText удаляет HTML-теги, лишние пробелы и артефакты из текста лида
func CleanText(raw string) string {
	clean := htmlTagRegex.ReplaceAllString(raw, " ")
	clean = strings.ReplaceAll(clean, "&nbsp;", " ")
	clean = strings.ReplaceAll(clean, "&#039;", "'")
	clean = strings.ReplaceAll(clean, "&quot;", "\"")
	clean = strings.ReplaceAll(clean, "&amp;", "&")
	clean = spaceCollapse.ReplaceAllString(clean, " ")
	return strings.TrimSpace(clean)
}

// CalculateHypeScore вычисляет оценку кликабельности и важности новости (0-100)
func CalculateHypeScore(title, lead string) int {
	fullText := strings.ToLower(title + " " + lead)

	// 1. Негативные фильтры (сразу отсекаем информационный шум)
	negativeKeywords := []string{
		"women", "women's", "femenino", "femenina", "liga f", "wsl",
		"u21", "u19", "u18", "u17", "academy", "youth team",
		"quiz", "trivia", "player ratings", "predicted lineup",
	}
	for _, kw := range negativeKeywords {
		if strings.Contains(fullText, kw) {
			return 0 // Полный сброс, новость пропускается
		}
	}

	score := 10 // Базовый балл за факт публикации в топ-таблоиде

	// 2. Топ-клубы Европы (+30 баллов)
	topClubs := []string{
		"real madrid", "barcelona", "manchester city", "man city",
		"liverpool", "arsenal", "manchester united", "man utd",
		"chelsea", "bayern", "psg", "paris saint-germain",
		"juventus", "inter", "atletico madrid", "dortmund", "bayer leverkusen",
	}
	for _, club := range topClubs {
		if strings.Contains(fullText, club) {
			score += 30
			break // Один раз за клуб
		}
	}

	// 3. Топ-игроки и супер-тренеры (+25 баллов)
	topPersonalities := []string{
		"haaland", "mbappe", "vinicius", "bellingham", "yamal",
		"guardiola", "arteta", "klopp", "mourinho", "ancelotti",
		"salah", "palmer", "rodri", "flick", "alonso", "romano",
	}
	for _, p := range topPersonalities {
		if strings.Contains(fullText, p) {
			score += 25
			break
		}
	}

	// 4. Громкие события
	// Отставки тренеров (+40)
	sackingKeywords := []string{"sacked", "fired", "resigns", "resignation", "part ways", "destituido", "entlassen", "dismissed"}
	for _, kw := range sackingKeywords {
		if strings.Contains(fullText, kw) {
			score += 40
			break
		}
	}

	// Трансферные бомбы (+35)
	transferKeywords := []string{"here we go", "agreed terms", "bid accepted", "release clause", "fichaje", "contract agreed", "signing", "record bid"}
	for _, kw := range transferKeywords {
		if strings.Contains(fullText, kw) {
			score += 35
			break
		}
	}

	// Скандалы, расследования и баны (+30)
	scandalKeywords := []string{"investigation", "banned", "ban", "sanction", "police", "charged", "appeal rejected", "var leak", "brawl"}
	for _, kw := range scandalKeywords {
		if strings.Contains(fullText, kw) {
			score += 30
			break
		}
	}

	// Тяжелые травмы лидеров (+20)
	injuryKeywords := []string{"acl", "rupture", "surgery", "ruled out for season", "months out"}
	for _, kw := range injuryKeywords {
		if strings.Contains(fullText, kw) {
			score += 20
			break
		}
	}

	if score > 100 {
		score = 100
	}
	return score
}

