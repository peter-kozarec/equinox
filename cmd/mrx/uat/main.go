package main

import (
	"context"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"peter-kozarec/equinox/cmd/mrx"
	"peter-kozarec/equinox/internal/ctrader"
	"syscall"
	"time"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	logger.Info("MRX started", zap.String("environment", "uat"), zap.String("version", mrx.Version))
	defer logger.Info("MRX finished")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	c, err := ctrader.DialDemo(logger)
	if err != nil {
		logger.Fatal("unable to connect to demo device", zap.Error(err))
	}
	defer logger.Info("connection closed")
	defer c.Close()

	logger.Info("connected")
	c.KeepAlive(time.Second * 30)

	if err := c.AuthorizeApplication(ctx, os.Getenv("CtAppId"), os.Getenv("CtAppSecret")); err != nil {
		logger.Fatal("unable to authorize application", zap.Error(err))
	}
	logger.Info("application authorized")

	accounts, err := c.GetAccountList(ctx, os.Getenv("CtAccessToken"))
	if err != nil {
		logger.Fatal("unable to get account list", zap.Error(err))
	}
	if len(accounts) == 0 {
		logger.Fatal("no accounts found")
	}
	logger.Info("accounts found", zap.Int("accounts", len(accounts)))

	if err := c.AuthorizeAccount(ctx, int64(*accounts[0].CtidTraderAccountId), os.Getenv("CtAccessToken")); err != nil {
		logger.Fatal("unable to authorize account", zap.Error(err))
	}
	logger.Info("account authorized")

	select {
	case <-ctx.Done():
	}
}
