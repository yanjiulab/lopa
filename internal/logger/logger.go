package logger

import (
	"go.uber.org/zap"
)

var (
	log  *zap.Logger
	slog *zap.SugaredLogger
)

// Init creates the global zap logger.
func Init() error {
	l, err := zap.NewProduction()
	if err != nil {
		return err
	}
	log = l
	slog = l.Sugar()
	return nil
}

// L returns the structured zap logger.
func L() *zap.Logger {
	if log == nil {
		_ = Init()
	}
	return log
}

// S returns the sugared zap logger.
func S() *zap.SugaredLogger {
	if slog == nil {
		_ = Init()
	}
	return slog
}

