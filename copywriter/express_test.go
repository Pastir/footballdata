package copywriter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/copywriter"
	"github.com/Pastir/footballdata/prediction"
)

func createTestPlan() *analytics.SmartExpressPlan {
	return &analytics.SmartExpressPlan{
		Title:            "УМНЫЙ ЭКСПРЕСС",
		TotalOdds:        4.15,
		Confidence:       8.5,
		ValueEdge:        7.4,
		RecommendedStake: 1.0,
		Selections: []analytics.ExpressSelectionOption{
			{
				Match: analytics.ExpressCandidateMatch{
					HomeTeamName: "Арсенал",
					AwayTeamName: "Эвертон",
					StartingAt:   time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC),
				},
				MarketKey:  prediction.MarketKey1X2,
				MarketName: "Победа в матче (1X2)",
				Selection:  "1",
				Odds:       1.45,
				Reason:     "Арсенал мощно выступает на своём поле",
			},
			{
				Match: analytics.ExpressCandidateMatch{
					HomeTeamName: "Астон Вилла",
					AwayTeamName: "Манчестер Сити",
					StartingAt:   time.Date(2026, 10, 24, 11, 30, 0, 0, time.UTC),
				},
				MarketKey:  prediction.MarketKey1X2,
				MarketName: "Победа в матче (1X2)",
				Selection:  "2",
				Odds:       1.70,
				Reason:     "Манчестер Сити превосходит соперника в классе",
			},
			{
				Match: analytics.ExpressCandidateMatch{
					HomeTeamName: "Челси",
					AwayTeamName: "Тоттенхэм",
					StartingAt:   time.Date(2026, 10, 24, 16, 30, 0, 0, time.UTC),
				},
				MarketKey:  prediction.MarketKeyDoubleChance,
				MarketName: "Двойной шанс",
				Selection:  "1X",
				Odds:       1.48,
				Reason:     "Челси крайне неуступчив дома",
			},
		},
	}
}

func TestGenerateExpressPostStyle1(t *testing.T) {
	plan := createTestPlan()
	cfg := copywriter.ExpressPostConfig{
		LeagueName:    "Премьер-лига",
		RoundName:     "8",
		BookmakerName: "FONBET",
		PromoCode:     "FDL",
		BonusText:     "Фрибет до 15 000 ₽",
	}

	html := copywriter.GenerateExpressPostStyle1(cfg, plan)

	if !strings.Contains(html, "[1 ТЕСТ]") {
		t.Errorf("missing [1 ТЕСТ] label")
	}
	if !strings.Contains(html, "4.15") {
		t.Errorf("missing total odds in express post")
	}
	if !strings.Contains(html, "<blockquote>") {
		t.Errorf("missing blockquote in style 1")
	}
	if !strings.Contains(html, "Арсенал — Эвертон") {
		t.Errorf("missing match Arsenal in express post")
	}
}

func TestGenerateExpressPostStyle2(t *testing.T) {
	plan := createTestPlan()
	cfg := copywriter.ExpressPostConfig{
		LeagueName:    "Премьер-лига",
		RoundName:     "8",
		BookmakerName: "FONBET",
		PromoCode:     "FDL",
	}

	html := copywriter.GenerateExpressPostStyle2(cfg, plan)

	if !strings.Contains(html, "[2 ТЕСТ]") {
		t.Errorf("missing [2 ТЕСТ] label")
	}
	if !strings.Contains(html, "<pre>") {
		t.Errorf("missing <pre> block in style 2")
	}
	if !strings.Contains(html, "4.15") {
		t.Errorf("missing total odds in style 2")
	}
}

func TestGenerateExpressPostStyle3Caption(t *testing.T) {
	plan := createTestPlan()
	cfg := copywriter.ExpressPostConfig{
		LeagueName:    "Премьер-лига",
		RoundName:     "8",
		BookmakerName: "FONBET",
		PromoCode:     "FDL",
	}

	caption := copywriter.GenerateExpressPostStyle3Caption(cfg, plan)

	if !strings.Contains(caption, "[3 ТЕСТ]") {
		t.Errorf("missing [3 ТЕСТ] label")
	}
	if !strings.Contains(caption, "4.15") {
		t.Errorf("missing total odds in style 3 caption")
	}
}
