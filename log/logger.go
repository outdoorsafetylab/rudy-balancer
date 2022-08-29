package log

import (
	"fmt"

	"service/config"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func Init() error {
	cfg := config.Get()
	var config zap.Config
	if cfg.GetBool("log.dev") {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zapdriver.NewProductionConfig()
	}
	config.DisableCaller = true
	config.DisableStacktrace = true
	config.Level.SetLevel(zap.DebugLevel)
	var err error
	logger, err = config.Build()
	if err != nil {
		return err
	}
	return nil
}

func Logger() *zap.Logger {
	return logger
}

type Level int

const (
	Debug Level = iota
	Info
	Warning
	Error
	Fatal
)

func Debugf(fmtstr string, args ...interface{}) {
	Write(Debug, fmt.Sprintf(fmtstr, args...))
}

func Infof(fmtstr string, args ...interface{}) {
	Write(Info, fmt.Sprintf(fmtstr, args...))
}

func Warningf(fmtstr string, args ...interface{}) {
	Write(Warning, fmt.Sprintf(fmtstr, args...))
}

func Warnf(fmtstr string, args ...interface{}) {
	Write(Warning, fmt.Sprintf(fmtstr, args...))
}

func Errorf(fmtstr string, args ...interface{}) {
	Write(Error, fmt.Sprintf(fmtstr, args...))
}

func Write(lv Level, msg string, fields ...zap.Field) {
	var writer func(string, ...zap.Field)
	// logger := logger.WithOptions(zap.AddCallerSkip(1))
	switch lv {
	case Debug:
		writer = logger.Debug
	case Info:
		writer = logger.Info
	case Warning:
		writer = logger.Warn
	case Error:
		writer = logger.Error
	case Fatal:
		writer = logger.Fatal
	}
	writer(msg, fields...)
}
