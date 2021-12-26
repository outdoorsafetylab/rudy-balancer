package log

import (
	"fmt"
	"service/config"

	"github.com/crosstalkio/log"
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
		config = zap.NewProductionConfig()
	}
	var err error
	logger, err = config.Build()
	if err != nil {
		return err
	}
	return nil
}

func GetLogger() *zap.Logger {
	return logger
}

func GetSugar() log.Sugar {
	return log.NewSugar(log.NewLogger(write))
}

func write(lv log.Level, payload interface{}) {
	var writer func(string, ...zap.Field)
	logger := logger.WithOptions(zap.AddCallerSkip(3))
	switch lv {
	case log.Debug:
		writer = logger.Debug
	case log.Info:
		writer = logger.Info
	case log.Warning:
		writer = logger.Warn
	case log.Error:
		writer = logger.Error
	case log.Fatal:
		writer = logger.Fatal
	}
	switch v := payload.(type) {
	case string:
		writer(v)
	case []zap.Field:
		writer("", v...)
	default:
		writer(fmt.Sprintf("%v", payload))
	}
}
