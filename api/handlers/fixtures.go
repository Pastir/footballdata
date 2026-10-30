package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/logs"
	"github.com/jackc/pgconn"
	"github.com/valyala/fastjson"
	"go.uber.org/zap"
)

/*
     Обновление матчей сезона, включая  participants и coaches
    ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.participants;fixtures.coaches -taskChild upsertFixturesSeasons

     Обновление результатов матчей по таймам
    ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.scores -taskChild upsertFixturesScores

    -task upsertFixturesSeasons -leagueId 2 -seasonIsCurrent true

    List functions
	UpsertFixtures(string, *fastjson.Value)
	UpsertFixtureScores(string, *fastjson.Value)
	UpsertFixtureLineups(string, *fastjson.Value)
	UpsertFixtureState(string, *fastjson.Value)
	UpsertFixtureStatistics(string, *fastjson.Value)
	UpsertFixtureWeatherReport(string, *fastjson.Value)
	UpsertFixtureReferees(string, *fastjson.Value)
	UpsertFixturePeriods(string, *fastjson.Value)
	UpsertFixtureMeta(string, *fastjson.Value)
    UpsertFixtureEvents(string, *fastjson.Value)
    UpsertFixtureTimeLines(string, *fastjson.Value)
*/

// UpsertFixturesId - fixture data processing
func UpsertFixturesId(d *fastjson.Value, taskData config.TaskData) {

	query := `INSERT INTO provider_fixtures(id, sport_id, league_id, season_id, stage_id, group_id, aggregate_id, 
                              round_id, state_id, venue_id, starting_at, leg, details, length, 
                              placeholder, has_odds, starting_at_timestamp) VALUES `

	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id              = EXCLUDED.sport_id
                                                      ,league_id             = EXCLUDED.league_id
                                                      ,season_id             = EXCLUDED.season_id
                                                      ,stage_id              = EXCLUDED.stage_id
                                                      ,group_id              = EXCLUDED.group_id
                                                      ,aggregate_id          = EXCLUDED.aggregate_id
                                                      ,round_id              = EXCLUDED.round_id
                                                      ,state_id              = EXCLUDED.state_id
                                                      ,venue_id              = EXCLUDED.venue_id
                                                      ,starting_at           = EXCLUDED.starting_at
                                                      ,leg                   = EXCLUDED.leg
                                                      ,details               = EXCLUDED.details
                                                      ,length                = EXCLUDED.length
                                                      ,placeholder           = EXCLUDED.placeholder
                                                      ,has_odds              = EXCLUDED.has_odds
                                                      ,starting_at_timestamp = EXCLUDED.starting_at_timestamp
                                                      ,updated_at            = NOW()`

	var qi string

	qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		db.CastInterfaceString(d.GetInt("data", "id")),
		db.CastInterfaceString(d.GetInt("data", "sport_id")),
		db.CastInterfaceString(d.GetInt("data", "league_id")),
		db.CastInterfaceString(d.GetInt("data", "season_id")),
		db.CastInterfaceString(d.GetInt("data", "stage_id")),
		db.CastInterfaceString(d.GetInt("data", "group_id")),
		db.CastInterfaceString(d.GetInt("data", "aggregate_id")),
		db.CastInterfaceString(d.GetInt("data", "round_id")),
		db.CastInterfaceString(d.GetInt("data", "state_id")),
		db.CastInterfaceString(d.GetInt("data", "venue_id")),
		db.CastInterfaceString(string(d.GetStringBytes("data", "starting_at"))),
		db.CastInterfaceString(string(d.GetStringBytes("data", "leg"))),
		db.CastInterfaceString(string(d.GetStringBytes("data", "details"))),
		db.CastInterfaceString(d.GetInt("data", "length")),
		db.CastInterfaceString(d.GetBool("data", "placeholder")),
		db.CastInterfaceString(d.GetBool("data", "has_odds")),
		db.CastInterfaceString(d.GetInt("data", "starting_at_timestamp")),
	)

	if strings.Compare(qi, "") != 0 {
		query = query + qi + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskData.Name(), "UpsertFixturesId", err.(*pgconn.PgError))
		}
	}
}

// UpsertFixtures - fixture data processing
func UpsertFixtures(taskName string, d *fastjson.Value) {

	data := d.GetArray()

	query := `INSERT INTO provider_fixtures(id, sport_id, league_id, season_id, stage_id, group_id, aggregate_id, 
                              round_id, state_id, venue_id, starting_at, leg, details, length, 
                              placeholder, has_odds, starting_at_timestamp) VALUES `

	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET sport_id              = EXCLUDED.sport_id
                                                      ,league_id             = EXCLUDED.league_id
                                                      ,season_id             = EXCLUDED.season_id
                                                      ,stage_id              = EXCLUDED.stage_id
                                                      ,group_id              = EXCLUDED.group_id
                                                      ,aggregate_id          = EXCLUDED.aggregate_id
                                                      ,round_id              = EXCLUDED.round_id
                                                      ,state_id              = EXCLUDED.state_id
                                                      ,venue_id              = EXCLUDED.venue_id
                                                      ,starting_at           = EXCLUDED.starting_at
                                                      ,leg                   = EXCLUDED.leg
                                                      ,details               = EXCLUDED.details
                                                      ,length                = EXCLUDED.length
                                                      ,placeholder           = EXCLUDED.placeholder
                                                      ,has_odds              = EXCLUDED.has_odds
                                                      ,starting_at_timestamp = EXCLUDED.starting_at_timestamp
                                                      ,updated_at            = NOW()`

	var qi string
	for _, v := range data {

		qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),`,
			db.CastInterfaceString(v.GetInt("id")),
			db.CastInterfaceString(v.GetInt("sport_id")),
			db.CastInterfaceString(v.GetInt("league_id")),
			db.CastInterfaceString(v.GetInt("season_id")),
			db.CastInterfaceString(v.GetInt("stage_id")),
			db.CastInterfaceString(v.GetInt("group_id")),
			db.CastInterfaceString(v.GetInt("aggregate_id")),
			db.CastInterfaceString(v.GetInt("round_id")),
			db.CastInterfaceString(v.GetInt("state_id")),
			db.CastInterfaceString(v.GetInt("venue_id")),
			db.CastInterfaceString(string(v.GetStringBytes("starting_at"))),
			db.CastInterfaceString(string(v.GetStringBytes("leg"))),
			db.CastInterfaceString(string(v.GetStringBytes("details"))),
			db.CastInterfaceString(v.GetInt("length")),
			db.CastInterfaceString(v.GetBool("placeholder")),
			db.CastInterfaceString(v.GetBool("has_odds")),
			db.CastInterfaceString(v.GetInt("starting_at_timestamp")),
		)
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskName, "UpsertFixtures", err.(*pgconn.PgError))
		}
	}
}

// UpsertFixtureScores - scores data processing
// Included in fixtures
func UpsertFixtureScores(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_fixtures_scores(id,fixture_id,type_id,participant_id,participant,goals,description) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET fixture_id      = EXCLUDED.fixture_id
                                                      ,type_id        = EXCLUDED.type_id
                                                      ,participant_id = EXCLUDED.participant_id
                                                      ,participant    = EXCLUDED.participant
                                                      ,goals          = EXCLUDED.goals
                                                      ,description    = EXCLUDED.description                                                  
                                                      ,updated_at     = NOW()`
	var qi string
	for _, v := range d.GetArray() {
		for _, sc := range v.GetArray("scores") {
			qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(sc.GetInt("id")),
				db.CastInterfaceString(sc.GetInt("fixture_id")),
				db.CastInterfaceString(sc.GetInt("type_id")),
				db.CastInterfaceString(sc.GetInt("participant_id")),
				db.CastInterfaceString(string(sc.GetStringBytes("score", "participant"))),
				db.CastInterfaceString(sc.GetInt("score", "goals")),
				db.CastInterfaceString(string(sc.GetStringBytes("description"))),
			)
		}
	}
	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskName, "UpsertScores", err.(*pgconn.PgError))
		}
	}
}

