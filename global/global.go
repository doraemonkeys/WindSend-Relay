package global

import (
	"log"

	mylog "github.com/doraemonkeys/mylog/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger *zap.Logger
)

func InitLogger(level string) {
	l, err := zapcore.ParseLevel(level)
	if err != nil {
		log.Printf("invalid log level: %s, use INFO instead", level)
		l = zapcore.InfoLevel
	}
	Logger = mylog.NewBuilder().Level(l).Build()
	mylog.ReplaceGlobals(Logger)
}
