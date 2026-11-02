package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Pastir/ManageServer"
	"github.com/Pastir/footballdata/analytics"
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/footballdata/db"
	"github.com/Pastir/footballdata/logs"
	"github.com/Pastir/footballdata/prediction"
	"github.com/Pastir/footballdata/press"
	"github.com/Pastir/footballdata/publisher"
	"github.com/Pastir/footballdata/scheduler"
	"github.com/Pastir/footballdata/server"
	"go.uber.org/zap"
)

func main() {
	var mServer = ManageServer.ServiceWatcher{}
	mServer.AddService(&config.Config)
	mServer.AddService(&logs.Logs)
	mServer.AddService(&db.DBx)

	if err := mServer.InitServices(); err != nil {
		fmt.Fprintf(os.Stderr, "service init error: %v\n", err)
		os.Exit(1)
	}

	var taskData config.TaskData
	taskData.Init()
	taskData.Args = config.Config.Args

	log := logs.Logs["access"]
	if log == nil {
		l, _ := zap.NewDevelopment()
		log = l
	}

	log.Info("lab task started",
		zap.String("ts", time.Now().Format(time.RFC3339)),
		zap.String("task", taskData.Name()),
		zap.String("type", taskData.GetType()),
		zap.Int("leagueId", taskData.GetLeagueId()),
	)

	switch taskData.Name() {
	case "PublishToTelegram":
		if err := publisher.Dispatch(context.Background(), taskData); err != nil {
			log.Error("publication task failed", zap.Error(err))
			fmt.Fprintf(os.Stderr, "❌ Ошибка публикации: %v\n", err)
			os.Exit(1)
		}

	case "CheckRoundClosing":
		handleCheckRoundClosing(context.Background(), taskData, log)

	case "SchedulerTick":
		handleSchedulerTick(context.Background(), taskData, log)

	case "SchedulerDaemon":
		handleSchedulerDaemon(context.Background(), taskData, log)

	case "SettleBets":
		handleSettleBets(context.Background(), taskData, log)

	case "ApiServer", "LiveServer", "Server":
		handleApiServer(context.Background(), taskData, log)

	case "PressDigest", "Press":
		handlePressDigest(context.Background(), taskData, log)

	case "AnomalyCard", "BanterCard":
		taskData.Args.Type = "banter"
		if err := publisher.Dispatch(context.Background(), taskData); err != nil {
			log.Error("banter card task failed", zap.Error(err))
			fmt.Fprintf(os.Stderr, "❌ Ошибка формирования карточки аномалии: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, `Использование: ./lab -conf config.yaml -task <TASK> [ФЛАГИ]

Доступные задачи (-task):
  PublishToTelegram  — Публикация конкретного материала по типу (-type)
  CheckRoundClosing  — Диагностика статуса закрытия туров лиг (-leagueId опционально)
  SchedulerTick      — Разовый цикл проверки и публикации созревших задач (для crontab)
  SchedulerDaemon    — Непрерывный фоновый планировщик публикаций
  SettleBets         — Авторасчёт открытых ставок/экспрессов по результатам матчей и отчёт ROI
  ApiServer          — REST API сервер для отдачи live и архивных данных туров (-port, -pollInterval, -static)
  PressDigest        — Сбор европейских таблоидов, генерация выжимки через Gemini и публикация (-publish)

Доступные типы публикаций (-type для task=PublishToTelegram):
  table           — Турнирная таблица (рендер PNG + пост в Telegram)
  roundFixtures   — Расписание следующего тура (котировки 1X2, H2H, Hero-xG, Silent Pin)
  schedule5Leagues — Пакетная публикация расписания 5 топ-лиг с «Матчем тура» и Silent Pin для АПЛ
  roundResults    — Результаты прошедшего тура (постер Топ-5 лидеров + нативная таблица с xG + Mini App)
  tuesdaySummary  — Вторничные итоги уикенда (постер лидеров 1920x1080 + интерактивная таблица xG)
  express         — Умный экспресс тура (Data-Driven ставка на базе xG с кнопкой БК и Mini App)
  previewMatch    — Превью конкретного матча (-fixturesId <ID>)
  roundDigest     — Интерактивный дайджест тура
  leaders         — Лидеры сезона: бомбардиры, ассистенты, Гол+Пас, сухие матчи
  facts           — Факты и серии: инфографика 1200x1200px + пост Telegram с Mini App
  recalculateStreaks — Принудительный перерасчёт серий и рекордов в таблице analytics_streaks
  catalog         — Каталог турниров (генерация active_leagues.json и синхронизация Mini App)
  syncAll         — Пакетная генерация данных для всех активных лиг (Mini App)
  banter          — Карточки футбольных аномалий и драмы (1200x1200px: ограбление по xG, валидол 90+, джокер)

Основные флаги:
  -conf string        Путь к файлу конфигурации (по умолчанию ./config.yaml)
  -leagueId int       ID лиги в базе (по умолчанию 8 — АПЛ; 0 — все топ-5 лиг)
  -roundId int        Переопределение ID раунда/тура
  -fixturesId string  ID матча для previewMatch
  -publish bool       Отправка в Telegram (true / false, по умолчанию false)
  -asDocument bool    Отправка графики документом без сжатия (true / false)
  -output string      Путь для сохранения PNG файла
  -locale string      Язык локализации названий команд (ru, en, ua; по умолчанию ru)
  -timezone string    Отображаемый часовой пояс (по умолчанию UTC)
  -force bool         Принудительная публикация (игнорировать защиту от дубликатов)
  -port string        Порт HTTP сервера для task=ApiServer (по умолчанию 8080)
  -pollInterval dur   Интервал in-play опроса матчей (по умолчанию 20s, e.g. 15s, 30s)
  -static string      Путь к статическим файлам Mini App (по умолчанию ./web/viewer)
`)
		os.Exit(1)
	}
}

func handleCheckRoundClosing(ctx context.Context, taskData config.TaskData, log *zap.Logger) {
	pool := db.DBx.PGxPool
	if pool == nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка: база данных не подключена\n")
		os.Exit(1)
	}

	leagues := []int{8, 564, 384, 82, 301}
	if taskData.GetLeagueId() > 0 {
		leagues = []int{taskData.GetLeagueId()}
	}

	now := time.Now().UTC()
	fmt.Printf("\n=== ДИАГНОСТИКА ЗАКРЫТИЯ ТУРОВ (UTC: %s) ===\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("%-5s | %-16s | %-6s | %-10s | %-10s | %-16s | %-7s | %s\n",
		"ID", "Лига", "Тур", "Начало", "Конец", "Матчи (З/П/И/О)", "Готов?", "Причина")
	fmt.Println(strings.Repeat("-", 108))

	for _, lID := range leagues {
		status, err := analytics.GetRoundClosingStatus(ctx, pool, lID, now)
		if err != nil {
			fmt.Printf("%-5d | %-16s | %-6s | %-10s | %-10s | %-16s | %-7s | Ошибка: %v\n",
				lID, fmt.Sprintf("Лига #%d", lID), "-", "-", "-", "-", "НЕТ", err)
			continue
		}

		readyStr := "❌ НЕТ"
		if status.IsReadyForClosing {
			readyStr = "✅ ДА"
		}

		matchesStat := fmt.Sprintf("%d/%d/%d/%d (%d)",
			status.FinishedMatches,
			status.PostponedMatches,
			status.InPlayMatches,
			status.UpcomingMatches,
			status.TotalMatches,
		)

		fmt.Printf("%-5d | %-16s | %-6s | %-10s | %-10s | %-16s | %-7s | %s\n",
			status.LeagueID,
			status.LeagueName,
			status.RoundName,
			status.StartingAt.Format("2006-01-02"),
			status.EndingAt.Format("2006-01-02"),
			matchesStat,
			readyStr,
			status.Reason,
		)
	}
	fmt.Println(strings.Repeat("-", 108))
	fmt.Println("Пояснение к столбцу матчей: (Завершено / Перенесено / В игре / Предстоит (Всего))")
}

func handleSchedulerTick(ctx context.Context, taskData config.TaskData, log *zap.Logger) {
	pool := db.DBx.PGxPool
	if pool == nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка: база данных не подключена\n")
		os.Exit(1)
	}

	tg := config.Config.TG
	var botToken string
	var channelID int64
	if tg != nil {
		botToken = tg.BotToken
		channelID, _ = strconv.ParseInt(tg.ChannelID, 10, 64)
	}

	storage := publisher.NewPGStorage(pool)
	_ = storage.EnsureSchema(ctx)

	leagues := []int{8, 564, 384, 82, 301}
	if taskData.GetLeagueId() > 0 {
		leagues = []int{taskData.GetLeagueId()}
	}

	provider := scheduler.NewDBTaskProvider(pool, storage, botToken, channelID, leagues)
	now := time.Now().UTC()

	tasks, err := provider.GetDueTasks(ctx, now)
	if err != nil {
		log.Error("failed to get due tasks", zap.Error(err))
		fmt.Fprintf(os.Stderr, "❌ Ошибка получения созревших задач: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== SCHEDULER TICK (UTC: %s) ===\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("Обнаружено созревших задач к публикации: %d\n", len(tasks))

	if len(tasks) == 0 {
		fmt.Println("ℹ️ На текущий момент нет задач, ожидающих публикации (все туры либо в игре, либо уже опубликованы).")
	} else {
		for i, task := range tasks {
			fmt.Printf("[%d/%d] Задача: %s (Лига %d, Раунд %d, Тип: %s)\n",
				i+1, len(tasks), task.Title, task.LeagueID, task.RoundID, task.PublicationType)

			if !taskData.GetPublish() {
				fmt.Printf("      → Режим симуляции (флаг -publish=false). Пропуск отправки.\n")
				continue
			}

			if task.ExecuteFunc != nil {
				fmt.Printf("      → Выполнение публикации...\n")
				if err := task.ExecuteFunc(ctx); err != nil {
					log.Error("failed to execute task", zap.String("title", task.Title), zap.Error(err))
					fmt.Printf("      ❌ Ошибка: %v\n", err)
				} else {
					fmt.Printf("      ✅ Успешно опубликовано!\n")
				}
			}
		}
	}

	// Фоновая актуализация статических данных Mini App (JSON)
	fmt.Printf("\n🔄 Актуализация статических данных Mini App (active_leagues.json, туры, лидеры)...\n")
	var syncTd config.TaskData
	syncTd.Init()
	syncTd.Args = config.LineArgs{
		TaskName: "PublishToTelegram",
		Type:     "syncAll",
		Publish:  false,
	}
	if syncErr := publisher.Dispatch(ctx, syncTd); syncErr != nil {
		log.Warn("scheduler tick viewer sync failed", zap.Error(syncErr))
	}
}

func handleSchedulerDaemon(ctx context.Context, taskData config.TaskData, log *zap.Logger) {
	pool := db.DBx.PGxPool
	if pool == nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка: база данных не подключена\n")
		os.Exit(1)
	}

	tg := config.Config.TG
	var botToken string
	var channelID int64
	if tg != nil {
		botToken = tg.BotToken
		channelID, _ = strconv.ParseInt(tg.ChannelID, 10, 64)
	}

	storage := publisher.NewPGStorage(pool)
	_ = storage.EnsureSchema(ctx)

	leagues := []int{8, 564, 384, 82, 301}
	if taskData.GetLeagueId() > 0 {
		leagues = []int{taskData.GetLeagueId()}
	}

	provider := scheduler.NewDBTaskProvider(pool, storage, botToken, channelID, leagues)
	pipeline := scheduler.NewPipeline(storage, nil, 3)
	daemon := scheduler.NewDaemon(pipeline, provider, storage, 1*time.Minute)

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("🚀 Планировщик запущен в режиме демона (интервал проверки 1 мин). Для остановки нажмите Ctrl+C...")
	if err := daemon.Run(sigCtx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("daemon terminated with error", zap.Error(err))
		fmt.Fprintf(os.Stderr, "❌ Ошибка демона: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Планировщик штатно остановлен (graceful shutdown).")
}

func handleSettleBets(ctx context.Context, taskData config.TaskData, log *zap.Logger) {
	pool := db.DBx.PGxPool
	if pool == nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка: база данных не подключена\n")
		os.Exit(1)
	}

	predStorage := prediction.NewPGStorage(pool)
	_ = predStorage.EnsureSchema(ctx)

	scoreProvider := prediction.NewDBFixtureScoreProvider(pool)
	worker := prediction.NewSettlementWorker(predStorage, scoreProvider)

	fmt.Println("\n=== АВТОРАСЧЁТ СТАВОК И ПРОГНОЗОВ (SETTLEMENT) ===")
	settledCount, err := worker.ProcessPending(ctx)
	if err != nil {
		log.Error("settlement worker error", zap.Error(err))
		fmt.Fprintf(os.Stderr, "❌ Ошибка при расчёте ставок: %v\n", err)
	} else {
		fmt.Printf("✅ Успешно рассчитано ставок на данном цикле: %d\n", settledCount)
	}

	report, err := predStorage.GetPerformanceReport(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Не удалось получить отчет эффективности: %v\n", err)
		return
	}

	fmt.Println("\n📊 ИТОГОВЫЙ ОТЧЁТ ЭФФЕКТИВНОСТИ КАНАЛА (PERFORMANCE REPORT):")
	fmt.Printf("Всего ставок:      %d (Выиграно: %d, Проиграно: %d, Возврат: %d)\n",
		report.TotalBets, report.WonBets, report.LostBets, report.RefundBets)
	fmt.Printf("WinRate:           %.1f%%\n", report.WinRate)
	fmt.Printf("Средний кэф:       %.2f\n", report.AvgOdds)
	fmt.Printf("Объём ставок:      %.2f u\n", report.TotalStaked)
	fmt.Printf("Чистая прибыль:    %+.2f u\n", report.TotalProfit)
	fmt.Printf("ROI / Yield:       %+.1f%%\n", report.ROI)
	fmt.Printf("Profit Factor:     %.2f\n", report.ProfitFactor)
	fmt.Println("==================================================")
}

func handleApiServer(ctx context.Context, taskData config.TaskData, log *zap.Logger) {
	pool := db.DBx.PGxPool
	if pool == nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка: база данных не подключена\n")
		os.Exit(1)
	}

	port := taskData.GetPort()
	pollInterval := taskData.GetPollInterval()
	staticDir := taskData.GetStaticDir()

	srv := server.NewServer(pool, port, staticDir, pollInterval, log)

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("🚀 Запуск Live REST API сервера на порту :%s (pollInterval: %v, static: %s)\n", port, pollInterval, staticDir)
	if err := srv.Start(sigCtx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("api server terminated with error", zap.Error(err))
		fmt.Fprintf(os.Stderr, "❌ Ошибка сервера: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("REST API сервер штатно остановлен (graceful shutdown).")
}

func handlePressDigest(ctx context.Context, taskData config.TaskData, log *zap.Logger) {
	pool := db.DBx.PGxPool
	if pool == nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка: база данных не подключена\n")
		os.Exit(1)
	}

	publish := taskData.Args.Publish
	limit := 1 // по умолчанию 1 самая топовая статья
	if taskData.Cycle > 0 {
		limit = taskData.Cycle
	}

	fmt.Printf("🚀 Запуск дайджеста европейской прессы (Publish: %v, Limit: %d)...\n", publish, limit)
	if err := press.RunPipeline(ctx, pool, &config.Config, publish, limit); err != nil {
		log.Error("press digest pipeline failed", zap.Error(err))
		fmt.Fprintf(os.Stderr, "❌ Ошибка дайджеста: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✨ Дайджест успешно завершен!")
}

