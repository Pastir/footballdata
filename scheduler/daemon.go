package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/Pastir/footballdata/publisher"
)

// TaskProvider возвращает список задач, созревших для выполнения на текущем тике.
type TaskProvider interface {
	GetDueTasks(ctx context.Context, now time.Time) ([]PublicationTask, error)
}

// Daemon фоновый процесс, выполняющий периодическую проверку и выпуск публикаций.
type Daemon struct {
	pipeline     *Pipeline
	taskProvider TaskProvider
	storage      publisher.Storage
	interval     time.Duration
}

// NewDaemon создаёт новый экземпляр демона.
func NewDaemon(
	pipeline *Pipeline,
	taskProvider TaskProvider,
	storage publisher.Storage,
	interval time.Duration,
) *Daemon {
	if interval <= 0 {
		interval = 1 * time.Minute
	}
	return &Daemon{
		pipeline:     pipeline,
		taskProvider: taskProvider,
		storage:      storage,
		interval:     interval,
	}
}

// Run запускает бесконечный цикл планировщика до отмены контекста (graceful shutdown).
func (d *Daemon) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	// Выполняем первую проверку сразу при старте
	if err := d.Tick(ctx); err != nil {
		// Логируем, но продолжаем работу
		fmt.Printf("[daemon] initial tick warning: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ticker.C:
			if err := d.processDueTasks(ctx, t); err != nil {
				fmt.Printf("[daemon] tick error: %v\n", err)
			}
		}
	}
}

// Tick выполняет одну итерацию проверки и публикации.
func (d *Daemon) Tick(ctx context.Context) error {
	return d.processDueTasks(ctx, time.Now())
}

func (d *Daemon) processDueTasks(ctx context.Context, now time.Time) error {
	if d.taskProvider == nil {
		return nil
	}

	tasks, err := d.taskProvider.GetDueTasks(ctx, now)
	if err != nil {
		return fmt.Errorf("get due tasks: %w", err)
	}

	for _, task := range tasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, err = d.pipeline.Execute(ctx, task)
		if err != nil {
			fmt.Printf("[daemon] failed to execute task %s: %v\n", task.Title, err)
			continue
		}
	}

	return nil
}
