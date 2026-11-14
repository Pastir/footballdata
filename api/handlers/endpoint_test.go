package handler

import "testing"

func TestReplaceToken(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		tokens map[string]interface{}
		want   string
	}{
		{
			name: "date range tokens",
			url:  "https://api.sportmonks.com/v3/football/fixtures/between/{start_date}/{end_date}",
			tokens: map[string]interface{}{
				"start_date": "2026-11-01",
				"end_date":   "2026-11-10",
			},
			want: "https://api.sportmonks.com/v3/football/fixtures/between/2026-11-01/2026-11-10",
		},
		{
			name: "single date token without start_date/end_date",
			url:  "https://api.sportmonks.com/v3/football/fixtures/date/{date}",
			tokens: map[string]interface{}{
				"date": "2026-11-13",
			},
			want: "https://api.sportmonks.com/v3/football/fixtures/date/2026-11-13",
		},
		{
			name: "season and team int tokens",
			url:  "https://api.sportmonks.com/v3/football/squads/seasons/{seasonID}/teams/{teamID}",
			tokens: map[string]interface{}{
				"season": 23614,
				"team":   18,
			},
			want: "https://api.sportmonks.com/v3/football/squads/seasons/23614/teams/18",
		},
		{
			name: "rounds int64 token",
			url:  "https://api.sportmonks.com/v3/football/standings/rounds/{roundID}",
			tokens: map[string]interface{}{
				"rounds": int64(372195),
			},
			want: "https://api.sportmonks.com/v3/football/standings/rounds/372195",
		},
		{
			name: "mixed string and int tokens",
			url:  "https://api.sportmonks.com/v3/football/seasons/{seasonID}/date/{date}",
			tokens: map[string]interface{}{
				"season": 23614,
				"date":   "2026-11-13",
			},
			want: "https://api.sportmonks.com/v3/football/seasons/23614/date/2026-11-13",
		},
		{
			name: "league ID token",
			url:  "https://api.sportmonks.com/v3/football/standings/live/leagues/{leagueID}",
			tokens: map[string]interface{}{
				"leagues": 8,
			},
			want: "https://api.sportmonks.com/v3/football/standings/live/leagues/8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := replaceToken(tc.url, tc.tokens)
			if got != tc.want {
				t.Errorf("replaceToken() = %q, want %q", got, tc.want)
			}
		})
	}
}

