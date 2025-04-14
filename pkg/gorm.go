package pkg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func NewSqliteDB(path string, logger *zap.Logger) *gorm.DB {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		panic("failed to create directory:" + err.Error())
	}
	// Set timeout to 10 seconds
	newDB, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s?_pragma=busy_timeout(10000)", path)), &gorm.Config{
		Logger: DefaultGormLogger(logger),
	})
	if err != nil {
		panic("failed to connect database:" + err.Error())
	}
	// Foreign Key Constraints
	tx := newDB.Exec("PRAGMA foreign_keys = ON")
	if tx.Error != nil {
		panic("failed to set foreign_keys:" + tx.Error.Error())
	}

	db, err := newDB.DB()
	if err != nil {
		panic("failed to connect database:" + err.Error())
	}
	// SQLite by default only allows single writer at the same time.
	db.SetMaxOpenConns(1)
	return newDB
}

type GormLogrusLogger struct {
	logger *zap.Logger
	level  gormLogger.LogLevel
}

func (l GormLogrusLogger) Printf(format string, a ...any) {
	switch l.level {
	case gormLogger.Info:
		l.logger.Sugar().Infof(format, a...)
	case gormLogger.Warn:
		l.logger.Sugar().Warnf(format, a...)
	case gormLogger.Error:
		l.logger.Sugar().Errorf(format, a...)
	case gormLogger.Silent:
		// do nothing
	default:
		l.logger.Sugar().Warnf(format, a...)
	}
}

func NewGormLoggerWriter(logger *zap.Logger, level gormLogger.LogLevel) gormLogger.Writer {
	return GormLogrusLogger{
		logger: logger,
		level:  level,
	}
}

func DefaultGormLogger(l *zap.Logger) gormLogger.Interface {
	// var SlowThreshold time.Duration
	// switch l.Level() {
	// case zapcore.DebugLevel:
	// 	SlowThreshold = time.Millisecond * 300
	// case zapcore.InfoLevel:
	// 	SlowThreshold = time.Millisecond * 200
	// default:
	// 	SlowThreshold = time.Millisecond * 200
	// }
	level := gormLogger.Warn
	var newLogger gormLogger.Interface = gormLogger.New(
		NewGormLoggerWriter(l, level),
		gormLogger.Config{
			// SlowThreshold:             SlowThreshold,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		})
	return newLogger
}
