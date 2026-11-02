package analytics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
)

// AnomalyType определяет категорию футбольной аномалии
type AnomalyType string

const (
	AnomalyTypeXGRobbery      AnomalyType = "xg_robbery"      // Ограбление тура / Автобус недели
	AnomalyTypeLateDrama      AnomalyType = "late_drama"      // Валидол 90+ / Вырванная победа
	AnomalyTypeSuperSub       AnomalyType = "super_sub"       // Джокер тура / Супер-саб
	AnomalyTypeEfficiencyHero AnomalyType = "efficiency_hero" // Человек-КПД / Хет-трик или дубль
	AnomalyTypeGoalkeeperWall AnomalyType = "goalkeeper_wall" // Вратарь-Стена / Мастеркласс сейвов
)

// MatchAnomaly описывает обнаруженную аномалию или драму в сыгранном матче
type MatchAnomaly struct {
	Type          AnomalyType
	BadgeText     string  // «Ограбление тура», «Валидол 90+», «Джокер тура»
	BadgeClass    string  // Tailwind-классы для бейджа
	TagText       string  // «🚌 Автобус высшей категории», «⏰ Магия Neverkusen»
	SeverityScore float64 // Оценка силы аномалии (0–100) для ранжирования лучшего события тура

	// Данные матча
	FixtureID  int64
	LeagueID   int
	LeagueName string
	RoundID    int
	RoundName  string

	TeamHomeName string
	TeamHomeLogo string
	TeamAwayName string
	TeamAwayLogo string
	ScoreHome    int
	ScoreAway    int

	// Статистика
	HomeXG         *float64
	AwayXG         *float64
	HomePossession int
	AwayPossession int
	HomeShots      int
	AwayShots      int
	HomeSaves      int
	AwaySaves      int

	// Герой матча (для карточек игроков)
	IsHeroPlayer  bool
	HeroName      string
	HeroTeam      string
	HeroAvatar    string
	HeroStatBig   string // «2 ГОЛА», «11 СЕЙВОВ», «ХЕТ-ТРИК»
	HeroStatSmall string // «за 14 минут со скамейки»

	// Оформление
	FooterIcon     string
	FooterQuote    string
	AnomalySummary string
}

// FindRoundAnomalies сканирует сыгранные матчи тура и возвращает обнаруженные аномалии
func FindRoundAnomalies(ctx context.Context, pool *pgxpool.Pool, leagueID int, roundID int) ([]MatchAnomaly, error) {
	info, err := GetLeagueSeasonInfo(ctx, pool, leagueID)
	if err != nil {
		return nil, fmt.Errorf("get league info: %w", err)
	}

	var matches []RoundResultMatch
	if roundID > 0 {
		m, _, err := GetRoundResultsByRoundID(ctx, pool, roundID, "ru")
		if err != nil {
			return nil, fmt.Errorf("get round results by id %d: %w", roundID, err)
		}
		matches = m
	} else {
		m, err := GetRoundResults(ctx, pool, &info)
		if err != nil {
			return nil, fmt.Errorf("get round results: %w", err)
		}
		matches = m
	}

	var anomalies []MatchAnomaly

	for _, m := range matches {
		if !isFinished(m) || m.Scores.HomeCurrent == nil || m.Scores.AwayCurrent == nil {
			continue
		}

		// 1. Проверка на Ограбление по xG (Автобус)
		if a, ok := detectXGRobbery(m, info); ok {
			anomalies = append(anomalies, a)
		}

		// 2. Проверка на Драму на 90+ минуте
		if a, ok := detectLateDrama(m, info); ok {
			anomalies = append(anomalies, a)
		}

		// 3. Проверка на Джокера / Супер-саба
		if a, ok := detectSuperSub(m, info); ok {
			anomalies = append(anomalies, a)
		}

		// 4. Проверка на Вратаря-Стену
		if a, ok := detectGoalkeeperWall(m, info); ok {
			anomalies = append(anomalies, a)
		}
	}

	// Сортируем по силе аномалии (SeverityScore) descending
	sort.Slice(anomalies, func(i, j int) bool {
		return anomalies[i].SeverityScore > anomalies[j].SeverityScore
	})

	return anomalies, nil
}

