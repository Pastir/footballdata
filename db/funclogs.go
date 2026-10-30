package db

import (
	"github.com/Pastir/footballdata/logs"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx"
	pgx2 "github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"time"
)

func PGxPanic(taskName string, nameFunc string, err *pgx.PgError) {
	logs.Logs["error"].Panic("panic db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.String("detail", err.Detail),
		zap.Error(err),
	)
}

func PGxScanPanic(taskName string, nameFunc string, err pgx2.ScanArgError) {
	logs.Logs["error"].Panic("panic db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.Error(err),
	)
}

func PGxConnPanic(taskName string, nameFunc string, err *pgconn.PgError) {
	logs.Logs["error"].Panic("panic db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.Error(err),
	)
}

func PGxError(taskName string, nameFunc string, err *pgx.PgError) {
	logs.Logs["error"].Error("error db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.String("message", err.Message),
		zap.String("detail", err.Detail),
		zap.Error(err),
	)
}

func PGxConnError(taskName string, nameFunc string, err *pgconn.PgError) {
	logs.Logs["error"].Error("error db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.String("message", err.Message),
		zap.String("detail", err.Detail),
		zap.Error(err),
	)
}
