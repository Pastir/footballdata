package analytics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
)

/*
	 Базовая привлекательность матча.

	 Strength имеет больший вес, потому что матч двух сильных команд
	 обычно интереснее независимо от краткосрочной формы.

	 Диапазон BaseScore: 0..100.

	 base := baseStrengthWeight*strength + baseFormWeight*form

	 Турнирная значимость усиливается по мере приближения
	 чемпионата к завершению.

	 Например:
		5/38  → progress ≈ 0.13 → seasonWeight ≈ 0.02
		19/38 → progress = 0.50 → seasonWeight = 0.25
		36/38 → progress ≈ 0.95 → seasonWeight ≈ 0.90

	 Квадрат progress специально подавляет влияние таблицы
	 в начале сезона и резко увеличивает его ближе к концу.
	 progress := matchSeasonProgress(match,ctx.TotalTeamMatches)

	 seasonWeight := progress * progress

	 Максимально Importance может добавить 30 баллов.

	 Пример в начале сезона:

		importance = 100
		seasonWeight = 0.02

		bonus = 100/100 × 0.02 × 30 = 0.6

	 Практически не влияет.

	 Пример в конце сезона:

		importance = 100
		seasonWeight = 0.90

		bonus = 100/100 × 0.90 × 30 = 27

	 Решающий матч получает очень большой приоритет.

importanceBonus :=

	(importance / 100.0) *
	seasonWeight *
	maxImportanceBonus

final := base + importanceBonus
*/
const (
	// baseStrengthWeight определяет вес силы команд при расчёте
	// базовой привлекательности матча.
	//
	// StrengthScore (0..100) рассчитывается по текущим позициям
	// обеих команд в турнирной таблице.
	//
	// Чем выше команды находятся в таблице, тем выше StrengthScore.
	// Например, матч #1 vs #2 получит значительно более высокий
	// рейтинг, чем матч #12 vs #15.
	//
	// Вес 0.65 означает, что положение команд в таблице составляет
	// 65% BaseScore и является главным фактором выбора матча тура.
	baseStrengthWeight = 0.65

	// baseFormWeight определяет вес текущей формы команд.
	//
	// FormScore (0..100) рассчитывается по последним матчам:
	//
	//	W = 3 очка
	//	D = 1 очко
	//	L = 0 очков
	//
	// Учитывается форма обеих команд.
	//
	// Вес 0.35 означает, что текущая форма составляет 35%
	// базового рейтинга матча.
	//
	// Таким образом матч сильных команд остаётся приоритетным,
	// но хорошая форма обеих команд повышает его привлекательность.
	baseFormWeight = 0.35

	// maxImportanceBonus — максимальный дополнительный бонус
	// за турнирную значимость матча.
	//
	// Бонус используется для матчей, которые могут существенно
	// повлиять на:
	//
	//	- борьбу за чемпионство;
	//	- места в еврокубках;
	//	- борьбу за выживание.
	//
	// Значимость такого матча дополнительно зависит от стадии сезона.
	// В начале чемпионата бонус практически отсутствует, поскольку
	// текущие позиции ещё мало что значат.
	//
	// Ближе к концу сезона влияние увеличивается. Например,
	// матч #1 vs #2 в 37-м туре или #17 vs #18 в борьбе
	// за сохранение места в лиге может получить большой бонус.
	//
	// 30.0 означает, что Importance может добавить максимум
	// 30 баллов к BaseScore.
	maxImportanceBonus = 30.0
)

// MatchCandidate — матч, который участвует в выборе матча тура.
type MatchCandidate struct {
	FixtureID int64

	TeamHomeID int64
	TeamAwayID int64

	TeamHomeName string
	TeamAwayName string

	PositionHome int
	PositionAway int

	PointsHome int
	PointsAway int

	PlayedHome int
	PlayedAway int

	FormHome []string // ["W","D","L","W","W"] — от старого к новому
	FormAway []string
}

// RoundContext содержит данные чемпионата,
// необходимые для оценки значимости матчей.
type RoundContext struct {
	TeamCount        int
	TotalTeamMatches int
	Standings        []Standing
	// standingIdx — индекс для O(1) поиска по TeamID; строится в RankMatchesOfRound.
	standingIdx map[int64]Standing
}

// MatchOfRoundResult — результат оценки одного матча.
type MatchOfRoundResult struct {
	FixtureID int64

	TeamHome string
	TeamAway string

	StrengthScore   float64
	FormScore       float64
	BaseScore       float64
	ImportanceScore float64
	ImportanceBonus float64
	FinalScore      float64

	Reasons []string
}

