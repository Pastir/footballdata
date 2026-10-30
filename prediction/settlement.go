package prediction

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var reThreshold = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)`)

// ParseScore парсит строку счёта вида "2:1", "2-1" в количество голов хозяев и гостей.
func ParseScore(scoreStr string) (home int, away int, err error) {
	s := strings.TrimSpace(scoreStr)
	sep := ":"
	if strings.Contains(s, "-") {
		sep = "-"
	}

	parts := strings.Split(s, sep)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid score format %q: expected home%saway", scoreStr, sep)
	}

	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("parse home goals from %q: %w", parts[0], err)
	}

	a, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("parse away goals from %q: %w", parts[1], err)
	}

	return h, a, nil
}

// EvaluateItemOutcome рассчитывает исход одного выбора на основе фактического счёта матча.
func EvaluateItemOutcome(marketKey MarketKey, selection, scoreFT string) (BetStatus, string, error) {
	home, away, err := ParseScore(scoreFT)
	if err != nil {
		return StatusPending, "", fmt.Errorf("evaluate outcome: %w", err)
	}

	totalGoals := home + away
	normSel := strings.ToLower(strings.TrimSpace(selection))
	normSel = strings.ReplaceAll(normSel, ",", ".")

	switch marketKey {
	case MarketKey1X2:
		switch normSel {
		case "1", "home", "home win", "п1":
			if home > away {
				return StatusWon, fmt.Sprintf("Счёт %d:%d — победа хозяев", home, away), nil
			}
			return StatusLost, fmt.Sprintf("Счёт %d:%d — ставка на П1 не сыграла", home, away), nil
		case "x", "draw", "х", "ничья":
			if home == away {
				return StatusWon, fmt.Sprintf("Счёт %d:%d — зафиксирована ничья", home, away), nil
			}
			return StatusLost, fmt.Sprintf("Счёт %d:%d — ничья не состоялась", home, away), nil
		case "2", "away", "away win", "п2":
			if away > home {
				return StatusWon, fmt.Sprintf("Счёт %d:%d — победа гостей", home, away), nil
			}
			return StatusLost, fmt.Sprintf("Счёт %d:%d — ставка на П2 не сыграла", home, away), nil
		default:
			return StatusPending, "", fmt.Errorf("unknown 1X2 selection: %q", selection)
		}

	case MarketKeyDoubleChance:
		switch normSel {
		case "1x", "1х":
			if home >= away {
				return StatusWon, fmt.Sprintf("Счёт %d:%d — хозяева не проиграли", home, away), nil
			}
			return StatusLost, fmt.Sprintf("Счёт %d:%d — победа гостей", home, away), nil
		case "12":
			if home != away {
				return StatusWon, fmt.Sprintf("Счёт %d:%d — без ничьей", home, away), nil
			}
			return StatusLost, fmt.Sprintf("Счёт %d:%d — ничейный результат", home, away), nil
		case "x2", "х2":
			if away >= home {
				return StatusWon, fmt.Sprintf("Счёт %d:%d — гости не проиграли", home, away), nil
			}
			return StatusLost, fmt.Sprintf("Счёт %d:%d — победа хозяев", home, away), nil
		default:
			return StatusPending, "", fmt.Errorf("unknown Double Chance selection: %q", selection)
		}

	case MarketKeyOverUnder:
		// Поддержка "Over 2.5", "ТБ 2.5", "Under 2.5", "ТМ 2.5", "ТБ 2,5", "ТБ2.5"
		isOver := strings.HasPrefix(normSel, "over") || strings.HasPrefix(normSel, "тб")
		isUnder := strings.HasPrefix(normSel, "under") || strings.HasPrefix(normSel, "тм")

		if !isOver && !isUnder {
			return StatusPending, "", fmt.Errorf("unknown Over/Under selection: %q", selection)
		}

		// Извлекаем числовой порог
		match := reThreshold.FindString(normSel)
		if match == "" {
			return StatusPending, "", fmt.Errorf("cannot extract threshold from: %q", selection)
		}
		threshold, err := strconv.ParseFloat(match, 64)
		if err != nil {
			return StatusPending, "", fmt.Errorf("parse threshold %q: %w", match, err)
		}

		floatTotal := float64(totalGoals)
		if isOver {
			if floatTotal > threshold {
				return StatusWon, fmt.Sprintf("Тотал голов %d > %.1f — Выигрыш", totalGoals, threshold), nil
			} else if floatTotal == threshold {
				return StatusRefund, fmt.Sprintf("Тотал голов %d = %.1f — Возврат", totalGoals, threshold), nil
			}
			return StatusLost, fmt.Sprintf("Тотал голов %d <= %.1f — Проигрыш", totalGoals, threshold), nil
		}

		// isUnder
		if floatTotal < threshold {
			return StatusWon, fmt.Sprintf("Тотал голов %d < %.1f — Выигрыш", totalGoals, threshold), nil
		} else if floatTotal == threshold {
			return StatusRefund, fmt.Sprintf("Тотал голов %d = %.1f — Возврат", totalGoals, threshold), nil
		}
		return StatusLost, fmt.Sprintf("Тотал голов %d >= %.1f — Проигрыш", totalGoals, threshold), nil

	case MarketKeyBTTS:
		switch normSel {
		case "yes", "да", "обе забьют - да":
			if home > 0 && away > 0 {
				return StatusWon, fmt.Sprintf("Счёт %d:%d — обе команды забили", home, away), nil
			}
			return StatusLost, fmt.Sprintf("Счёт %d:%d — одна из команд не забила", home, away), nil
		case "no", "нет", "обе забьют - нет":
			if home == 0 || away == 0 {
				return StatusWon, fmt.Sprintf("Счёт %d:%d — хотя бы одна команда не забила", home, away), nil
			}
			return StatusLost, fmt.Sprintf("Счёт %d:%d — обе команды отличились", home, away), nil
		default:
			return StatusPending, "", fmt.Errorf("unknown BTTS selection: %q", selection)
		}

	default:
		return StatusPending, "", fmt.Errorf("unsupported market key: %q", marketKey)
	}
}

// EvaluatePrediction рассчитывает итоговый статус и прибыль/убыток для всей карточки прогноза.
func EvaluatePrediction(p *Prediction) (BetStatus, float64) {
	if len(p.Items) == 0 {
		return StatusPending, 0.0
	}

	stake := p.RecommendedStake
	if stake <= 0 {
		stake = 1.0
	}

	var (
		hasPending bool
		hasLost    bool
		allRefund  = true
		effectiveOdds = 1.0
	)

	for _, item := range p.Items {
		switch item.SettlementStatus {
		case StatusPending:
			hasPending = true
		case StatusLost:
			hasLost = true
			allRefund = false
		case StatusWon:
			allRefund = false
			effectiveOdds *= item.Odds
		case StatusRefund:
			// При возврате коэффициент исхода равен 1.0
			effectiveOdds *= 1.0
		}
	}

	// Если хотя бы одно событие ещё не рассчитано и нет явного проигрыша экспресса
	if hasPending {
		// Для экспресса: если хоть одно событие уже проиграло, весь экспресс проигран
		if p.BetType == BetTypeAccumulator && hasLost {
			return StatusLost, -roundUnits(stake)
		}
		return StatusPending, 0.0
	}

	// Если есть проигравшее событие
	if hasLost {
		return StatusLost, -roundUnits(stake)
	}

	// Если все события с возвратом
	if allRefund {
		return StatusRefund, 0.0
	}

	// Все события выиграли (или выиграли с возвратом части)
	profit := stake * (effectiveOdds - 1.0)
	return StatusWon, roundUnits(profit)
}

func roundUnits(val float64) float64 {
	return math.Round(val*1000) / 1000
}