// detectXGRobbery выявляет матчи, где фаворит доминировал по xG и владению, но не выиграл
func detectXGRobbery(m RoundResultMatch, info LeagueSeasonInfo) (MatchAnomaly, bool) {
	if m.HomeXG == nil || m.AwayXG == nil {
		return MatchAnomaly{}, false
	}

	hXG := *m.HomeXG
	aXG := *m.AwayXG
	hScore := *m.Scores.HomeCurrent
	aScore := *m.Scores.AwayCurrent

	hPoss := extractStatInt(m.Stats, "possession", true)
	aPoss := extractStatInt(m.Stats, "possession", false)
	hShots := extractStatInt(m.Stats, "shots_total", true)
	aShots := extractStatInt(m.Stats, "shots_total", false)
	hSaves := extractStatInt(m.Stats, "saves", true)
	aSaves := extractStatInt(m.Stats, "saves", false)

	diffXG := math.Abs(hXG - aXG)

	// Ситуация А: Хозяева имели перевес по xG >= 1.7, но победили гости
	if (hXG-aXG) >= 1.7 && hScore < aScore {
		severity := 60.0 + (hXG-aXG)*12.0
		quote := fmt.Sprintf("«%s нанесли %d ударов, но %s увезли победу с %.2f xG»", m.TeamHomeName, hShots, m.TeamAwayName, aXG)

		return MatchAnomaly{
			Type:           AnomalyTypeXGRobbery,
			BadgeText:      "Ограбление тура",
			BadgeClass:     "bg-rose-500/20 text-rose-400 border-rose-500/40",
			TagText:        "🚌 Автобус высшей категории",
			SeverityScore:  severity,
			FixtureID:      m.FixtureID,
			LeagueID:       info.LeagueID,
			LeagueName:     info.LeagueName,
			RoundID:        m.RoundID,
			RoundName:      m.RoundName,
			TeamHomeName:   m.TeamHomeName,
			TeamHomeLogo:   m.TeamHomeLogo,
			TeamAwayName:   m.TeamAwayName,
			TeamAwayLogo:   m.TeamAwayLogo,
			ScoreHome:      hScore,
			ScoreAway:      aScore,
			HomeXG:         m.HomeXG,
			AwayXG:         m.AwayXG,
			HomePossession: hPoss,
			AwayPossession: aPoss,
			HomeShots:      hShots,
			AwayShots:      aShots,
			HomeSaves:      hSaves,
			AwaySaves:      aSaves,
			FooterIcon:     "⚖️",
			FooterQuote:    quote,
			AnomalySummary: fmt.Sprintf("+%.2f xG разница", diffXG),
		}, true
	}

	// Ситуация Б: Гости имели перевес по xG >= 1.7, но победили хозяева
	if (aXG-hXG) >= 1.7 && aScore < hScore {
		severity := 60.0 + (aXG-hXG)*12.0
		quote := fmt.Sprintf("«%s нанесли %d ударов, но %s увезли победу с %.2f xG»", m.TeamAwayName, aShots, m.TeamHomeName, hXG)

		return MatchAnomaly{
			Type:           AnomalyTypeXGRobbery,
			BadgeText:      "Ограбление тура",
			BadgeClass:     "bg-rose-500/20 text-rose-400 border-rose-500/40",
			TagText:        "🚌 Автобус высшей категории",
			SeverityScore:  severity,
			FixtureID:      m.FixtureID,
			LeagueID:       info.LeagueID,
			LeagueName:     info.LeagueName,
			RoundID:        m.RoundID,
			RoundName:      m.RoundName,
			TeamHomeName:   m.TeamHomeName,
			TeamHomeLogo:   m.TeamHomeLogo,
			TeamAwayName:   m.TeamAwayName,
			TeamAwayLogo:   m.TeamAwayLogo,
			ScoreHome:      hScore,
			ScoreAway:      aScore,
			HomeXG:         m.HomeXG,
			AwayXG:         m.AwayXG,
			HomePossession: hPoss,
			AwayPossession: aPoss,
			HomeShots:      hShots,
			AwayShots:      aShots,
			HomeSaves:      hSaves,
			AwaySaves:      aSaves,
			FooterIcon:     "⚖️",
			FooterQuote:    quote,
			AnomalySummary: fmt.Sprintf("+%.2f xG разница", diffXG),
		}, true
	}

	// Ситуация В: Грандиозный перевес по xG >= 2.2, но глухая ничья
	if diffXG >= 2.2 && hScore == aScore {
		severity := 55.0 + diffXG*10.0
		dominator := m.TeamHomeName
		if aXG > hXG {
			dominator = m.TeamAwayName
		}
		quote := fmt.Sprintf("«%s создали %.2f xG, но потеряли очки в глухой обороне»", dominator, math.Max(hXG, aXG))

		return MatchAnomaly{
			Type:           AnomalyTypeXGRobbery,
			BadgeText:      "Сенсация тура",
			BadgeClass:     "bg-amber-500/20 text-amber-400 border-amber-500/40",
			TagText:        "🛡️ Железный забор",
			SeverityScore:  severity,
			FixtureID:      m.FixtureID,
			LeagueID:       info.LeagueID,
			LeagueName:     info.LeagueName,
			RoundID:        m.RoundID,
			RoundName:      m.RoundName,
			TeamHomeName:   m.TeamHomeName,
			TeamHomeLogo:   m.TeamHomeLogo,
			TeamAwayName:   m.TeamAwayName,
			TeamAwayLogo:   m.TeamAwayLogo,
			ScoreHome:      hScore,
			ScoreAway:      aScore,
			HomeXG:         m.HomeXG,
			AwayXG:         m.AwayXG,
			HomePossession: hPoss,
			AwayPossession: aPoss,
			HomeShots:      hShots,
			AwayShots:      aShots,
			HomeSaves:      hSaves,
			AwaySaves:      aSaves,
			FooterIcon:     "🧱",
			FooterQuote:    quote,
			AnomalySummary: fmt.Sprintf("+%.2f xG разница", diffXG),
		}, true
	}

	return MatchAnomaly{}, false
}