// UpsertFixtureLineups -  fixture lineups data processing
// Included in fixtures
func UpsertFixtureLineups(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_lineups(id, sport_id, fixture_id, league_id, player_id, team_id, position_id, 
                             detailed_position_id, formation_field, type_id, jersey_number, formation_position) VALUES `

	conflictQuery := ` ON CONFLICT (id, league_id) DO UPDATE SET sport_id              = EXCLUDED.sport_id
                                                      ,fixture_id           = EXCLUDED.fixture_id
                                                      ,league_id            = EXCLUDED.league_id
                                                      ,player_id            = EXCLUDED.player_id
                                                      ,team_id              = EXCLUDED.team_id
                                                      ,position_id          = EXCLUDED.position_id
                                                      ,detailed_position_id = EXCLUDED.detailed_position_id
                                                      ,formation_field      = EXCLUDED.formation_field
                                                      ,type_id              = EXCLUDED.type_id
                                                      ,jersey_number        = EXCLUDED.jersey_number
                                                      ,formation_position   = EXCLUDED.formation_position
                                                      ,updated_at           = NOW()`

	var qi string
	for _, fx := range d.GetArray() {
		for _, v := range fx.GetArray("lineups") {
			qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),`,
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("sport_id")),
				db.CastInterfaceString(v.GetInt("fixture_id")),
				db.CastInterfaceString(fx.GetInt("league_id")),
				db.CastInterfaceString(v.GetInt("player_id")),
				db.CastInterfaceString(v.GetInt("team_id")),
				db.CastInterfaceString(v.GetInt("position_id")),
				db.CastInterfaceString(v.GetInt("detailed_position_id")),
				db.CastInterfaceString(string(v.GetStringBytes("formation_field"))),
				db.CastInterfaceString(v.GetInt("type_id")),
				db.CastInterfaceString(v.GetInt("jersey_number")),
				db.CastInterfaceString(v.GetInt("formation_position")),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			// error handling violates foreign key constraint
			if err.(*pgconn.PgError).Code == "23503" {
				HandlingViolatesFKError(err, query)
			} else {
				db.PGxConnError(taskName, "UpsertFixtureLineups", err.(*pgconn.PgError))
			}
		}
	}
}

