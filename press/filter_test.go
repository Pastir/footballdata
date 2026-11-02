package press_test

import (
	"testing"

	"github.com/Pastir/footballdata/press"
)

func TestCalculateHypeScore(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		lead      string
		minScore  int
		maxScore  int
		wantZero  bool
	}{
		{
			name:     "Transfer bomb for top club",
			title:    "Real Madrid agrees terms with Erling Haaland: Record bid accepted",
			lead:     "Manchester City striker set for blockbuster summer move as here we go confirmed.",
			minScore: 80,
			maxScore: 100,
		},
		{
			name:     "Manager sacked",
			title:    "Erik ten Hag sacked by Manchester United after emergency board meeting",
			lead:     "Club announces manager has been dismissed with immediate effect.",
			minScore: 70,
			maxScore: 100,
		},
		{
			name:     "Negative filter - Women football",
			title:    "Arsenal Women secure dramatic Champions League victory",
			lead:     "A late goal in the WSL derby seals the points.",
			wantZero: true,
		},
		{
			name:     "Negative filter - Youth academy",
			title:    "Chelsea U18 team cruises past Southampton in youth cup",
			lead:     "Academy stars shine under lights.",
			wantZero: true,
		},
		{
			name:     "Routine minor news",
			title:    "Brentford defender returns to light individual training",
			lead:     "The player has been sidelined since October.",
			minScore: 10,
			maxScore: 35,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := press.CalculateHypeScore(tt.title, tt.lead)
			if tt.wantZero && score != 0 {
				t.Errorf("expected 0 for noise article, got %d", score)
			}
			if !tt.wantZero {
				if score < tt.minScore || score > tt.maxScore {
					t.Errorf("expected score between %d and %d, got %d", tt.minScore, tt.maxScore, score)
				}
			}
		})
	}
}

func TestCleanText(t *testing.T) {
	raw := `<p>Breaking news: <b>Real Madrid</b>&nbsp;signs superstar!&#039;s deal &quot;confirmed&quot;&amp;verified.</p>`
	expected := `Breaking news: Real Madrid signs superstar!'s deal "confirmed"&verified.`

	got := press.CleanText(raw)
	if got != expected {
		t.Errorf("CleanText failed.\nGot:      %q\nExpected: %q", got, expected)
	}
}