func pluralMinutes(n int) string {
	if n%10 == 1 && n%100 != 11 {
		return fmt.Sprintf("%d минуту", n)
	}
	if (n%10 >= 2 && n%10 <= 4) && !(n%100 >= 12 && n%100 <= 14) {
		return fmt.Sprintf("%d минуты", n)
	}
	return fmt.Sprintf("%d минут", n)
}

func pluralGoals(n int) string {
	if n%10 == 1 && n%100 != 11 {
		return fmt.Sprintf("%d гол", n)
	}
	if (n%10 >= 2 && n%10 <= 4) && !(n%100 >= 12 && n%100 <= 14) {
		return fmt.Sprintf("%d гола", n)
	}
	return fmt.Sprintf("%d голов", n)
}

func effectiveMinute(g goalDetail) int {
	return g.minute + g.extraMinute
}

type goalDetail struct {
	minute      int
	extraMinute int
	minuteStr   string
	teamID      int64
	player      string
	scoreH      int
	scoreA      int
}

// detectLateDrama выявляет волевые камбэки в концовке, победные голы и спасения на 88+ и 90+ минутах
func detectLateDrama(m RoundResultMatch, info LeagueSeasonInfo) (MatchAnomaly, bool) {
	if len(m.Events) == 0 || m.Scores.HomeCurrent == nil || m.Scores.AwayCurrent == nil {
		return MatchAnomaly{}, false
	}

	hScore := *m.Scores.HomeCurrent
	aScore := *m.Scores.AwayCurrent

	var goals []goalDetail
	curH := 0
	curA := 0

	for _, ev := range m.Events {
		evType := strings.ToUpper(ev.Type)
		if !strings.Contains(evType, "GOAL") {
			continue
		}

		extra := 0
		if ev.ExtraMinute != nil {
			extra = *ev.ExtraMinute
		}

		mStr := fmt.Sprintf("%d'", ev.Minute)
		if extra > 0 {
			mStr = fmt.Sprintf("90+%d'", extra)
		}

		if strings.Contains(ev.Result, "-") {
			parts := strings.Split(ev.Result, "-")
			if len(parts) == 2 {
				h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				a, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil {
					curH = h
					curA = a
				}
			}
		} else {
			if ev.TeamID == m.TeamHomeID {
				curH++
			} else if ev.TeamID == m.TeamAwayID {
				curA++
			}
		}

		goals = append(goals, goalDetail{
			minute:      ev.Minute,
			extraMinute: extra,
			minuteStr:   mStr,
			teamID:      ev.TeamID,
			player:      ev.Player,
			scoreH:      curH,
			scoreA:      curA,
		})
	}

	if len(goals) == 0 {
		return MatchAnomaly{}, false
	}

	// Ищем первый поздний гол (начиная с 85-й минуты)
	firstLateIdx := -1
	for i, g := range goals {
		if g.minute >= 85 {
			firstLateIdx = i
			break
		}
	}
	if firstLateIdx == -1 {
		return MatchAnomaly{}, false
	}

	scoreBeforeH := 0
	scoreBeforeA := 0
	if firstLateIdx > 0 {
		scoreBeforeH = goals[firstLateIdx-1].scoreH
		scoreBeforeA = goals[firstLateIdx-1].scoreA
	}

	lateGoals := goals[firstLateIdx:]
	lastGoal := lateGoals[len(lateGoals)-1]

	var lateMinutes []string
	for _, lg := range lateGoals {
		lateMinutes = append(lateMinutes, lg.minuteStr)
	}
	minutesJoined := strings.Join(lateMinutes, ", ")

	hPoss := extractStatInt(m.Stats, "possession", true)
	aPoss := extractStatInt(m.Stats, "possession", false)
	hShots := extractStatInt(m.Stats, "shots_total", true)
	aShots := extractStatInt(m.Stats, "shots_total", false)
	hSaves := extractStatInt(m.Stats, "saves", true)
	aSaves := extractStatInt(m.Stats, "saves", false)

	// 1. Сценарий: ВОЛЕВОЙ СУПЕРКАМБЭК В КОНЦОВКЕ (Losing before late goals, won at full time)
	isHomeComeback := scoreBeforeH < scoreBeforeA && hScore > aScore
	isAwayComeback := scoreBeforeA < scoreBeforeH && aScore > hScore

	if isHomeComeback || isAwayComeback {
		winnerName := m.TeamHomeName
		winnerTeamID := m.TeamHomeID
		losingScore := fmt.Sprintf("%d:%d", scoreBeforeH, scoreBeforeA)

		if isAwayComeback {
			winnerName = m.TeamAwayName
			winnerTeamID = m.TeamAwayID
			losingScore = fmt.Sprintf("%d:%d", scoreBeforeA, scoreBeforeH)
		}

		var winnerLateGoals []goalDetail
		for _, lg := range lateGoals {
			if lg.teamID == winnerTeamID {
				winnerLateGoals = append(winnerLateGoals, lg)
			}
		}

		quote := ""
		tag := "⚡ Волевой камбэк"
		summary := ""

		if len(winnerLateGoals) >= 2 {
			firstMin := effectiveMinute(winnerLateGoals[0])
			lastMin := effectiveMinute(winnerLateGoals[len(winnerLateGoals)-1])
			spanMinutes := lastMin - firstMin
			if spanMinutes <= 0 {
				spanMinutes = 1
			}

			quote = fmt.Sprintf("«%s» уступал %s до %s минуты, но забил %s за %s (%s) и вырвал невероятную победу!",
				winnerName, losingScore, winnerLateGoals[0].minuteStr, pluralGoals(len(winnerLateGoals)), pluralMinutes(spanMinutes), minutesJoined)
			tag = fmt.Sprintf("⚡ Камбэк за %s!", pluralMinutes(spanMinutes))
			summary = fmt.Sprintf("Камбэк с %s за %s (%s)", losingScore, pluralMinutes(spanMinutes), lastGoal.minuteStr)
		} else {
			quote = fmt.Sprintf("«%s» уступал %s в концовке, но переломил ход встречи (%s) и вырвал победу!",
				winnerName, losingScore, minutesJoined)
			summary = fmt.Sprintf("Камбэк с %s: победа на %s", losingScore, lastGoal.minuteStr)
		}

		return MatchAnomaly{
			Type:           AnomalyTypeLateDrama,
			BadgeText:      "Камбэк тура",
			BadgeClass:     "bg-amber-500/20 text-amber-400 border-amber-500/40",
			TagText:        tag,
			SeverityScore:  98.0,
			FixtureID:      m.FixtureID,
			LeagueID:       info.LeagueID,
			LeagueName:     info.LeagueName,
			RoundID:        m.RoundID,
			RoundName:      m.RoundName,
			TeamHomeName:   m.TeamHomeName,
			TeamHomeLogo:   m.TeamHomeLogo,
			TeamAwayName:   m.TeamAwayName,
			TeamAwayLogo:   m.TeamAwayLogo,
			ScoreHome:      hScore,
			ScoreAway:      aScore,
			HomeXG:         m.HomeXG,
			AwayXG:         m.AwayXG,
			HomePossession: hPoss,
			AwayPossession: aPoss,
			HomeShots:      hShots,
			AwayShots:      aShots,
			HomeSaves:      hSaves,
			AwaySaves:      aSaves,
			FooterIcon:     "⚡",
			FooterQuote:    quote,
			AnomalySummary: summary,
		}, true
	}

	// 2. Сценарий: ПОБЕДНЫЙ ГОЛ В КОНЦОВКЕ (Tied before late goals, won in dying minutes)
	if scoreBeforeH == scoreBeforeA && hScore != aScore && (lastGoal.minute >= 88 || lastGoal.extraMinute > 0) {
		winnerName := m.TeamHomeName
		if aScore > hScore {
			winnerName = m.TeamAwayName
		}
		severity := 85.0 + float64(lastGoal.extraMinute)*2.0
		quote := fmt.Sprintf("«Победный гол на %s минуте матча решил исход триллера в пользу %s»", lastGoal.minuteStr, winnerName)

		return MatchAnomaly{
			Type:           AnomalyTypeLateDrama,
			BadgeText:      "Валидол 90+",
			BadgeClass:     "bg-rose-500/20 text-rose-400 border-rose-500/40",
			TagText:        fmt.Sprintf("⏰ Развязка на %s", lastGoal.minuteStr),
			SeverityScore:  severity,
			FixtureID:      m.FixtureID,
			LeagueID:       info.LeagueID,
			LeagueName:     info.LeagueName,
			RoundID:        m.RoundID,
			RoundName:      m.RoundName,
			TeamHomeName:   m.TeamHomeName,
			TeamHomeLogo:   m.TeamHomeLogo,
			TeamAwayName:   m.TeamAwayName,
			TeamAwayLogo:   m.TeamAwayLogo,
			ScoreHome:      hScore,
			ScoreAway:      aScore,
			HomeXG:         m.HomeXG,
			AwayXG:         m.AwayXG,
			HomePossession: hPoss,
			AwayPossession: aPoss,
			HomeShots:      hShots,
			AwayShots:      aShots,
			HomeSaves:      hSaves,
			AwaySaves:      aSaves,
			FooterIcon:     "🔥",
			FooterQuote:    quote,
			AnomalySummary: fmt.Sprintf("Победный гол на %s минуте", lastGoal.minuteStr),
		}, true
	}

	// 3. Сценарий: СПАСЕНИЕ НА ФЛАЖКЕ (Losing before late goals, equalized at 88'+)
	if (scoreBeforeH < scoreBeforeA || scoreBeforeA < scoreBeforeH) && hScore == aScore && (lastGoal.minute >= 88 || lastGoal.extraMinute > 0) {
		saverName := m.TeamHomeName
		if scoreBeforeA < scoreBeforeH {
			saverName = m.TeamAwayName
		}
		quote := fmt.Sprintf("«Спасительный гол на %s минуте вырвал ничью для %s на последних секундах!»", lastGoal.minuteStr, saverName)

		return MatchAnomaly{
			Type:           AnomalyTypeLateDrama,
			BadgeText:      "Спасение 90+",
			BadgeClass:     "bg-cyan-500/20 text-cyan-400 border-cyan-500/40",
			TagText:        fmt.Sprintf("🧊 Спасение на %s", lastGoal.minuteStr),
			SeverityScore:  86.0,
			FixtureID:      m.FixtureID,
			LeagueID:       info.LeagueID,
			LeagueName:     info.LeagueName,
			RoundID:        m.RoundID,
			RoundName:      m.RoundName,
			TeamHomeName:   m.TeamHomeName,
			TeamHomeLogo:   m.TeamHomeLogo,
			TeamAwayName:   m.TeamAwayName,
			TeamAwayLogo:   m.TeamAwayLogo,
			ScoreHome:      hScore,
			ScoreAway:      aScore,
			HomeXG:         m.HomeXG,
			AwayXG:         m.AwayXG,
			HomePossession: hPoss,
			AwayPossession: aPoss,
			HomeShots:      hShots,
			AwayShots:      aShots,
			HomeSaves:      hSaves,
			AwaySaves:      aSaves,
			FooterIcon:     "🧊",
			FooterQuote:    quote,
			AnomalySummary: fmt.Sprintf("Гол на %s спас ничью", lastGoal.minuteStr),
		}, true
	}

	return MatchAnomaly{}, false
}

