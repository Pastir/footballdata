package prediction

import (
	"context"
	"fmt"
)

// FixtureScoreProvider интерфейс для получения финального счёта завершённого матча.
type FixtureScoreProvider interface {
	// GetFinishedScore возвращает ftScore, htScore и boolean finished (true если матч завершен).
	GetFinishedScore(ctx context.Context, fixtureID int) (ftScore string, htScore string, finished bool, err error)
}

// SettlementWorker производит периодический опрос и автоматический расчет исходов ставок.
type SettlementWorker struct {
	storage       Storage
	scoreProvider FixtureScoreProvider
}

// NewSettlementWorker создаёт новый воркер расчёта ставок.
func NewSettlementWorker(storage Storage, scoreProvider FixtureScoreProvider) *SettlementWorker {
	return &SettlementWorker{
		storage:       storage,
		scoreProvider: scoreProvider,
	}
}

// ProcessPending проверяет все незавершённые прогнозы и рассчитывает завершённые события.
// Возвращает количество успешно рассчитанных прогнозов.
func (w *SettlementWorker) ProcessPending(ctx context.Context) (int, error) {
	pendingList, err := w.storage.ListPendingPredictions(ctx)
	if err != nil {
		return 0, fmt.Errorf("list pending: %w", err)
	}

	settledCount := 0

	for _, pred := range pendingList {
		predUpdated := false

		for i := range pred.Items {
			item := &pred.Items[i]
			if item.SettlementStatus != StatusPending {
				continue
			}

			ftScore, htScore, finished, err := w.scoreProvider.GetFinishedScore(ctx, item.FixtureID)
			if err != nil || !finished {
				continue
			}

			status, details, err := EvaluateItemOutcome(item.MarketKey, item.Selection, ftScore)
			if err != nil {
				continue
			}

			if err = w.storage.SettleItem(ctx, item.ID, status, ftScore, htScore, details); err != nil {
				return settledCount, fmt.Errorf("settle item %d: %w", item.ID, err)
			}

			item.SettlementStatus = status
			item.ActualScoreFT = ftScore
			item.ActualScoreHT = htScore
			item.SettlementDetails = details
			predUpdated = true
		}

		if predUpdated {
			finalStatus, profit := EvaluatePrediction(&pred)
			if finalStatus != StatusPending {
				if err = w.storage.SettlePrediction(ctx, pred.ID, finalStatus, profit); err != nil {
					return settledCount, fmt.Errorf("settle prediction %d: %w", pred.ID, err)
				}
				settledCount++
			}
		}
	}

	return settledCount, nil
}