// UpsertFixtureStatistics - fixture statistics processing
// Included in fixtures
func UpsertFixtureStatistics(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_fixtures_statistics(id, fixture_id, league_id, type_id, participant_id, value, location) VALUES `

	conflictQuery := ` ON CONFLICT (fixture_id, league_id, participant_id, type_id) DO UPDATE SET id = EXCLUDED.id
                                                      ,value          = EXCLUDED.value
                                                      ,location       = EXCLUDED.location 
                                                      ,updated_at     = NOW()`

	var qi string
	for _, fx := range d.GetArray() {
		for _, v := range fx.GetArray("statistics") {
			qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s),`,
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("fixture_id")),
				db.CastInterfaceString(fx.GetInt("league_id")),
				db.CastInterfaceString(v.GetInt("type_id")),
				db.CastInterfaceString(v.GetInt("participant_id")),
				db.CastInterfaceString(v.GetInt("data", "value")),
				db.CastInterfaceString(string(v.GetStringBytes("location"))),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskName, "UpsertFixtureStatistics", err.(*pgconn.PgError))
		}
	}
}

// UpsertFixtureWeatherReport -  Weather report processing
// Included in fixtures
func UpsertFixtureWeatherReport(taskName string, d *fastjson.Value) {

	data := d.GetArray()

	query := `INSERT INTO provider_weather_report(id, fixture_id, venue_id, temperature_day, temperature_morning, temperature_evening, 
    temperature_night, feels_like_day, feels_like_morning, feels_like_evening, feels_like_night, wind_speed, 
    wind_direction, humidity, pressure, clouds, description, icon, type, metric) VALUES `

	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET fixture_id           = EXCLUDED.fixture_id
                                                      ,venue_id            = EXCLUDED.venue_id
                                                      ,temperature_day     = EXCLUDED.temperature_day
                                                      ,temperature_morning = EXCLUDED.temperature_morning
                                                      ,temperature_evening = EXCLUDED.temperature_evening
                                                      ,temperature_night   = EXCLUDED.temperature_night
                                                      ,feels_like_day      = EXCLUDED.feels_like_day
                                                      ,feels_like_morning  = EXCLUDED.feels_like_morning
                                                      ,feels_like_evening  = EXCLUDED.feels_like_evening
                                                      ,feels_like_night    = EXCLUDED.feels_like_night
                                                      ,wind_speed          = EXCLUDED.wind_speed
                                                      ,wind_direction      = EXCLUDED.wind_direction
                                                      ,humidity            = EXCLUDED.humidity
                                                      ,pressure            = EXCLUDED.pressure
                                                      ,clouds              = EXCLUDED.clouds
                                                      ,description         = EXCLUDED.description
                                                      ,icon                = EXCLUDED.icon
                                                      ,type                = EXCLUDED.type
                                                      ,metric              = EXCLUDED.metric
                                                      ,updated_at          = NOW()`

	var qi string
	for _, fx := range data {
		if fx.GetInt("weather_report", "id") != 0 {
			qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),`,
				db.CastInterfaceString(fx.GetInt("weather_report", "id")),
				db.CastInterfaceString(fx.GetInt("weather_report", "fixture_id")),
				db.CastInterfaceString(fx.GetInt("weather_report", "venue_id")),
				db.CastInterfaceString(fx.GetFloat64("weather_report", "temperature", "day")),
				db.CastInterfaceString(fx.GetFloat64("weather_report", "temperature", "morning")),
				db.CastInterfaceString(fx.GetFloat64("weather_report", "temperature", "evening")),
				db.CastInterfaceString(fx.GetFloat64("weather_report", "temperature", "night")),
				db.CastInterfaceString(fx.GetFloat64("weather_report", "feels_like", "day")),
				db.CastInterfaceString(fx.GetFloat64("weather_report", "feels_like", "morning")),
				db.CastInterfaceString(fx.GetFloat64("weather_report", "feels_like", "evening")),
				db.CastInterfaceString(fx.GetFloat64("weather_report", "feels_like", "night")),
				db.CastInterfaceString(fx.GetFloat64("weather_report", "wind", "speed")),
				db.CastInterfaceString(fx.GetInt("weather_report", "wind", "direction")),
				db.CastInterfaceString(string(fx.GetStringBytes("weather_report", "humidity"))),
				db.CastInterfaceString(fx.GetInt("weather_report", "pressure")),
				db.CastInterfaceString(string(fx.GetStringBytes("weather_report", "clouds"))),
				db.CastInterfaceString(string(fx.GetStringBytes("weather_report", "description"))),
				db.CastInterfaceString(string(fx.GetStringBytes("weather_report", "icon"))),
				db.CastInterfaceString(string(fx.GetStringBytes("weather_report", "type"))),
				db.CastInterfaceString(string(fx.GetStringBytes("weather_report", "metric"))),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskName, "UpsertFixtureWeatherReport", err.(*pgconn.PgError))
		}
	}
}

