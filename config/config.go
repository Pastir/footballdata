package config

import (
	"flag"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

const defaultConfigFile = "./config.yaml"

var (
	Config = Conf{}
	Args   = LineArgs{}
)

type (
	TaskData struct {
		Args    LineArgs
		Replace map[string]interface{}
		Include string
		Cycle   int
	}

	LineArgs struct {
		conf     string
		Endpoint string
		//Имя задачи
		TaskName string
		//ID Лиги
		LeagueId int
		//ID Сезона
		SeasonsId     string
		PlayersId     string
		CoachesId     string
		FixturesId    string
		StartDate     string
		EndDate       string
		Date          string
		CurrentSeason bool
		NameBK        string
		TeamsId       string
		Include       string
		TaskChild     string
		Locale        string
		Type          string
		Output        string
		Publish       bool
		AsDocument    bool
		Timezone      string
		RoundId       int
		ProviderId    int
		Force         bool
		Style         string
		Port          string
		PollInterval  time.Duration
		StaticDir     string
	}

	env struct {
		Environment string `yaml:"Environment"`
	}

	pg struct {
		DbName            string        `yaml:"DbName"`
		DbUser            string        `yaml:"DbUser"`
		DbPass            string        `yaml:"DbPass"`
		DbPort            uint16        `yaml:"DbPort"`
		ConnectionTimeout time.Duration `yaml:"ConnectionTimeout"`
		DbHost            string        `yaml:"DbHost"`
		SslMode           string        `yaml:"SslMode"`
		MaxConnLifetime   time.Duration `yaml:"MaxConnLifetime"`
		MaxConnIdleTime   time.Duration `yaml:"MaxConnIdleTime"`
		MaxConns          int32         `yaml:"MaxConns"`
		MinConns          int32         `yaml:"MinConns"`
		HealthCheckPeriod time.Duration `yaml:"HealthCheckPeriod"`
	}

	ch struct {
		DbName string `yaml:"DbName"`
		DbUser string `yaml:"DbUser"`
		DbPass string `yaml:"DbPass"`
		DbPort int    `yaml:"DbPort"`
		DbHost string `yaml:"DbHost"`
	}

	logs struct {
		LogDir        string `yaml:"LogDir"`
		Debug         bool   `yaml:"Debug"`
		Warning       bool   `yaml:"Warning"`
		Info          bool   `yaml:"Info"`
		Error         bool   `yaml:"Error"`
		Critical      bool   `yaml:"Critical"`
		OutConsole    bool   `yaml:"OutConsole"`
		OutFile       bool   `yaml:"OutFile"`
		MaxFileSize   int    `yaml:"MaxFileSize"`
		MaxFilesCount int    `yaml:"MaxFilesCount"`
		MaxAge        int    `yaml:"MaxAge"`
		Compress      bool   `yaml:"Compress"`
	}

	// tgConf — настройки публикации в Telegram.
	tgConf struct {
		BotToken  string `yaml:"BotToken"`  // токен бота, выданный @BotFather
		ChannelID string `yaml:"ChannelID"` // ID канала: @handle или -1001234567890
	}

	// webAppConf — настройки Telegram Mini App / WebApp просмотра инфографики.
	webAppConf struct {
		BaseURL      string `yaml:"BaseURL"`
		BotShortName string `yaml:"BotShortName"`
		BotUsername  string `yaml:"BotUsername"`
		MediaDir     string `yaml:"MediaDir"`
	}

	service struct {
		ConfigFile string
	}

	// bookmakerConf — настройки букмекера-партнёра и реферальной программы.
	bookmakerConf struct {
		Name  string `yaml:"Name"`
		Promo string `yaml:"Promo"`
		Bonus string `yaml:"Bonus"`
		URL   string `yaml:"URL"`
	}

	// LLMProviderConf — настройки отдельного провайдера языковой модели (Gemini, DeepSeek, OpenAI).
	LLMProviderConf struct {
		APIKey  string `yaml:"api_key"`
		Model   string `yaml:"model"`
		BaseURL string `yaml:"base_url"`
		Timeout string `yaml:"timeout"`
	}

	// LLMConf — мультипровайдерная конфигурация языковых моделей для генерации текстов и дайджеста.
	LLMConf struct {
		ActiveProvider string                     `yaml:"active_provider"`
		FallbackOrder  []string                   `yaml:"fallback_order"`
		Providers      map[string]LLMProviderConf `yaml:"providers"`
	}

	Conf struct {
		Logs      *logs
		PG        *pg
		CH        *ch
		Env       *env
		TG        *tgConf        `yaml:"telegram"`
		WebApp    *webAppConf    `yaml:"webapp"`
		Bookmaker *bookmakerConf `yaml:"bookmaker"`
		LLM       *LLMConf       `yaml:"llm"`
		Server    service
		Args      LineArgs
	}
)

func (td *TaskData) Init() {
	td.Replace = make(map[string]interface{})
}

func (td *TaskData) GetProviderId() int { return td.Args.ProviderId }

func (td *TaskData) Name() string {
	return td.Args.TaskName
}

func (td *TaskData) GetEndpoint() string {
	return td.Args.Endpoint
}

func (td *TaskData) GetLeagueId() int {
	return td.Args.LeagueId
}

func (td *TaskData) GetSeasonsId() string {
	return td.Args.SeasonsId
}

func (td *TaskData) GetStartDate() string {
	return td.Args.StartDate
}

func (td *TaskData) GetEndDate() string {
	return td.Args.EndDate
}

func (td *TaskData) GetDate() string {
	return td.Args.Date
}

func (td *TaskData) GetCurrentSeason() bool {
	return td.Args.CurrentSeason
}

func (td *TaskData) GetNameBK() string {
	return td.Args.NameBK
}

func (td *TaskData) SetCurrentSeason() {
	td.Args.CurrentSeason = true
}

func (td *TaskData) SetReplace(key string, val interface{}) {
	td.Replace[key] = val
}

func (td *TaskData) GetArgs() LineArgs {
	return td.Args
}

func (td *TaskData) GetFixtureId() string {
	return td.Args.FixturesId
}

func (td *TaskData) GetPlayerId() string {
	return td.Args.PlayersId
}

func (td *TaskData) GetCoachesId() string {
	return td.Args.CoachesId
}

func (td *TaskData) GetTeamId() string {
	return td.Args.TeamsId
}

func (td *TaskData) GetTaskChild() string {
	return td.Args.TaskChild
}

func (td *TaskData) GetType() string {
	return td.Args.Type
}

func (td *TaskData) GetOutput() string {
	return td.Args.Output
}

func (td *TaskData) GetPublish() bool {
	return td.Args.Publish
}

func (td *TaskData) GetAsDocument() bool {
	return td.Args.AsDocument
}

func (td *TaskData) GetForce() bool {
	return td.Args.Force
}

func (td *TaskData) GetTimezone() string {
	if td.Args.Timezone == "" {
		return "UTC"
	}
	return td.Args.Timezone
}

func (td *TaskData) GetRoundId() int {
	return td.Args.RoundId
}

func (td *TaskData) GetLocale() string {
	if td.Args.Locale == "" {
		return "ru"
	}
	return td.Args.Locale
}

func (td *TaskData) GetStyle() string {
	return td.Args.Style
}

func (td *TaskData) GetPort() string {
	if td.Args.Port == "" {
		return "8080"
	}
	return td.Args.Port
}

func (td *TaskData) GetPollInterval() time.Duration {
	if td.Args.PollInterval <= 0 {
		return 20 * time.Second
	}
	return td.Args.PollInterval
}

func (td *TaskData) GetStaticDir() string {
	if td.Args.StaticDir == "" {
		return "./web/viewer"
	}
	return td.Args.StaticDir
}

func (c *Conf) Init() error {

	c.ParseCommandLine()

	// read configuration file
	source, err := os.ReadFile(c.Server.ConfigFile)
	if err != nil {
		return err
	}

	// Parsing yaml file
	err = yaml.Unmarshal(source, &c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Conf) Close() error { return nil }
func (c *Conf) Check() error { return nil }

func (c *Conf) Name() string {
	return "config"
}

func (c *Conf) ParseCommandLine() {
	flag.StringVar(&c.Server.ConfigFile, "conf", defaultConfigFile, "Path to services file")
	flag.StringVar(&c.Args.TaskName, "task", "", "-task upsertFixturesSeasons")
	flag.StringVar(&c.Args.TaskChild, "taskChild", "", "-taskChild upsertFixturesScores")
	flag.StringVar(&c.Args.Include, "include", "", "-include events;fixtures.scores")
	flag.IntVar(&c.Args.LeagueId, "leagueId", 0, " -leagueId 8")
	flag.StringVar(&c.Args.PlayersId, "playerId", "", " -playerId 1,2,3,45")
	flag.StringVar(&c.Args.CoachesId, "coachesId", "", " -coachesId 1,2,3,45")
	flag.BoolVar(&c.Args.CurrentSeason, "seasonIsCurrent", false, "choice current(s) seasons; arguments (false|true)")
	flag.StringVar(&c.Args.SeasonsId, "seasonsId", "", "-seasonsId 12,13")
	flag.StringVar(&c.Args.FixturesId, "fixturesId", "", "-fixturesId 12,13")
	flag.StringVar(&c.Args.StartDate, "startDate", "", "format 'YYYY-MM-DD'")
	flag.StringVar(&c.Args.EndDate, "endDate", "", "format 'YYYY-MM-DD'")
	flag.StringVar(&c.Args.Date, "date", "", "format 'YYYY-MM-DD'")
	flag.StringVar(&c.Args.NameBK, "nameBK", "", "winline|parimatch|etc...")
	flag.StringVar(&c.Args.TeamsId, "teamsId", "", "Teams IDs, example: 1,2,3,45")
	flag.StringVar(&c.Args.Locale, "locale", "ru", "-locale ru|en|ua")
	flag.StringVar(&c.Args.Locale, "l", "ru", "-l ru|en|ua (alias for -locale)")
	flag.StringVar(&c.Args.Type, "type", "", "-type table|roundFixtures|previewMatch|roundResults")
	flag.StringVar(&c.Args.Output, "output", "", "-output path/to/image.png")
	flag.BoolVar(&c.Args.Publish, "publish", false, "choice publish to telegram; arguments (false|true)")
	flag.BoolVar(&c.Args.AsDocument, "asDocument", false, "choice send media as uncompressed document; arguments (false|true)")
	flag.StringVar(&c.Args.Timezone, "timezone", "UTC", "-timezone UTC")
	flag.IntVar(&c.Args.RoundId, "roundId", 0, "-roundId 29")
	flag.IntVar(&c.Args.ProviderId, "providerId", 2, "-providerId 2")
	flag.BoolVar(&c.Args.Force, "force", false, "force republication even if already published")
	flag.StringVar(&c.Args.Style, "style", "", "-style 1|2|3 (варианты стиля подачи)")
	flag.StringVar(&c.Args.Port, "port", "8080", "HTTP server port (default 8080)")
	flag.DurationVar(&c.Args.PollInterval, "pollInterval", 20*time.Second, "In-play polling interval, e.g. 15s, 20s, 30s (default 20s)")
	flag.StringVar(&c.Args.StaticDir, "static", "./web/viewer", "Path to static web files directory (default ./web/viewer)")

	os.Args = normalizeBoolArgs(os.Args, "seasonIsCurrent", "publish", "asDocument", "force")
	flag.Parse()
}

func normalizeBoolArgs(args []string, boolFlags ...string) []string {
	if len(args) <= 1 {
		return args
	}
	res := make([]string, 0, len(args))
	res = append(res, args[0])
	boolSet := make(map[string]bool, len(boolFlags)*2)
	for _, b := range boolFlags {
		boolSet["-"+b] = true
		boolSet["--"+b] = true
	}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if boolSet[arg] && i+1 < len(args) {
			next := args[i+1]
			if next == "true" || next == "false" || next == "1" || next == "0" {
				res = append(res, arg+"="+next)
				i++
				continue
			}
		}
		res = append(res, arg)
	}
	return res
}