func isFinished(m RoundResultMatch) bool {
	if m.StateShort == nil {
		return false
	}
	st := *m.StateShort
	return st == "FT" || st == "AET" || st == "FT_PEN"
}

// detectSuperSub выявляет игроков, оформивших дубль после выхода на замену
func detectSuperSub(m RoundResultMatch, info LeagueSeasonInfo) (MatchAnomaly, bool) {
	// Подсчитываем голы каждого игрока
	playerGoals := make(map[string]int)
	playerTeam := make(map[string]int64)

	for _, ev := range m.Events {
		evType := strings.ToUpper(ev.Type)
		if strings.Contains(evType, "GOAL") && ev.Player != "" {
			playerGoals[ev.Player]++
			if ev.TeamID > 0 {
				playerTeam[ev.Player] = ev.TeamID
			}
		}
	}

	// Ищем игрока с 2+ голами
	for playerName, goals := range playerGoals {
		if goals >= 2 {
			// Проверяем, был ли игрок запасным
			isSub := false
			var playerImg string

			for _, players := range m.Lineups {
				for _, p := range players {
					if strings.EqualFold(p.Name, playerName) {
						if !p.IsStarter {
							isSub = true
						}
						playerImg = p.ImagePath
						break
					}
				}
			}

			// Если запасной забил 2+ гола — это супер-саб!
			if isSub {
				teamName := m.TeamHomeName
				teamLogo := m.TeamHomeLogo
				if playerTeam[playerName] == m.TeamAwayID {
					teamName = m.TeamAwayName
					teamLogo = m.TeamAwayLogo
				}
				if playerImg == "" {
					playerImg = teamLogo
				}

				return MatchAnomaly{
					Type:           AnomalyTypeSuperSub,
					BadgeText:      "Джокер тура",
					BadgeClass:     "bg-emerald-500/20 text-emerald-400 border-emerald-500/40",
					TagText:        "⚡ Супер-замена",
					SeverityScore:  85.0 + float64(goals)*5.0,
					FixtureID:      m.FixtureID,
					LeagueID:       info.LeagueID,
					LeagueName:     info.LeagueName,
					RoundID:        m.RoundID,
					RoundName:      m.RoundName,
					TeamHomeName:   m.TeamHomeName,
					TeamHomeLogo:   m.TeamHomeLogo,
					TeamAwayName:   m.TeamAwayName,
					TeamAwayLogo:   m.TeamAwayLogo,
					ScoreHome:      *m.Scores.HomeCurrent,
					ScoreAway:      *m.Scores.AwayCurrent,
					HomeXG:         m.HomeXG,
					AwayXG:         m.AwayXG,
					HomePossession: extractStatInt(m.Stats, "possession", true),
					AwayPossession: extractStatInt(m.Stats, "possession", false),
					HomeShots:      extractStatInt(m.Stats, "shots_total", true),
					AwayShots:      extractStatInt(m.Stats, "shots_total", false),
					HomeSaves:      extractStatInt(m.Stats, "saves", true),
					AwaySaves:      extractStatInt(m.Stats, "saves", false),
					IsHeroPlayer:   true,
					HeroName:       playerName,
					HeroTeam:       teamName,
					HeroAvatar:     playerImg,
					HeroStatBig:    fmt.Sprintf("%d ГОЛА", goals),
					HeroStatSmall:  "вышел со скамейки запасных",
					FooterIcon:     "🃏",
					FooterQuote:    fmt.Sprintf("«Идеальный выход на замену: %s решил судьбу встречи»", playerName),
					AnomalySummary: fmt.Sprintf("Дубль запасного (%s)", teamName),
				}, true
			}
		}
	}

	return MatchAnomaly{}, false
}

