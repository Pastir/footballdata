package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/publisher"
	"github.com/jackc/pgx/v4/pgxpool"
)

// DBTaskProvider предоставляет созревшие задачи на публикацию на основе анализа реальных данных БД:
// 1. Вторничные / послематчевые итоги тура (roundResults / tuesdaySummary) по готовности закрытия туров.
// 2. Пятничные превью уикенда (weekendHub) перед первыми матчами пятницы.
type DBTaskProvider struct {
	pool      *pgxpool.Pool
	storage   publisher.Storage
	botToken  string
	channelID int64
	leagues   []int
}

// NewDBTaskProvider создаёт провайдер задач планировщика.
func NewDBTaskProvider(
	pool *pgxpool.Pool,
	storage publisher.Storage,
	botToken string,
	channelID int64,
	leagues []int,
) *DBTaskProvider {
	if len(leagues) == 0 {
		// По умолчанию 5 ведущих европейских чемпионатов:
		// 8 — АПЛ, 564 — Ла Лига, 384 — Серия А, 82 — Бундеслига, 301 — Лига 1
		leagues = []int{8, 564, 384, 82, 301}
	}
	return &DBTaskProvider{
		pool:      pool,
		storage:   storage,
		botToken:  botToken,
		channelID: channelID,
		leagues:   leagues,
	}
}

// CheckRoundStatusAll возвращает диагностическую информацию по статусу закрытия туров для всех отслеживаемых лиг.
func (p *DBTaskProvider) CheckRoundStatusAll(ctx context.Context, now time.Time) ([]analytics.RoundClosingStatus, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	var results []analytics.RoundClosingStatus
	for _, leagueID := range p.leagues {
		status, err := analytics.GetRoundClosingStatus(ctx, p.pool, leagueID, now)
		if err != nil {
			// Пропускаем лигу или фиксируем ошибку
			continue
		}
		results = append(results, *status)
	}
	return results, nil
}

// GetDueTasks возвращает список публикаций, готовых к отправке в канал на текущий момент времени.
func (p *DBTaskProvider) GetDueTasks(ctx context.Context, now time.Time) ([]PublicationTask, error) {
	var tasks []PublicationTask

	// 1. Проверяем итоги туров (roundResults) по каждой лиге
	for _, leagueID := range p.leagues {
		status, err := analytics.GetRoundClosingStatus(ctx, p.pool, leagueID, now)
		if err != nil {
			continue
		}

		if !status.IsReadyForClosing {
			continue
		}

		st := *status

		// 1.1 Проверяем результаты тура (roundResults)
		if p.storage != nil {
			already, err := p.storage.IsAlreadyPublished(ctx, p.channelID, publisher.PubTypeRoundResults, status.LeagueID, status.RoundID)
			if err == nil && !already {
				// Stale check
				if !status.EndingAt.IsZero() && now.Sub(status.EndingAt) > 36*time.Hour {
					_ = p.storage.RecordSkipped(ctx, p.channelID, publisher.PubTypeRoundResults, status.LeagueID, status.RoundID,
						fmt.Sprintf("stale round results: ended at %s (>36h ago)", status.EndingAt.Format("2006-01-02 15:04")))
				} else {
					tasks = append(tasks, PublicationTask{
						ChannelID:       p.channelID,
						BotToken:        p.botToken,
						PublicationType: publisher.PubTypeRoundResults,
						LeagueID:        st.LeagueID,
						SeasonID:        st.SeasonID,
						RoundID:         st.RoundID,
						Title:           fmt.Sprintf("%s — %s Результаты тура", st.LeagueName, st.RoundName),
						ExecuteFunc: func(execCtx context.Context) error {
							var td config.TaskData
							td.Init()
							td.Args = config.LineArgs{
								TaskName: "PublishToTelegram",
								Type:     "roundResults",
								LeagueId: st.LeagueID,
								RoundId:  st.RoundID,
								Publish:  true,
							}
							return publisher.Dispatch(execCtx, td)
						},
					})
				}
			}
		}

		// 1.2 Автопилот: Карточка аномалии и драмы тура (banter)
		if p.storage != nil {
			alreadyBanter, err := p.storage.IsAlreadyPublished(ctx, p.channelID, publisher.PubTypeBanter, status.LeagueID, status.RoundID)
			if err == nil && !alreadyBanter {
				if status.EndingAt.IsZero() || now.Sub(status.EndingAt) <= 36*time.Hour {
					anomalies, err := analytics.FindRoundAnomalies(ctx, p.pool, status.LeagueID, status.RoundID)
					if err == nil && len(anomalies) > 0 && anomalies[0].SeverityScore >= 75.0 {
						best := anomalies[0]
						tasks = append(tasks, PublicationTask{
							ChannelID:       p.channelID,
							BotToken:        p.botToken,
							PublicationType: publisher.PubTypeBanter,
							LeagueID:        st.LeagueID,
							SeasonID:        st.SeasonID,
							RoundID:         st.RoundID,
							Title:           fmt.Sprintf("%s — %s Карточка драмы (%s, Score: %.1f)", st.LeagueName, st.RoundName, best.BadgeText, best.SeverityScore),
							ExecuteFunc: func(execCtx context.Context) error {
								var td config.TaskData
								td.Init()
								td.Args = config.LineArgs{
									TaskName: "PublishToTelegram",
									Type:     "banter",
									LeagueId: st.LeagueID,
									RoundId:  st.RoundID,
									Publish:  true,
								}
								return publisher.Dispatch(execCtx, td)
							},
						})
					}
				}
			}
		}
	}

	// 2. Проверяем пятничный пакет расписания уикенда (weekendHub)
	// Расписание публикуется в пятницу перед первыми матчами (начиная с 09:00 UTC)
	if now.Weekday() == time.Friday || now.Weekday() == time.Saturday {
		hubDue, nextRoundID, err := p.checkWeekendHubDue(ctx, now)
		if err == nil && hubDue {
			tasks = append(tasks, PublicationTask{
				ChannelID:       p.channelID,
				BotToken:        p.botToken,
				PublicationType: publisher.PubTypeRoundFixtures,
				LeagueID:        8, // Флагманская лига — АПЛ
				RoundID:         nextRoundID,
				Title:           "Пакет расписания уикенда (Топ-5 лиг)",
				ExecuteFunc: func(execCtx context.Context) error {
					var td config.TaskData
					td.Init()
					td.Args = config.LineArgs{
						TaskName: "PublishToTelegram",
						Type:     "weekendHub",
						LeagueId: 8,
						RoundId:  nextRoundID,
						Publish:  true,
					}
					return publisher.Dispatch(execCtx, td)
				},
			})
		}
	}

	return tasks, nil
}

