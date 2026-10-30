package db

import (
	"context"
	"fmt"
	"github.com/Pastir/footballdata/config"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
	"net"
	"time"
)

type (
	PostgreSQL struct {
		// PGxPool active connection to db
		PGxPool *pgxpool.Pool
	}
)

var DBx PostgreSQL

// Init manual init database connection
func (db *PostgreSQL) Init() error {

	var cfg *pgxpool.Config

	connString := fmt.Sprintf("user=%s host=%s dbname=%s password=%s port=%d connect_timeout=%d sslmode=%s",
		config.Config.PG.DbUser,
		config.Config.PG.DbHost,
		config.Config.PG.DbName,
		config.Config.PG.DbPass,
		config.Config.PG.DbPort,
		config.Config.PG.ConnectionTimeout,
		config.Config.PG.SslMode)
	cfg, err := pgxpool.ParseConfig(connString)

	cfg.MaxConns = config.Config.PG.MaxConns
	cfg.MinConns = config.Config.PG.MinConns
	cfg.MaxConnLifetime = config.Config.PG.MaxConnLifetime * time.Hour
	cfg.HealthCheckPeriod = config.Config.PG.HealthCheckPeriod * time.Minute
	cfg.MaxConnIdleTime = config.Config.PG.MaxConnIdleTime * time.Minute
	cfg.ConnConfig.ConnectTimeout = config.Config.PG.ConnectionTimeout * time.Second
	cfg.ConnConfig.DialFunc = (&net.Dialer{
		KeepAlive: cfg.HealthCheckPeriod,
		Timeout:   cfg.ConnConfig.ConnectTimeout,
	}).DialContext

	db.PGxPool, err = pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		return err
	}

	return nil
}

func (db *PostgreSQL) Close() error {
	db.PGxPool.Close()
	return nil
}

func (db *PostgreSQL) Check() error {
	err := db.PGxPool.Ping(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func (db *PostgreSQL) Name() string {
	return "postgresql"
}
