package copywriter

import (
	"fmt"
	"math"
	"strings"

	"github.com/Pastir/footballdata/analytics"
)

// GenerateBanterPostHTML формирует остроумный текст для поста в Telegram с карточкой аномалии
func GenerateBanterPostHTML(a analytics.MatchAnomaly, miniAppURL string) string {
	if miniAppURL == "" {
		miniAppURL = "https://t.me/football_data_lab_bot/data_lab"
	}
	if !strings.HasPrefix(miniAppURL, "http://") && !strings.HasPrefix(miniAppURL, "https://") {
		miniAppURL = "https://" + miniAppURL
	}

	hXG := 0.0
	if a.HomeXG != nil {
		hXG = *a.HomeXG
	}
	aXG := 0.0
	if a.AwayXG != nil {
		aXG = *a.AwayXG
	}

	switch a.Type {
	case analytics.AnomalyTypeXGRobbery:
		winner := a.TeamAwayName
		loser := a.TeamHomeName
		winnerXG := aXG
		if a.ScoreHome > a.ScoreAway {
			winner = a.TeamHomeName
			loser = a.TeamAwayName
			winnerXG = hXG
		}

		return fmt.Sprintf(
			`🚌 <b>ОГРАБЛЕНИЕ ТУРА: КПД 1000%%</b>

<b>%s %d : %d %s</b>

%s понадобилось всего <b>%.2f xG</b> и святой дух вратаря, чтобы наказать %s и увезти очки. Футбольная справедливость временно покинула стадион.

📊 <b>Анатомия матча:</b>
• xG: <b>%.2f vs %.2f</b>
• Удары: <b>%d vs %d</b>
• Владение: <b>%d%% vs %d%%</b>
• Сейвы: <b>%d vs %d</b>

👉 <a href="%s">Смотреть подробный разбор матча в Mini App</a>`,
			a.TeamHomeName, a.ScoreHome, a.ScoreAway, a.TeamAwayName,
			winner, winnerXG, loser,
			hXG, aXG,
			a.HomeShots, a.AwayShots,
			a.HomePossession, a.AwayPossession,
			a.HomeSaves, a.AwaySaves,
			miniAppURL,
		)

	case analytics.AnomalyTypeLateDrama:
		if a.BadgeText == "Камбэк тура" {
			return fmt.Sprintf(
				`⚡ <b>СУПЕРКАМБЭК ТУРА: ВЫРВАЛИ ПОБЕДУ В КОНЦОВКЕ!</b>

<b>%s %d : %d %s</b>

%s

📊 <b>Анатомия камбэка:</b>
• xG: <b>%.2f vs %.2f</b>
• Удары: <b>%d vs %d</b>
• Владение: <b>%d%% vs %d%%</b>
• Развязка: <b>%s</b>

Никогда не уходите со стадиона и не выключайте трансляцию до финального свистка!

👉 <a href="%s">Смотреть подробный разбор матча в Mini App</a>`,
				a.TeamHomeName, a.ScoreHome, a.ScoreAway, a.TeamAwayName,
				a.FooterQuote,
				hXG, aXG,
				a.HomeShots, a.AwayShots,
				a.HomePossession, a.AwayPossession,
				a.AnomalySummary,
				miniAppURL,
			)
		}

		if a.BadgeText == "Спасение 90+" {
			return fmt.Sprintf(
				`🧊 <b>СПАСЕНИЕ ТУРА: НИЧЬЯ НА ПОСЛЕДНИХ СЕКУНДАХ!</b>

<b>%s %d : %d %s</b>

%s

📊 <b>Анатомия драмы:</b>
• xG: <b>%.2f vs %.2f</b>
• Удары: <b>%d vs %d</b>
• Владение: <b>%d%% vs %d%%</b>
• Исход: <b>%s</b>

👉 <a href="%s">Смотреть подробный разбор матча в Mini App</a>`,
				a.TeamHomeName, a.ScoreHome, a.ScoreAway, a.TeamAwayName,
				a.FooterQuote,
				hXG, aXG,
				a.HomeShots, a.AwayShots,
				a.HomePossession, a.AwayPossession,
				a.AnomalySummary,
				miniAppURL,
			)
		}

		return fmt.Sprintf(
			`⏱️ <b>ВАЛИДОЛ ТУРА: РАЗВЯЗКА В КОНЦОВКЕ!</b>

<b>%s %d : %d %s</b>

%s

📊 <b>Цифры триллера:</b>
• xG: <b>%.2f vs %.2f</b>
• Удары: <b>%d vs %d</b>
• Владение: <b>%d%% vs %d%%</b>
• Исход: <b>%s</b>

Никогда не выключайте трансляцию до свистка судьи! 

👉 <a href="%s">Смотреть график атак матча в Mini App</a>`,
			a.TeamHomeName, a.ScoreHome, a.ScoreAway, a.TeamAwayName,
			a.FooterQuote,
			hXG, aXG,
			a.HomeShots, a.AwayShots,
			a.HomePossession, a.AwayPossession,
			a.AnomalySummary,
			miniAppURL,
		)

	case analytics.AnomalyTypeSuperSub:
		return fmt.Sprintf(
			`🃏 <b>СУПЕР-ДЖОКЕР ТУРА: ВЫШЕЛ И РЕШИЛ</b>

<b>%s («%s»)</b>

Матч: <b>%s %d : %d %s</b>

Тренер бросил в бой свежие силы со скамейки — и игрок перевернул матч в одиночку: <b>%s</b>! Премия за идеальное чутье тренерского штаба.

👉 <a href="%s">Рейтинг игроков и оценки тура в Mini App</a>`,
			a.HeroName, a.HeroTeam,
			a.TeamHomeName, a.ScoreHome, a.ScoreAway, a.TeamAwayName,
			a.HeroStatBig,
			miniAppURL,
		)

	case analytics.AnomalyTypeGoalkeeperWall:
		saves := a.HomeSaves
		keeperTeam := a.TeamHomeName
		if a.AwaySaves > a.HomeSaves {
			saves = a.AwaySaves
			keeperTeam = a.TeamAwayName
		}

		return fmt.Sprintf(
			`🐙 <b>ВРАТАРЬ-ОСЬМИНОГ: МАСТЕРКЛАСС ТУРА</b>

<b>%s %d : %d %s</b>

Настоящий бенефис голкипера: <b>%d сейвов</b> за 90 минут! Пока полевые игроки трещали под шквалом атак, страж ворот «%s» тащил буквально всё.

📊 <b>Давление на ворота:</b>
• Удары соперника: <b>%d</b>
• Сейвы: <b>%d</b>
• Спасено от xG: <b>%.2f</b>

👉 <a href="%s">Рейтинг вратарей сезона в Mini App</a>`,
			a.TeamHomeName, a.ScoreHome, a.ScoreAway, a.TeamAwayName,
			saves, keeperTeam,
			a.HomeShots+a.AwayShots-saves,
			saves,
			math.Max(hXG, aXG),
			miniAppURL,
		)

	default:
		return fmt.Sprintf(
			`🔥 <b>СОБЫТИЕ ТУРА: %s</b>

<b>%s %d : %d %s</b>

%s

👉 <a href="%s">Смотреть подробности в Mini App</a>`,
			strings.ToUpper(a.BadgeText),
			a.TeamHomeName, a.ScoreHome, a.ScoreAway, a.TeamAwayName,
			a.FooterQuote,
			miniAppURL,
		)
	}
}