// UpsertFixtureReferees - fixture referees statistics processing
// Included in fixtures
func UpsertFixtureReferees(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_fixtures_referees(id, fixture_id, referee_id, type_id) VALUES `

	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET fixture_id      = EXCLUDED.fixture_id
                                                      ,referee_id     = EXCLUDED.referee_id
                                                      ,type_id        = EXCLUDED.type_id
                                                      ,updated_at     = NOW()`

	var qi string
	for _, fx := range d.GetArray() {
		for _, v := range fx.GetArray("referees") {
			qi += fmt.Sprintf(`(%s, %s, %s, %s),`,
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("fixture_id")),
				db.CastInterfaceString(v.GetInt("referee_id")),
				db.CastInterfaceString(v.GetInt("type_id")),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskName, "UpsertFixtureReferees", err.(*pgconn.PgError))
		}
	}
}

// UpsertFixturePeriods - fixture periods statistics processing
// Included in fixtures
func UpsertFixturePeriods(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_fixtures_periods(id, fixture_id, type_id, started, ended, 
                                      counts_from, actual_period_start, ticking, sort_order, time_added, minutes, 
                                      seconds) VALUES `

	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET fixture_id           = EXCLUDED.fixture_id
                                                      ,type_id             = EXCLUDED.type_id
                                                      ,started             = EXCLUDED.started
                                                      ,ended               = EXCLUDED.ended
                                                      ,counts_from         = EXCLUDED.counts_from
                                                      ,actual_period_start = EXCLUDED.actual_period_start
                                                      ,ticking             = EXCLUDED.ticking
                                                      ,sort_order          = EXCLUDED.sort_order
                                                      ,time_added          = EXCLUDED.time_added
                                                      ,minutes             = EXCLUDED.minutes
                                                      ,seconds             = EXCLUDED.seconds
                                                      ,updated_at          = NOW()`

	var qi string
	for _, fx := range d.GetArray() {
		for _, v := range fx.GetArray("periods") {
			qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),`,
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("fixture_id")),
				db.CastInterfaceString(v.GetInt("type_id")),
				db.CastInterfaceString(v.GetInt("started")),
				db.CastInterfaceString(v.GetInt("ended")),
				db.CastInterfaceString(v.GetInt("counts_from")),
				db.CastInterfaceString(v.GetInt("actual_period_start")),
				db.CastInterfaceString(fx.GetBool("ticking")),
				db.CastInterfaceString(v.GetInt("sort_order")),
				db.CastInterfaceString(v.GetInt("time_added")),
				db.CastInterfaceString(v.GetInt("minutes")),
				db.CastInterfaceString(v.GetInt("seconds")),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskName, "UpsertFixturePeriods", err.(*pgconn.PgError))
		}
	}
}

// UpsertFixturePeriodsStatistics - fixture statistics periods processing
func UpsertFixturePeriodsStatistics(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_fixtures_periods_statistics(id, period_id, fixture_id, type_id, period_type_id, participant_id, value, location) VALUES `

	conflictQuery := ` ON CONFLICT (fixture_id, participant_id, period_id, type_id) DO UPDATE SET id = EXCLUDED.id
                                                      ,value          = EXCLUDED.value
                                                      ,location       = EXCLUDED.location 
                                                      ,period_type_id = EXCLUDED.period_type_id
                                                      ,updated_at     = NOW()`

	for _, fx := range d.GetArray() {
		var qi string
		for _, p := range fx.GetArray("periods") {
			for _, v := range p.GetArray("statistics") {
				qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s),`,
					db.CastInterfaceString(v.GetInt("id")),
					db.CastInterfaceString(v.GetInt("period_id")),
					db.CastInterfaceString(v.GetInt("fixture_id")),
					db.CastInterfaceString(v.GetInt("type_id")),
					db.CastInterfaceString(p.GetInt("type_id")),
					db.CastInterfaceString(v.GetInt("participant_id")),
					db.CastInterfaceString(v.GetInt("data", "value")),
					db.CastInterfaceString(string(v.GetStringBytes("location"))),
				)
			}
		}

		if strings.Compare(qi, "") != 0 {
			insertStr := query + strings.TrimRight(qi, ",") + conflictQuery
			_, err := db.DBx.PGxPool.Exec(context.Background(), insertStr)
			if err != nil {
				db.PGxConnError(taskName, "UpsertFixturePeriodsStatistics", err.(*pgconn.PgError))
			}
		}
	}
}

// UpsertFixtureMeta - fixture metadata statistics processing
// Included in fixtures
func UpsertFixtureMeta(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_fixtures_meta(id, fixture_id, type_id, value) VALUES `

	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET fixture_id      = EXCLUDED.fixture_id
                                                      ,type_id        = EXCLUDED.type_id
                                                      ,value          = EXCLUDED.value
                                                      ,updated_at     = NOW()`

	var qi string
	for _, fx := range d.GetArray() {
		for _, v := range fx.GetArray("metadata") {
			if v.GetObject("values") != nil {
				qi += fmt.Sprintf(`(%s, %s, %s, %s),`,
					db.CastInterfaceString(v.GetInt("id")),
					db.CastInterfaceString(v.GetInt("metadatable_id")),
					db.CastInterfaceString(v.GetInt("type_id")),
					db.CastInterfaceString(string(v.GetObject("values").String())),
				)
			}
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskName, "UpsertFixtureMeta", err.(*pgconn.PgError))
		}
	}
}

// UpsertFixturePlayersStatistics - fixture players statistics processing
// Included in fixtures
func UpsertFixturePlayersStatistics(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_fixtures_players_statistics(id, lineup_id, fixture_id, league_id, type_id, player_id, team_id, value) VALUES `

	conflictQuery := ` ON CONFLICT (id, league_id) DO UPDATE SET lineup_id      = EXCLUDED.lineup_id
                                                      ,fixture_id    = EXCLUDED.fixture_id
                                                      ,type_id       = EXCLUDED.type_id
                                                      ,player_id     = EXCLUDED.player_id
                                                      ,team_id       = EXCLUDED.team_id
                                                      ,league_id      = EXCLUDED.league_id
                                                      ,value         = EXCLUDED.value
                                                      ,updated_at    = NOW()`

	var qi string
	for _, fx := range d.GetArray() {
		for _, lu := range fx.GetArray("lineups") {
			for _, dt := range lu.GetArray("details") {
				if lu.GetInt("player_id") > 0 {
					qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s),`,
						db.CastInterfaceString(dt.GetInt("id")),
						db.CastInterfaceString(dt.GetInt("lineup_id")),
						db.CastInterfaceString(dt.GetInt("fixture_id")),
						db.CastInterfaceString(fx.GetInt("league_id")),
						db.CastInterfaceString(dt.GetInt("type_id")),
						db.CastInterfaceString(lu.GetInt("player_id")),
						db.CastInterfaceString(dt.GetInt("team_id")),
						db.CastInterfaceString(dt.GetFloat64("data", "value")),
					)
				} else {
					logs.Logs["error"].Error("UpsertFixturePlayersStatistics",
						zap.String("data", dt.String()))
				}
			}
		}

		if strings.Compare(qi, "") != 0 {
			queryItog := query + strings.TrimRight(qi, ",") + conflictQuery
			_, err := db.DBx.PGxPool.Exec(context.Background(), queryItog)
			if err != nil {
				// error handling violates foreign key constraint
				if err.(*pgconn.PgError).Code == "23503" {
					HandlingViolatesFKError(err, query)
				} else {
					db.PGxConnError(taskName, "UpsertFixtureParticipants", err.(*pgconn.PgError))
				}
			}
		}
		qi = ""
	}
}

// UpsertFixtureEvents - event processing
// Included in fixtures
func UpsertFixtureEvents(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_events(id, fixture_id, period_id, participant_id, type_id, section, player_id, 
                            related_player_id, result, info, addition, minute, extra_minute, injured, on_bench, 
                            coach_id, sub_type_id) VALUES `
	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET fixture_id         = EXCLUDED.fixture_id
                                                      ,period_id         = EXCLUDED.period_id
                                                      ,participant_id    = EXCLUDED.participant_id
                                                      ,type_id           = EXCLUDED.type_id
                                                      ,section           = EXCLUDED.section
                                                      ,player_id         = EXCLUDED.player_id
                                                      ,related_player_id = EXCLUDED.related_player_id
                                                      ,result            = EXCLUDED.result
                                                      ,info              = EXCLUDED.info
                                                      ,addition          = EXCLUDED.addition
                                                      ,minute            = EXCLUDED.minute
                                                      ,extra_minute      = EXCLUDED.extra_minute
                                                      ,injured           = EXCLUDED.injured
                                                      ,on_bench          = EXCLUDED.on_bench
                                                      ,coach_id          = EXCLUDED.coach_id
                                                      ,sub_type_id       = EXCLUDED.sub_type_id
                                                      ,updated_at        = NOW()`

	var qi string
	for _, fx := range d.GetArray() {
		for _, v := range fx.GetArray("events") {
			qi += fmt.Sprintf("(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),", db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("fixture_id")),
				db.CastInterfaceString(v.GetInt("period_id")),
				db.CastInterfaceString(v.GetInt("participant_id")),
				db.CastInterfaceString(v.GetInt("type_id")),
				db.CastInterfaceString(string(v.GetStringBytes("section"))),
				db.CastInterfaceString(v.GetInt("player_id")),
				db.CastInterfaceString(v.GetInt("related_player_id")),
				db.CastInterfaceString(string(v.GetStringBytes("result"))),
				db.CastInterfaceString(string(v.GetStringBytes("info"))),
				db.CastInterfaceString(string(v.GetStringBytes("addition"))),
				db.CastInterfaceString(v.GetInt("minute")),
				db.CastInterfaceString(v.GetInt("extra_minute")),
				db.CastInterfaceString(v.GetBool("injured")),
				db.CastInterfaceString(v.GetBool("on_bench")),
				db.CastInterfaceString(v.GetInt("coach_id")),
				db.CastInterfaceString(v.GetInt("sub_type_id")),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			db.PGxConnError(taskName, "UpsertFixtureEvents", err.(*pgconn.PgError))
		}
	}
}

// UpsertFixtureTimeLines - timeline processing
// Included in fixtures
func UpsertFixtureTimeLines(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_timelines(id, fixture_id, period_id, participant_id, type_id, section, player_id, 
                               related_player_id, result, info, addition, minute, extra_minute, injured, on_bench, 
                               coach_id, sub_type_id) VALUES `

	conflictQuery := ` ON CONFLICT (id) DO UPDATE SET fixture_id         = EXCLUDED.fixture_id
                                                      ,period_id         = EXCLUDED.period_id
                                                      ,participant_id    = EXCLUDED.participant_id
                                                      ,type_id           = EXCLUDED.type_id
                                                      ,section           = EXCLUDED.section
                                                      ,player_id         = EXCLUDED.player_id
                                                      ,related_player_id = EXCLUDED.related_player_id
                                                      ,result            = EXCLUDED.result
                                                      ,info              = EXCLUDED.info
                                                      ,addition          = EXCLUDED.addition
                                                      ,minute            = EXCLUDED.minute
                                                      ,extra_minute      = EXCLUDED.extra_minute
                                                      ,injured           = EXCLUDED.injured
                                                      ,on_bench          = EXCLUDED.on_bench
                                                      ,coach_id          = EXCLUDED.coach_id
                                                      ,sub_type_id       = EXCLUDED.sub_type_id
                                                      ,updated_at        = NOW()`

	var qi string

	for _, fx := range d.GetArray() {
		for _, v := range fx.GetArray("timeline") {
			qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s),`,
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(v.GetInt("fixture_id")),
				db.CastInterfaceString(v.GetInt("period_id")),
				db.CastInterfaceString(v.GetInt("participant_id")),
				db.CastInterfaceString(v.GetInt("type_id")),
				db.CastInterfaceString(string(v.GetStringBytes("section"))),
				db.CastInterfaceString(v.GetInt("player_id")),
				db.CastInterfaceString(v.GetInt("related_player_id")),
				db.CastInterfaceString(string(v.GetStringBytes("result"))),
				db.CastInterfaceString(string(v.GetStringBytes("info"))),
				db.CastInterfaceString(string(v.GetStringBytes("addition"))),
				db.CastInterfaceString(v.GetInt("minute")),
				db.CastInterfaceString(v.GetInt("extra_minute")),
				db.CastInterfaceString(v.GetBool("injured")),
				db.CastInterfaceString(v.GetBool("on_bench")),
				db.CastInterfaceString(v.GetInt("coach_id")),
				db.CastInterfaceString(v.GetInt("sub_type_id")),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		query = query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), query)
		if err != nil {
			// error handling violates foreign key constraint
			if err.(*pgconn.PgError).Code == "23503" {
				HandlingViolatesFKError(err, query)
			} else {
				db.PGxConnError(taskName, "UpsertFixtureParticipants", err.(*pgconn.PgError))
			}
		}
	}
}

// UpsertFixtureParticipants - Participants processing
// Include fixtures.participants
func UpsertFixtureParticipants(taskName string, d *fastjson.Value) {
	query := `INSERT INTO provider_fixtures_participants(fixture_id, participant_id, location, winner, position, result) VALUES `

	conflictQuery := ` ON CONFLICT (fixture_id, location) DO UPDATE SET winner     = EXCLUDED.winner
                                                      , participant_id = EXCLUDED.participant_id
                                                      , position   = EXCLUDED.position
                                                      , result     = EXCLUDED.result
                                                      , updated_at = NOW()`

	var qi string

	for _, fx := range d.GetArray() {
		winner0 := fx.GetBool("participants", "0", "meta", "winner")
		winner1 := fx.GetBool("participants", "1", "meta", "winner")
		var winner string

		for k, v := range fx.GetArray("participants") {

			if k == 0 {
				if !winner0 && !winner1 {
					winner = "D"
				} else if winner0 {
					winner = "W"
				} else if winner1 {
					winner = "L"
				}
			} else if k == 1 {
				if !winner0 && !winner1 {
					winner = "D"
				} else if winner0 {
					winner = "L"
				} else if winner1 {
					winner = "W"
				}
			}

			qi += fmt.Sprintf(`(%s, %s, %s, %s, %s, %s),`,
				db.CastInterfaceString(fx.GetInt("id")),
				db.CastInterfaceString(v.GetInt("id")),
				db.CastInterfaceString(string(v.GetStringBytes("meta", "location"))),
				db.CastInterfaceString(v.GetBool("meta", "winner")),
				db.CastInterfaceString(v.GetInt("meta", "position")),
				db.CastInterfaceString(winner),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		queryInsert := query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), queryInsert)
		if err != nil {

			// error handling violates foreign key constraint
			if err.(*pgconn.PgError).Code == "23503" {
				HandlingViolatesFKError(err, query)
			} else {
				db.PGxConnError(taskName, "UpsertFixtureParticipants", err.(*pgconn.PgError))
			}
		}
	}
}

// UpsertFixtureCoaches - Coaches processing
// Include fixtures.coaches
func UpsertFixtureCoaches(taskName string, d *fastjson.Value) {

	query := `INSERT INTO provider_fixtures_coaches(fixture_id, coaches_id, participant_id) VALUES `

	conflictQuery := ` ON CONFLICT (fixture_id, participant_id) DO UPDATE SET coaches_id = EXCLUDED.coaches_id
                                                      ,participant_id = EXCLUDED.participant_id
                                                      ,updated_at     = NOW()`

	var qi string

	for _, fx := range d.GetArray() {
		for _, v := range fx.GetArray("coaches") {
			qi += fmt.Sprintf(`(%s, %s, %s),`,
				db.CastInterfaceString(fx.GetInt("id")),
				db.CastInterfaceString(v.GetInt("meta", "coach_id")),
				db.CastInterfaceString(v.GetInt("meta", "participant_id")),
			)
		}
	}

	if strings.Compare(qi, "") != 0 {
		queryInsert := query + strings.TrimRight(qi, ",") + conflictQuery
		_, err := db.DBx.PGxPool.Exec(context.Background(), queryInsert)
		if err != nil {
			// error handling violates foreign key constraint
			if err.(*pgconn.PgError).Code == "23503" {
				UpsertCoachesFixturesErrorFK(d, taskName)
				_, err = db.DBx.PGxPool.Exec(context.Background(), queryInsert)
				if err != nil {
					db.PGxConnError(taskName, "Error violates foreign key constraint UpsertFixtureCoaches", err.(*pgconn.PgError))
				}
			} else {
				db.PGxConnError(taskName, "UpsertFixtureCoaches", err.(*pgconn.PgError))
			}
		}
	}
}

// UpsertFixturesSeasons Processes match season data, including additional statistics
func UpsertFixturesSeasons(d *fastjson.Value, taskData config.TaskData) {

	switch taskData.Args.TaskChild {
	case "upsertFixtures":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures -taskChild upsertFixturesSeasons -seasonIsCurrent true
		UpsertFixtures(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixturesScores":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.scores -taskChild upsertFixturesScores
		UpsertFixtureScores(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixtureEvents":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.events -taskChild upsertFixtureEvents
		UpsertFixtureEvents(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixtureTimeLines":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.timeline -taskChild upsertFixtureTimeLines -seasonIsCurrent true
		UpsertFixtureTimeLines(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixtureLineups":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.lineups -taskChild upsertFixtureLineups -seasonIsCurrent true
		UpsertFixtureLineups(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixtureStatistics":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.statistics -taskChild upsertFixtureStatistics -seasonIsCurrent true
		UpsertFixtureStatistics(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixtureWeatherReport":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.weatherReport -taskChild upsertFixtureWeatherReport -seasonIsCurrent true
		UpsertFixtureWeatherReport(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixtureReferees":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.referees -taskChild upsertFixtureReferees -seasonIsCurrent true
		UpsertFixtureReferees(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixturePeriods":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.periods -taskChild upsertFixturePeriods -seasonIsCurrent true
		UpsertFixturePeriods(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixtureMeta":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.metadata -taskChild upsertFixtureMeta -seasonIsCurrent true
		UpsertFixtureMeta(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixturePlayersStatistics":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.lineups.details -taskChild upsertFixturePlayersStatistics -seasonIsCurrent true
		UpsertFixturePlayersStatistics(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixtureParticipants":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.participants -taskChild upsertFixtureParticipants -seasonIsCurrent true
		UpsertFixtureParticipants(taskData.Name(), d.Get("data", "fixtures"))
	case "upsertFixtureCoaches":
		// ./prematch -conf config.yaml -task upsertFixturesSeasons -include fixtures.coaches -taskChild upsertFixtureCoaches
		UpsertFixtureCoaches(taskData.Name(), d.Get("data", "fixtures"))

	default:
		UpsertFixtures(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixtureScores(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixtureEvents(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixtureTimeLines(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixtureLineups(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixtureStatistics(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixtureWeatherReport(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixtureReferees(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixturePeriods(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixtureMeta(taskData.Name(), d.Get("data", "fixtures"))
		UpsertFixturePlayersStatistics(taskData.Name(), d.Get("data", "fixtures"))
	}
}

// UpsertFixturesBetweenDate
// ./prematch -conf config.yaml -task upsertFixturesDate -include scores -taskChild upsertFixturesScores
//
//	[ scores | events | timeline | lineups| statistics| weatherReport | referees| periods| metadata| coaches | statistics]
func UpsertFixturesBetweenDate(d *fastjson.Value, taskData config.TaskData) {

	switch taskData.Args.TaskChild {
	case "upsertFixtures":
		// ./prematch -conf config.yaml -task UpsertFixturesBetweenDate -taskChild upsertFixtures -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtures(taskData.Name(), d.Get("data"))
	case "upsertFixturesScores":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include scores -taskChild upsertFixturesScores -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureScores(taskData.Name(), d.Get("data"))
	case "upsertFixtureEvents":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include events -taskChild upsertFixtureEvents -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureEvents(taskData.Name(), d.Get("data"))
	case "upsertFixtureTimeLines":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include timeline -taskChild upsertFixtureTimeLines -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureTimeLines(taskData.Name(), d.Get("data"))
	case "upsertFixtureLineups":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include lineups -taskChild upsertFixtureLineups -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureLineups(taskData.Name(), d.Get("data"))
	case "upsertFixtureStatistics":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include statistics -taskChild upsertFixtureStatistics -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureStatistics(taskData.Name(), d.Get("data"))
	case "upsertFixtureWeatherReport":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include weatherReport -taskChild upsertFixtureWeatherReport -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureWeatherReport(taskData.Name(), d.Get("data"))
	case "upsertFixtureReferees":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include referees -taskChild upsertFixtureReferees -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureReferees(taskData.Name(), d.Get("data"))
	case "upsertFixturePeriods":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include periods -taskChild upsertFixturePeriods -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixturePeriods(taskData.Name(), d.Get("data"))
	case "upsertFixtureMeta":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include .metadata -taskChild upsertFixtureMeta -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureMeta(taskData.Name(), d.Get("data"))
	case "upsertFixturePlayersStatistics":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include lineups.details -taskChild upsertFixturePlayersStatistics -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixturePlayersStatistics(taskData.Name(), d.Get("data"))
	case "upsertFixtureParticipants":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include participants -taskChild upsertFixtureParticipants -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureParticipants(taskData.Name(), d.Get("data"))
	case "upsertFixtureCoaches":
		// ./prematch -conf config.yaml -task upsertFixturesBetweenDate -include coaches -taskChild upsertFixtureCoaches -startDate YYYY-MM-DD -endDate YYYY-MM-DD
		UpsertFixtureCoaches(taskData.Name(), d.Get("data"))

	default:
		UpsertFixtures(taskData.Name(), d.Get("data"))
		UpsertFixtureScores(taskData.Name(), d.Get("data"))
		UpsertFixtureEvents(taskData.Name(), d.Get("data"))
		UpsertFixtureTimeLines(taskData.Name(), d.Get("data"))
		UpsertFixtureLineups(taskData.Name(), d.Get("data"))
		UpsertFixtureStatistics(taskData.Name(), d.Get("data"))
		UpsertFixtureWeatherReport(taskData.Name(), d.Get("data"))
		UpsertFixtureReferees(taskData.Name(), d.Get("data"))
		UpsertFixturePeriods(taskData.Name(), d.Get("data"))
		UpsertFixtureMeta(taskData.Name(), d.Get("data"))
		UpsertFixturePlayersStatistics(taskData.Name(), d.Get("data"))
	}
}
