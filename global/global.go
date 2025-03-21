package global

import (
	mylog "github.com/doraemonkeys/mylog/zap"
	"go.uber.org/zap"
)

var (
	Logger *zap.Logger
)

func Init() {
	initLogger()
}

func initLogger() {
	Logger = mylog.NewBuilder().Build()
	mylog.ReplaceGlobals(Logger)
}
