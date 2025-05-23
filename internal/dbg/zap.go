package dbg

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewDevLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()

	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.DisableCaller = true

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

func NewProdLogger() *zap.Logger {
	cfg := zap.NewProductionConfig()

	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.DisableCaller = true

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return logger
}
