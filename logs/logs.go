package logs

import (
	"github.com/Pastir/footballdata/config"
	"github.com/Pastir/logger"
	"go.uber.org/zap"
	"time"
)

type Log map[string]*zap.Logger

var logsType = []string{"access", "info", "error"}
var Logs Log

func (l *Log) Init() error {
	Logs = make(map[string]*zap.Logger)
	for _, val := range logsType {

		typeLog := logger.ConfigLog{
			LogDir:     config.Config.Logs.LogDir,
			Filename:   val + ".log",
			MaxSize:    config.Config.Logs.MaxFileSize,
			MaxBackups: config.Config.Logs.MaxFilesCount,
			MaxAge:     config.Config.Logs.MaxAge,
			Compress:   config.Config.Logs.Compress,
		}

		Logs[val] = typeLog.SetLog()
	}
	return nil
}

func (l *Log) Close() error { return nil }
func (l *Log) Check() error { return nil }

func (l *Log) Name() string {
	return "logger"
}

func ErrorLog(taskName string, nameFunc string, err error) {
	Logs["error"].Error(nameFunc,
		zap.String("ts", time.Now().String()),
		zap.String("taskName", taskName),
		zap.String("nameFunc", nameFunc),
		zap.Error(err),
	)
}

func PanicLog(taskName string, nameFunc string, err error) {
	Logs["error"].Error(nameFunc,
		zap.String("ts", time.Now().String()),
		zap.String("taskName", taskName),
		zap.String("nameFunc", nameFunc),
		zap.Error(err),
	)
}

func AccessLog(taskName string, nameFunc string, err error) {
	Logs["access"].Info(nameFunc,
		zap.String("ts", time.Now().String()),
		zap.String("taskName", taskName),
		zap.String("nameFunc", nameFunc),
		zap.Error(err),
	)
}

func InfoLog(taskName string, nameFunc string, msg string) {
	Logs["access"].Info(nameFunc,
		zap.String("ts", time.Now().String()),
		zap.String("taskName", taskName),
		zap.String("nameFunc", nameFunc),
		zap.String("info", msg),
	)
}
