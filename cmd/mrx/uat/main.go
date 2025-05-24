package main

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"peter-kozarec/equinox/cmd/mrx"
	"peter-kozarec/equinox/cmd/mrx/advisor"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/ctrader"
	"peter-kozarec/equinox/internal/dbg"
	"peter-kozarec/equinox/internal/middleware"
	"syscall"
	"time"
)

func main() {
	logger := dbg.NewDevLogger()
	defer logger.Sync()

	logger.Info("MRX started", zap.String("environment", "uat"), zap.String("version", mrx.Version))
	defer logger.Info("MRX finished")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	router := bus.NewRouter(logger, RouterEventCapacity)
	c, err := ctrader.DialDemo(logger)
	if err != nil {
		logger.Fatal("unable to connect to demo device", zap.Error(err))
	}
	defer logger.Info("connection closed")
	defer c.Close()

	monitor := middleware.NewMonitor(logger, MonitorFlags)
	telemetry := middleware.NewTelemetry(logger)
	strategy := advisor.NewStrategy(logger, router)

	// Initialize middleware
	router.TickHandler = middleware.Chain(monitor.WithTick, telemetry.WithTick)(strategy.OnTick)
	router.BarHandler = middleware.Chain(monitor.WithBar, telemetry.WithBar)(strategy.OnBar)
	router.BalanceHandler = middleware.Chain(monitor.WithBalance, telemetry.WithBalance)(strategy.OnBalance)
	router.EquityHandler = middleware.Chain(monitor.WithEquity, telemetry.WithEquity)(strategy.OnEquity)
	router.PositionOpenedHandler = middleware.Chain(monitor.WithPositionOpened, telemetry.WithPositionOpened)(strategy.OnPositionOpened)
	router.PositionClosedHandler = middleware.Chain(monitor.WithPositionClosed, telemetry.WithPositionClosed)(strategy.OnPositionClosed)
	router.PositionPnLUpdatedHandler = middleware.Chain(monitor.WithPositionPnLUpdated, telemetry.WithPositionPnLUpdated)(strategy.OnPositionPnlUpdated)
	router.OrderHandler = nil

	if err := ctrader.Authenticate(ctx, c, int64(accountId), accessToken, appId, appSecret); err != nil {
		logger.Fatal("unable to authenticate", zap.Error(err))
	}
	if err := ctrader.Subscribe(ctx, c, int64(accountId), "BTCUSD", time.Minute, router); err != nil {
		logger.Fatal("unable to subscribe", zap.Error(err))
	}

	go router.Exec(ctx, time.Millisecond)

	defer router.PrintStatistics()
	defer telemetry.PrintStatistics()

	if err := <-router.Done(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal("something unexpected happened", zap.Error(err))
	}
}