// detectGoalkeeperWall выявляет матчи с 8+ сейвами вратаря
func detectGoalkeeperWall(m RoundResultMatch, info LeagueSeasonInfo) (MatchAnomaly, bool) {
	hSaves := extractStatInt(m.Stats, "saves", true)
	aSaves := extractStatInt(m.Stats, "saves", false)

	hScore := 0
	aScore := 0
	if m.Scores.HomeCurrent != nil {
		hScore = *m.Scores.HomeCurrent
	}
	if m.Scores.AwayCurrent != nil {
		aScore = *m.Scores.AwayCurrent
	}

	// Вратарь гостей отбил >= 8 ударов
	if aSaves >= 8 {
		severity := 70.0 + float64(aSaves)*2.0
		return MatchAnomaly{
			Type:           AnomalyTypeGoalkeeperWall,
			BadgeText:      "Стена тура",
			BadgeClass:     "bg-emerald-500/20 text-emerald-400 border-emerald-500/40",
			TagText:        "🐙 Вратарь-осьминог",
			SeverityScore:  severity,
			FixtureID:      m.FixtureID,
			LeagueID:       info.LeagueID,
			LeagueName:     info.LeagueName,
			RoundID:        m.RoundID,
			RoundName:      m.RoundName,
			TeamHomeName:   m.TeamHomeName,
			TeamHomeLogo:   m.TeamHomeLogo,
			TeamAwayName:   m.TeamAwayName,
			TeamAwayLogo:   m.TeamAwayLogo,
			ScoreHome:      hScore,
			ScoreAway:      aScore,
			HomeXG:         m.HomeXG,
			AwayXG:         m.AwayXG,
			HomePossession: extractStatInt(m.Stats, "possession", true),
			AwayPossession: extractStatInt(m.Stats, "possession", false),
			HomeShots:      extractStatInt(m.Stats, "shots_total", true),
			AwayShots:      extractStatInt(m.Stats, "shots_total", false),
			HomeSaves:      hSaves,
			AwaySaves:      aSaves,
			FooterIcon:     "🧤",
			FooterQuote:    fmt.Sprintf("«%d спасений голкипера спасли %s от разгрома»", aSaves, m.TeamAwayName),
			AnomalySummary: fmt.Sprintf("%d сейвов за 90 минут", aSaves),
		}, true
	}

	// Вратарь хозяев отбил >= 8 ударов
	if hSaves >= 8 {
		severity := 70.0 + float64(hSaves)*2.0
		return MatchAnomaly{
			Type:           AnomalyTypeGoalkeeperWall,
			BadgeText:      "Стена тура",
			BadgeClass:     "bg-emerald-500/20 text-emerald-400 border-emerald-500/40",
			TagText:        "🐙 Вратарь-осьминог",
			SeverityScore:  severity,
			FixtureID:      m.FixtureID,
			LeagueID:       info.LeagueID,
			LeagueName:     info.LeagueName,
			RoundID:        m.RoundID,
			RoundName:      m.RoundName,
			TeamHomeName:   m.TeamHomeName,
			TeamHomeLogo:   m.TeamHomeLogo,
			TeamAwayName:   m.TeamAwayName,
			TeamAwayLogo:   m.TeamAwayLogo,
			ScoreHome:      hScore,
			ScoreAway:      aScore,
			HomeXG:         m.HomeXG,
			AwayXG:         m.AwayXG,
			HomePossession: extractStatInt(m.Stats, "possession", true),
			AwayPossession: extractStatInt(m.Stats, "possession", false),
			HomeShots:      extractStatInt(m.Stats, "shots_total", true),
			AwayShots:      extractStatInt(m.Stats, "shots_total", false),
			HomeSaves:      hSaves,
			AwaySaves:      aSaves,
			FooterIcon:     "🧤",
			FooterQuote:    fmt.Sprintf("«%d сейвов голкипера принесли результат %s»", hSaves, m.TeamHomeName),
			AnomalySummary: fmt.Sprintf("%d сейвов за 90 минут", hSaves),
		}, true
	}

	return MatchAnomaly{}, false
}

func normalizeStatKey(k string) string {
	s := strings.ToLower(k)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

func extractStatInt(stats map[string]MatchStatMetric, key string, home bool) int {
	if stats == nil {
		return 0
	}
	target := normalizeStatKey(key)

	for k, item := range stats {
		normK := normalizeStatKey(k)
		normCode := normalizeStatKey(item.Code)
		normDev := normalizeStatKey(item.DeveloperName)

		isMatch := normK == target || normCode == target || normDev == target
		if !isMatch {
			switch target {
			case "possession":
				isMatch = strings.Contains(normK, "possession") || strings.Contains(normCode, "possession") || strings.Contains(normDev, "possession")
			case "shotstotal":
				isMatch = strings.Contains(normK, "shotstotal") || strings.Contains(normCode, "shotstotal") || strings.Contains(normDev, "shotstotal")
			case "saves":
				isMatch = strings.Contains(normK, "saves") || strings.Contains(normCode, "saves") || strings.Contains(normDev, "saves")
			}
		}

		if isMatch {
			val := item.Away
			if home {
				val = item.Home
			}
			if val != nil {
				return int(*val)
			}
		}
	}
	return 0
}
