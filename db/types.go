package db

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

type (

	//BookmakerData - содержит данные о типах ставок или о букмекерах
	BookmakerData struct {
		Id   int
		Name string
		UUID string
	}

	//MappingType Маппинг внутреннего названия с оригинальным
	MappingType struct {
		Name string
		UUID string
	}

	PointRules struct {
		Win  int `json:"win"`
		Draw int `json:"draw"`
		Loss int `json:"loss"`
	}

	SeasonData struct {
		SeasonId  int
		SeasonUid string
		Season    string
		LeagueId  int
		LeagueUid string
	}

	Statistics struct {
		Odds                       bool
		Players                    bool
		FixturesEvent              bool
		FixturesLineups            bool
		FixturesStatisticsPlayers  bool
		FixturesStatisticsFixtures bool
		Standings                  bool
		Predictions                bool
		TopScorers                 bool
		Current                    bool
		Type                       string
		LeagueId                   int
		SeasonId                   int
		Show                       bool
	}

	ListMatches struct {
		MatchId      int
		LeagueId     int
		SeasonId     int
		RoundId      int
		StageId      int
		HomeTeamId   int
		AwayTeamId   int
		HomeTeamName string
		AwayTeamName string
		LeagueName   string
	}
)

func (p *PointRules) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &p)
}

// SetOdds Установка статусов
func (s *Statistics) SetOdds(b bool) {
	s.Odds = b
}

func (s *Statistics) SetPlayers(b bool) {
	s.Players = b
}

func (s *Statistics) SetFixturesEvent(b bool) {
	s.FixturesEvent = b
}

func (s *Statistics) SetFixturesLineups(b bool) {
	s.FixturesLineups = b
}

func (s *Statistics) SetFixturesStatisticsPlayers(b bool) {
	s.FixturesStatisticsPlayers = b
}

func (s *Statistics) SetFixturesStatisticsFixtures(b bool) {
	s.FixturesStatisticsFixtures = b
}

func (s *Statistics) SetStandings(b bool) {
	s.Standings = b
}

func (s *Statistics) SetPredictions(b bool) {
	s.Predictions = b
}

func (s *Statistics) SetTopScorers(b bool) {
	s.TopScorers = b
}

func (s *Statistics) SetCurrent(b bool) {
	s.Current = b
}

func (s *Statistics) SetType(t string) {
	s.Type = t
}

func (s *Statistics) SetLeague(t int) {
	s.LeagueId = t
}

func (s *Statistics) SetSeason(t int) {
	s.SeasonId = t
}

func (s *Statistics) SetShow(t int) {
	s.SeasonId = t
}

func (s *Statistics) GetStringWhere() string {
	q := ""

	if s.Odds {
		q += " AND (mcs.coverage->'standings')::BOOLEAN"
	}
	if s.Players {
		q += " AND (mcs.coverage->'players')::BOOLEAN"
	}
	if s.FixturesEvent {
		q += " AND (mcs.coverage->'fixtures'->>'events')::BOOLEAN"
	}
	if s.FixturesLineups {
		q += " AND (mcs.coverage->'fixtures'->>'lineups')::BOOLEAN"
	}
	if s.FixturesStatisticsPlayers {
		q += " AND (mcs.coverage->'fixtures'->>'statistics_players')::BOOLEAN"
	}
	if s.FixturesStatisticsFixtures {
		q += " AND (mcs.coverage->'fixtures'->>'statistics_fixtures')::BOOLEAN"
	}
	if s.Standings {
		q += " AND (mcst.has_standings)::BOOLEAN"
	}
	if s.Predictions {
		q += " AND (mcs.coverage->'predictions')::BOOLEAN"
	}
	if s.TopScorers {
		q += " AND (mcs.coverage->'top_scorers')::BOOLEAN"
	}
	if s.Current {
		q += " AND mcs.current"
	}

	if s.SeasonId > 0 {
		q += " AND mcs.id = " + strconv.Itoa(s.SeasonId)
	}

	if s.LeagueId > 0 {
		q += " AND mcl.id = " + strconv.Itoa(s.LeagueId)
	}

	if s.Type == "League" {
		q += " AND mcl.type = 'League'"
	} else if s.Type == "Cup" {
		q += " AND mcl.type = 'Cup'"
	}

	q = strings.TrimLeft(q, " AND ")

	return q
}
