package main

import (
	"context"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"peter-kozarec/equinox/cmd/mrx"
	"syscall"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	logger.Info("MRX started", zap.String("environment", "live"), zap.String("version", mrx.Version))
	defer logger.Info("MRX finished")

	_, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()
}