// checkWeekendHubDue определяет, созрела ли публикация расписания уикенда.
func (p *DBTaskProvider) checkWeekendHubDue(ctx context.Context, now time.Time) (bool, int, error) {
	if p.pool == nil {
		return false, 0, fmt.Errorf("database pool is nil")
	}

	// Берём контекст АПЛ как флагманской лиги
	info, err := analytics.GetLeagueSeasonInfo(ctx, p.pool, 8)
	if err != nil {
		return false, 0, err
	}

	if info.NextRoundID == 0 {
		return false, 0, nil
	}

	// Проверяем, не публиковался ли уже уикенд-хаб для этого раунда
	if p.storage != nil {
		already, err := p.storage.IsAlreadyPublished(ctx, p.channelID, publisher.PubTypeRoundFixtures, 8, info.NextRoundID)
		if err == nil && already {
			return false, info.NextRoundID, nil
		}
	}

	// Проверяем время первого матча ближайшего тура
	var firstMatchKickoff *time.Time
	queryFirst := `SELECT MIN(starting_at) FROM provider_fixtures WHERE round_id = $1 AND state_id IN (1, 13, 26)`
	_ = p.pool.QueryRow(ctx, queryFirst, info.NextRoundID).Scan(&firstMatchKickoff)

	if firstMatchKickoff == nil {
		return false, info.NextRoundID, nil
	}

	// Если первый матч уже начался, публикация расписания тура устарела (спам).
	// Помечаем задачу как SKIPPED.
	if now.After(*firstMatchKickoff) {
		if p.storage != nil {
			_ = p.storage.RecordSkipped(ctx, p.channelID, publisher.PubTypeRoundFixtures, 8, info.NextRoundID,
				fmt.Sprintf("first match kickoff has passed (%s)", firstMatchKickoff.Format("2006-01-02 15:04")))
		}
		return false, info.NextRoundID, nil
	}

	// Публикуем, если до первого матча осталось менее 12 часов, либо наступило 09:00 UTC пятницы
	timeToMatch := firstMatchKickoff.Sub(now)
	if timeToMatch > 0 && timeToMatch <= 12*time.Hour {
		return true, info.NextRoundID, nil
	}

	fridayMorning := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC)
	if now.Weekday() == time.Friday && (now.Equal(fridayMorning) || now.After(fridayMorning)) && timeToMatch > 0 {
		return true, info.NextRoundID, nil
	}

	return false, info.NextRoundID, nil
}
