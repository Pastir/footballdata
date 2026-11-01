package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

// Server реализует легковесный HTTP REST API сервер для отдачи live и архивных данных.
type Server struct {
	httpServer *http.Server
	pool       *pgxpool.Pool
	store      *LiveStore
	poller     *LivePoller
	port       string
	staticDir  string
	log        *zap.Logger
}

// NewServer создает новый экземпляр HTTP сервера и инициализирует LiveStore и LivePoller.
func NewServer(pool *pgxpool.Pool, port string, staticDir string, pollInterval time.Duration, log *zap.Logger) *Server {
	if port == "" {
		port = "8080"
	}
	if log == nil {
		l, _ := zap.NewDevelopment()
		log = l
	}

	store := NewLiveStore()
	poller := NewLivePoller(pool, store, pollInterval, []int{8, 564, 384, 82, 301}, log)

	s := &Server{
		pool:      pool,
		store:     store,
		poller:    poller,
		port:      port,
		staticDir: staticDir,
		log:       log,
	}

	mux := http.NewServeMux()
	handlers := NewServerHandlers(pool, store, poller, staticDir, log)

	// API Маршруты
	mux.HandleFunc("/health", withCORS(handlers.HandleHealth))
	mux.HandleFunc("/api/health", withCORS(handlers.HandleHealth))
	mux.HandleFunc("/api/leagues", withCORS(handlers.HandleLeagues))
	mux.HandleFunc("/api/live", withCORS(handlers.HandleLiveMatches))
	mux.HandleFunc("/api/leaders", withCORS(handlers.HandleLeaders))

	// Параметризованные маршруты
	mux.HandleFunc("/api/leagues/", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/archive") {
			handlers.HandleLeagueArchive(w, r)
			return
		}
		handlers.HandleLeagues(w, r)
	}))

	mux.HandleFunc("/api/rounds/", withCORS(handlers.HandleRoundOnDemand))

	mux.HandleFunc("/api/fixtures/", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/live") {
			handlers.HandleFixtureLive(w, r)
			return
		}
		http.NotFound(w, r)
	}))

	// Статические файлы (Telegram Mini App)
	if staticDir != "" {
		if _, err := os.Stat(staticDir); err == nil {
			fs := http.FileServer(http.Dir(staticDir))
			mux.Handle("/", fs)
			log.Info("serving static assets", zap.String("dir", staticDir))
		}
	}

	s.httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start запускает REST API сервер и фоновый опросчик live-матчей.
func (s *Server) Start(ctx context.Context) error {
	s.log.Info("starting REST API server",
		zap.String("addr", s.httpServer.Addr),
		zap.Duration("poll_interval", s.poller.PollInterval()),
		zap.String("static_dir", s.staticDir),
	)

	// Запуск фонового in-play поллера
	go func() {
		if err := s.poller.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.log.Error("live poller worker error", zap.Error(err))
		}
	}()

	errChan := make(chan error, 1)
	go func() {
		fmt.Printf("🚀 REST API сервер запущен на порту :%s (pollInterval: %v)\n", s.port, s.poller.PollInterval())
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		s.log.Info("shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)

	case err := <-errChan:
		return fmt.Errorf("http server failed: %w", err)
	}
}
