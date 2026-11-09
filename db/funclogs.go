package db

import (
	"errors"
	"time"

	"github.com/Pastir/footballdata/logs"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx"
	"go.uber.org/zap"
)

func PGxPanic(taskName string, nameFunc string, err error) {
	detail := ""
	var pgErr *pgx.PgError
	if errors.As(err, &pgErr) && pgErr != nil {
		detail = pgErr.Detail
	}
	logs.Logs["error"].Panic("panic db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.String("detail", detail),
		zap.Error(err),
	)
}

func PGxScanPanic(taskName string, nameFunc string, err error) {
	logs.Logs["error"].Panic("panic db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.Error(err),
	)
}

func PGxConnPanic(taskName string, nameFunc string, err error) {
	logs.Logs["error"].Panic("panic db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.Error(err),
	)
}

func PGxError(taskName string, nameFunc string, err error) {
	msg, detail := "", ""
	var pgErr *pgx.PgError
	if errors.As(err, &pgErr) && pgErr != nil {
		msg = pgErr.Message
		detail = pgErr.Detail
	}
	logs.Logs["error"].Error("error db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.String("message", msg),
		zap.String("detail", detail),
		zap.Error(err),
	)
}

func PGxConnError(taskName string, nameFunc string, err error) {
	msg, detail := "", ""
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr != nil {
		msg = pgErr.Message
		detail = pgErr.Detail
	} else if err != nil {
		msg = err.Error()
	}
	logs.Logs["error"].Error("error db:",
		zap.String("ts", time.Now().String()),
		zap.String("funcName", nameFunc),
		zap.String("taskName", taskName),
		zap.String("message", msg),
		zap.String("detail", detail),
		zap.Error(err),
	)
}
