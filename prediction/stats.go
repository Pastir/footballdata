package prediction

import (
	"math"
)

// CalculatePerformance вычисляет агрегированную финансовую статистику по списку завершённых прогнозов.
func CalculatePerformance(predictions []Prediction) PerformanceReport {
	var (
		totalBets   int
		wonBets     int
		lostBets    int
		refundBets  int
		totalStaked float64
		totalProfit float64
		grossProfit float64
		grossLoss   float64
		wonOddsSum  float64
	)

	for _, p := range predictions {
		// Учитываем только завершённые ставки
		switch p.Status {
		case StatusWon:
			totalBets++
			wonBets++
			totalStaked += p.RecommendedStake
			totalProfit += p.ProfitUnits
			grossProfit += p.ProfitUnits
			wonOddsSum += p.TotalOdds
		case StatusLost:
			totalBets++
			lostBets++
			totalStaked += p.RecommendedStake
			totalProfit += p.ProfitUnits
			grossLoss += math.Abs(p.ProfitUnits)
		case StatusRefund:
			totalBets++
			refundBets++
			totalStaked += p.RecommendedStake
		default:
			// pending или cancelled не участвуют в статистике проходимости
			continue
		}
	}

	report := PerformanceReport{
		TotalBets:   totalBets,
		WonBets:     wonBets,
		LostBets:    lostBets,
		RefundBets:  refundBets,
		TotalStaked: roundTwo(totalStaked),
		TotalProfit: roundTwo(totalProfit),
	}

	// WinRate = Won / (Won + Lost) * 100 (возвраты исключаются из базы расчёта)
	settledDecisive := wonBets + lostBets
	if settledDecisive > 0 {
		report.WinRate = roundTwo((float64(wonBets) / float64(settledDecisive)) * 100.0)
	}

	// ROI / Yield = TotalProfit / TotalStaked * 100
	if totalStaked > 0 {
		report.ROI = roundTwo((totalProfit / totalStaked) * 100.0)
	}

	// Средний выигрышный коэффициент
	if wonBets > 0 {
		report.AvgOdds = roundTwo(wonOddsSum / float64(wonBets))
	}

	// Profit Factor = Gross Profit / Gross Loss
	if grossLoss > 0 {
		report.ProfitFactor = roundTwo(grossProfit / grossLoss)
	} else if grossProfit > 0 {
		report.ProfitFactor = 99.99 // условный максимум при нуле проигрышей
	}

	return report
}

func roundTwo(val float64) float64 {
	return math.Round(val*100) / 100
}