// RankMatchesOfRound оценивает все матчи тура
// и возвращает их от самого интересного к наименее интересному.
func RankMatchesOfRound(
	matches []MatchCandidate,
	ctx RoundContext,
) ([]MatchOfRoundResult, error) {
	if len(matches) == 0 {
		return nil, errors.New("no matches")
	}

	if ctx.TeamCount < 2 {
		return nil, errors.New("invalid team count")
	}

	if ctx.TotalTeamMatches <= 0 {
		return nil, errors.New("invalid total team matches")
	}

	// Строим индекс один раз для всего вызова — O(n) вместо O(n²)
	ctx.standingIdx = buildStandingIndex(ctx.Standings)

	results := make([]MatchOfRoundResult, 0, len(matches))

	for _, match := range matches {
		strength := strengthScore(
			match.PositionHome,
			match.PositionAway,
			ctx.TeamCount,
		)

		form := formScore(match.FormHome, match.FormAway)

		base :=
			baseStrengthWeight*strength +
				baseFormWeight*form

		importance, reasons := importanceScore(match, ctx)

		progress := matchSeasonProgress(match, ctx.TotalTeamMatches)

		seasonWeight := progress * progress

		importanceBonus :=
			(importance / 100.0) *
				seasonWeight *
				maxImportanceBonus

		final := base + importanceBonus

		results = append(results, MatchOfRoundResult{
			FixtureID: match.FixtureID,

			TeamHome: match.TeamHomeName,
			TeamAway: match.TeamAwayName,

			StrengthScore:   strength,
			FormScore:       form,
			BaseScore:       base,
			ImportanceScore: importance,
			ImportanceBonus: importanceBonus,
			FinalScore:      final,

			Reasons: reasons,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	return results, nil
}

// SelectMatchOfRound возвращает лучший матч тура.
func SelectMatchOfRound(
	matches []MatchCandidate,
	ctx RoundContext,
) (MatchOfRoundResult, error) {
	results, err := RankMatchesOfRound(matches, ctx)
	if err != nil {
		return MatchOfRoundResult{}, err
	}

	if len(results) == 0 {
		return MatchOfRoundResult{}, errors.New("no ranked matches")
	}

	return results[0], nil
}

// MatchOfRound — оркестратор: загружает standings из БД и запускает ранжирование.
// matches передаются снаружи (из GetNextRoundFixtures → BuildMatchCandidates).
func MatchOfRound(
	ctx context.Context,
	pool *pgxpool.Pool,
	info *LeagueSeasonInfo,
	matches []MatchCandidate,
) ([]MatchOfRoundResult, error) {
	standings, err := GetCurrentStandings(ctx, pool, info)
	if err != nil {
		return nil, fmt.Errorf("MatchOfRound: GetCurrentStandings: %w", err)
	}

	rctx := RoundContext{
		TeamCount:        info.TeamCount,
		TotalTeamMatches: info.TotalTeamMatches,
		Standings:        standings,
	}

	return RankMatchesOfRound(matches, rctx)
}

// BuildMatchCandidates преобразует []NextRoundMatch → []MatchCandidate
// для передачи в RankMatchesOfRound / MatchOfRound.
func BuildMatchCandidates(matches []NextRoundMatch) []MatchCandidate {
	candidates := make([]MatchCandidate, 0, len(matches))
	for _, m := range matches {
		posHome := 0
		if m.PositionHome != nil {
			posHome = *m.PositionHome
		}
		posAway := 0
		if m.PositionAway != nil {
			posAway = *m.PositionAway
		}
		ptsHome := 0
		if m.PointsHome != nil {
			ptsHome = *m.PointsHome
		}
		ptsAway := 0
		if m.PointsAway != nil {
			ptsAway = *m.PointsAway
		}
		playedHome := 0
		if m.OverallMatchesHome != nil {
			playedHome = *m.OverallMatchesHome
		}
		playedAway := 0
		if m.MatchesAway != nil {
			playedAway = *m.MatchesAway
		}

		candidates = append(candidates, MatchCandidate{
			FixtureID:    m.FixtureID,
			TeamHomeID:   m.TeamHomeID,
			TeamAwayID:   m.TeamAwayID,
			TeamHomeName: m.TeamHomeName,
			TeamAwayName: m.TeamAwayName,
			PositionHome: posHome,
			PositionAway: posAway,
			PointsHome:   ptsHome,
			PointsAway:   ptsAway,
			PlayedHome:   playedHome,
			PlayedAway:   playedAway,
			FormHome:     m.FormHome,
			FormAway:     m.FormAway,
		})
	}
	return candidates
}

// ---------------------------------------------------------
// STRENGTH
// ---------------------------------------------------------

// strengthScore оценивает силу пары по текущим позициям.
//
// #1 vs #2 получает 100.
// Чем ниже обе команды, тем меньше score.
func strengthScore(
	homePosition int,
	awayPosition int,
	teamCount int,
) float64 {
	if homePosition < 1 ||
		awayPosition < 1 ||
		homePosition > teamCount ||
		awayPosition > teamCount {
		return 0
	}

	homeStrength := teamCount + 1 - homePosition
	awayStrength := teamCount + 1 - awayPosition

	raw := homeStrength + awayStrength

	// Теоретический максимум реального матча:
	// #1 против #2.
	maxRaw := teamCount + (teamCount - 1)

	score := float64(raw) / float64(maxRaw) * 100

	return clamp(score, 0, 100)
}

// ---------------------------------------------------------
// FORM
// ---------------------------------------------------------

func formScore(homeForm, awayForm []string) float64 {
	homePoints, homeMatches := formPoints(homeForm)
	awayPoints, awayMatches := formPoints(awayForm)

	totalMatches := homeMatches + awayMatches

	if totalMatches == 0 {
		return 0
	}

	maxPoints := totalMatches * 3

	return float64(homePoints+awayPoints) /
		float64(maxPoints) *
		100
}

func formPoints(form []string) (points int, matches int) {
	for _, result := range form {
		switch strings.ToUpper(result) {
		case "W":
			points += 3
			matches++
		case "D":
			points++
			matches++
		case "L":
			matches++
		}
	}
	return points, matches
}

// ---------------------------------------------------------
// SEASON PROGRESS
// ---------------------------------------------------------

func matchSeasonProgress(
	match MatchCandidate,
	totalMatches int,
) float64 {
	if totalMatches <= 0 {
		return 0
	}

	// Команды могут иметь разное число сыгранных матчей
	// из-за переносов.
	played := float64(
		match.PlayedHome+match.PlayedAway,
	) / 2.0

	progress := played / float64(totalMatches)

	return clamp(progress, 0, 1)
}

// ---------------------------------------------------------
// IMPORTANCE
// ---------------------------------------------------------

// importanceScore возвращает значение 0..100.
//
// Здесь мы оцениваем:
//  1. борьбу за чемпионство;
//  2. борьбу около границ еврокубковых зон;
//  3. борьбу около границы вылета.
//
// Season progress здесь НЕ применяется.
// Он применяется отдельно при расчёте ImportanceBonus.
func importanceScore(
	match MatchCandidate,
	ctx RoundContext,
) (float64, []string) {
	var score float64
	var reasons []string

	home, homeOK := ctx.standingIdx[match.TeamHomeID]
	away, awayOK := ctx.standingIdx[match.TeamAwayID]

	if !homeOK || !awayOK {
		return 0, nil
	}

	// -------------------------------------------------
	// TITLE RACE
	// -------------------------------------------------

	titleScore := titleRaceScore(home, away, ctx)

	if titleScore > score {
		score = titleScore
	}

	if titleScore >= 50 {
		reasons = append(reasons, "title race")
	}

	// -------------------------------------------------
	// STANDINGS ZONES
	// -------------------------------------------------

	zoneScore, zoneReason := zoneRaceScore(home, away, ctx)

	if zoneScore > score {
		score = zoneScore
	}

	if zoneReason != "" {
		reasons = append(reasons, zoneReason)
	}

	return clamp(score, 0, 100), reasons
}

// ---------------------------------------------------------
// TITLE RACE
// ---------------------------------------------------------

func titleRaceScore(
	home Standing,
	away Standing,
	ctx RoundContext,
) float64 {
	leader, ok := standingAtPosition(ctx.Standings, 1)

	if !ok {
		return 0
	}

	homeDistance := leader.Points - home.Points
	awayDistance := leader.Points - away.Points

	homeRemaining :=
		(ctx.TotalTeamMatches - home.Played) * 3

	awayRemaining :=
		(ctx.TotalTeamMatches - away.Played) * 3

	// Команда математически уже не может бороться
	// за первое место.
	if homeDistance > homeRemaining &&
		awayDistance > awayRemaining {
		return 0
	}

	homeAlive := homeDistance <= homeRemaining
	awayAlive := awayDistance <= awayRemaining

	if !homeAlive || !awayAlive {
		return 0
	}

	pointsGap := absInt(home.Points - away.Points)

	remaining := minInt(homeRemaining, awayRemaining)

	if remaining <= 0 {
		return 0
	}

	closeness :=
		1.0 -
			float64(pointsGap)/
				float64(remaining+1)

	closeness = clamp(closeness, 0, 1)

	// Дополнительно учитываем положение команд:
	// матч #1 vs #2 важнее #1 vs #10.
	positionFactor :=
		1.0 -
			float64(
				(home.Position-1)+(away.Position-1),
			)/
				float64(ctx.TeamCount*2)

	positionFactor = clamp(positionFactor, 0, 1)

	return 100 * closeness * positionFactor
}

// ---------------------------------------------------------
// ZONE RACE
// ---------------------------------------------------------

func zoneRaceScore(
	home Standing,
	away Standing,
	ctx RoundContext,
) (float64, string) {
	// Standings уже отсортированы по position из БД — сортировка не нужна.
	boundaries := findRuleBoundaries(ctx.Standings)

	var bestScore float64
	var bestRule string

	for _, boundary := range boundaries {
		score := boundaryImpact(
			home,
			away,
			boundary,
			ctx.TotalTeamMatches,
		)

		if score > bestScore {
			bestScore = score
			bestRule = boundary.Rule
		}
	}

	if bestScore < 50 {
		return bestScore, ""
	}

	switch {
	case strings.Contains(strings.ToLower(bestRule), "relegation"):
		return bestScore, "relegation battle"
	default:
		return bestScore, bestRule + " race"
	}
}

// RuleBoundary — граница между двумя разными правилами таблицы.
//
// Например:
//
//	#4 Champions League
//	#5 Europa League
//
// или:
//
//	#17 ""
//	#18 Relegation
type RuleBoundary struct {
	UpperPosition int
	LowerPosition int

	UpperPoints int
	LowerPoints int

	Rule string // человекочитаемое название (из Standing.Rule)
}

// findRuleBoundaries находит границы между зонами в таблице.
// Предусловие: standings уже отсортированы по position (гарантирует GetCurrentStandings).
func findRuleBoundaries(standings []Standing) []RuleBoundary {
	if len(standings) < 2 {
		return nil
	}

	var boundaries []RuleBoundary

	for i := 0; i < len(standings)-1; i++ {
		current := standings[i]
		next := standings[i+1]

		// Используем RuleDevName для сравнения зон (он заполнен, Rule — для отображения)
		currentZone := normalizeRule(current.RuleDevName)
		nextZone := normalizeRule(next.RuleDevName)

		if currentZone == nextZone {
			continue
		}

		// Если обе стороны пустые — границы нет
		if currentZone == "" && nextZone == "" {
			continue
		}

		// Берём Rule (человекочитаемое) той стороны, которая имеет зону
		rule := next.Rule
		if rule == "" {
			rule = current.Rule
		}

		boundaries = append(boundaries, RuleBoundary{
			UpperPosition: current.Position,
			LowerPosition: next.Position,

			UpperPoints: current.Points,
			LowerPoints: next.Points,

			Rule: rule,
		})
	}

	return boundaries
}

func boundaryImpact(
	home Standing,
	away Standing,
	boundary RuleBoundary,
	totalMatches int,
) float64 {
	homeDistance := distanceToBoundary(home.Position, boundary)
	awayDistance := distanceToBoundary(away.Position, boundary)

	// Если обе команды далеко от границы,
	// матч к этой борьбе не относится.
	if homeDistance > 3 || awayDistance > 3 {
		return 0
	}

	positionDistance := homeDistance + awayDistance

	positionFactor :=
		1.0 -
			float64(positionDistance)/6.0

	positionFactor = clamp(positionFactor, 0, 1)

	pointsGap := absInt(home.Points - away.Points)

	remainingHome :=
		(totalMatches - home.Played) * 3

	remainingAway :=
		(totalMatches - away.Played) * 3

	remaining := minInt(remainingHome, remainingAway)

	if remaining < 0 {
		remaining = 0
	}

	pointsFactor :=
		1.0 -
			float64(pointsGap)/
				float64(remaining+3)

	pointsFactor = clamp(pointsFactor, 0, 1)

	return 100 * positionFactor * pointsFactor
}

func distanceToBoundary(
	position int,
	boundary RuleBoundary,
) int {
	if position <= boundary.UpperPosition {
		return boundary.UpperPosition - position
	}

	if position >= boundary.LowerPosition {
		return position - boundary.LowerPosition
	}

	return 0
}

// ---------------------------------------------------------
// HELPERS
// ---------------------------------------------------------

// buildStandingIndex строит map teamID → Standing для O(1) поиска.
func buildStandingIndex(standings []Standing) map[int64]Standing {
	idx := make(map[int64]Standing, len(standings))
	for _, s := range standings {
		idx[s.TeamID] = s
	}
	return idx
}

func standingAtPosition(
	standings []Standing,
	position int,
) (Standing, bool) {
	for _, standing := range standings {
		if standing.Position == position {
			return standing, true
		}
	}

	return Standing{}, false
}

func normalizeRule(rule string) string {
	return strings.ToLower(strings.TrimSpace(rule))
}

func clamp(
	value,
	minValue,
	maxValue float64,
) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
