package analytics

import (
	"testing"
	"time"
)

func TestEvaluateRoundClosing_CleanFinish(t *testing.T) {
	kickoff := time.Date(2026, 10, 19, 19, 0, 0, 0, time.UTC)
	endingAt := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)

	// Ситуация 1: 10 из 10 матчей сыграно, прошло 2 часа 30 минут с момента стартового свистка
	status := &RoundClosingStatus{
		LeagueID:         8,
		RoundID:          407868,
		EndingAt:         endingAt,
		TotalMatches:     10,
		FinishedMatches:  10,
		PostponedMatches: 0,
		InPlayMatches:    0,
		UpcomingMatches:  0,
		LastMatchKickoff: &kickoff,
	}

	nowAfterBuffer := kickoff.Add(2*time.Hour + 30*time.Minute)
	EvaluateRoundClosing(status, nowAfterBuffer)

	if !status.IsReadyForClosing {
		t.Errorf("expected round to be ready for closing, got false: reason=%s", status.Reason)
	}
	if status.HasPostponements {
		t.Errorf("expected no postponements, got true")
	}

	// Ситуация 2: 10 из 10 матчей сыграно, но прошло всего 45 минут от свистка (буфер ещё не истёк)
	statusBufferWait := &RoundClosingStatus{
		LeagueID:         8,
		RoundID:          407868,
		EndingAt:         endingAt,
		TotalMatches:     10,
		FinishedMatches:  10,
		PostponedMatches: 0,
		InPlayMatches:    0,
		UpcomingMatches:  0,
		LastMatchKickoff: &kickoff,
	}
	nowDuringMatch := kickoff.Add(45 * time.Minute)
	EvaluateRoundClosing(statusBufferWait, nowDuringMatch)

	if statusBufferWait.IsReadyForClosing {
		t.Errorf("expected round NOT to be ready due to match buffer, got true")
	}
}

func TestEvaluateRoundClosing_InPlayMatches(t *testing.T) {
	endingAt := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	status := &RoundClosingStatus{
		LeagueID:        8,
		RoundID:         407868,
		EndingAt:        endingAt,
		TotalMatches:    10,
		FinishedMatches: 9,
		InPlayMatches:   1, // 1 матч прямо сейчас идёт
		UpcomingMatches: 0,
	}

	EvaluateRoundClosing(status, time.Now())
	if status.IsReadyForClosing {
		t.Errorf("expected round NOT to be ready when matches are in play, got true")
	}
}

func TestEvaluateRoundClosing_FallbackPostponedMatches(t *testing.T) {
	endingAt := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC) // понедельник 19 октября
	kickoff := time.Date(2026, 10, 18, 15, 30, 0, 0, time.UTC)

	// Ситуация: 9 матчей сыграно, 1 матч отложен (например, из-за кубка или погоды)
	// До наступления дедлайна вторника 08:30 UTC: тур не закрывается, ждёт
	statusBeforeDeadline := &RoundClosingStatus{
		LeagueID:         8,
		RoundID:          407868,
		EndingAt:         endingAt,
		TotalMatches:     10,
		FinishedMatches:  9,
		PostponedMatches: 1,
		InPlayMatches:    0,
		UpcomingMatches:  0,
		LastMatchKickoff: &kickoff,
	}

	// Понедельник 22:00 UTC (все сыгранные матчи завершились, перенесённый учтён)
	nowMondayNight := time.Date(2026, 10, 19, 22, 0, 0, 0, time.UTC)
	EvaluateRoundClosing(statusBeforeDeadline, nowMondayNight)

	if !statusBeforeDeadline.IsReadyForClosing {
		t.Errorf("expected round to be ready when all available matches finished and remaining are postponed")
	}
	if !statusBeforeDeadline.HasPostponements {
		t.Errorf("expected HasPostponements = true")
	}

	// Ситуация 2: 1 матч остался в статусе Not Started (забыт провайдером или перенесён без смены статуса)
	statusWithNS := &RoundClosingStatus{
		LeagueID:         8,
		RoundID:          407868,
		EndingAt:         endingAt,
		TotalMatches:     10,
		FinishedMatches:  9,
		PostponedMatches: 0,
		InPlayMatches:    0,
		UpcomingMatches:  1, // 1 матч так и остался висеть как NS
		LastMatchKickoff: &kickoff,
	}

	// Понедельник вечер: тур не закрывается, ждёт дедлайна
	EvaluateRoundClosing(statusWithNS, nowMondayNight)
	if statusWithNS.IsReadyForClosing {
		t.Errorf("expected round NOT to be ready on Monday night if upcoming matches exist")
	}

	// Вторник утро 09:00 UTC (дедлайн 08:30 UTC прошёл): тур принудительно закрывается
	nowTuesdayMorning := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	EvaluateRoundClosing(statusWithNS, nowTuesdayMorning)
	if !statusWithNS.IsReadyForClosing {
		t.Errorf("expected round to be ready on Tuesday morning fallback deadline, got false: reason=%s", statusWithNS.Reason)
	}
	if !statusWithNS.HasPostponements {
		t.Errorf("expected HasPostponements = true after Tuesday fallback deadline")
	}
}

func TestEvaluateRoundClosing_ZeroMatches(t *testing.T) {
	status := &RoundClosingStatus{
		TotalMatches: 0,
	}
	EvaluateRoundClosing(status, time.Now())
	if status.IsReadyForClosing {
		t.Errorf("expected false for zero matches, got true")
	}

	statusNoFinished := &RoundClosingStatus{
		TotalMatches:    10,
		FinishedMatches: 0,
		UpcomingMatches: 10,
	}
	EvaluateRoundClosing(statusNoFinished, time.Now())
	if statusNoFinished.IsReadyForClosing {
		t.Errorf("expected false for 0 finished matches, got true")
	}
}
