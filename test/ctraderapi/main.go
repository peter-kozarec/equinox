package main

import (
	"context"
	"errors"
	"github.com/peter-kozarec/equinox/internal/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/ctrader"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"strconv"

	"syscall"
	"time"
)

var appId = os.Getenv("CtAppId")
var appSecret = os.Getenv("CtAppSecret")
var accountId, _ = strconv.Atoi(os.Getenv("CtAccountId"))
var accessToken = os.Getenv("CtAccessToken")

func main() {
	// For development - pretty console output
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	router := bus.NewRouter(logger, 1000)
	c, err := ctrader.DialDemo(logger)
	if err != nil {
		logger.Fatal("unable to connect to demo device", zap.Error(err))
	}
	defer logger.Info("connection closed")
	defer c.Close()

	monitor := middleware.NewMonitor(logger, middleware.MonitorOrders|middleware.MonitorPositionsClosed|middleware.MonitorPositionsOpened|middleware.MonitorBars)
	telemetry := middleware.NewTelemetry(logger)
	advisor := strategy.NewAdvisor(logger, router)

	if err := ctrader.Authenticate(ctx, c, int64(accountId), accessToken, appId, appSecret); err != nil {
		logger.Fatal("unable to authenticate", zap.Error(err))
	}
	orderHandler, err := ctrader.InitTradeSession(ctx, c, int64(accountId), "EURUSD", time.Minute, router)
	if err != nil {
		logger.Fatal("unable to initialize trading session", zap.Error(err))
	}

	// Initialize middleware
	router.TickHandler = middleware.Chain(monitor.WithTick, telemetry.WithTick)(advisor.NewTick)
	router.BarHandler = middleware.Chain(monitor.WithBar, telemetry.WithBar)(advisor.NewBar)
	router.BalanceHandler = middleware.Chain(monitor.WithBalance, telemetry.WithBalance)(middleware.NoopBalanceHdl)
	router.EquityHandler = middleware.Chain(monitor.WithEquity, telemetry.WithEquity)(middleware.NoopEquityHdl)
	router.PositionOpenedHandler = middleware.Chain(monitor.WithPositionOpened, telemetry.WithPositionOpened)(middleware.NoopPosOpnHdl)
	router.PositionClosedHandler = middleware.Chain(monitor.WithPositionClosed, telemetry.WithPositionClosed)(advisor.PositionClosed)
	router.PositionPnLUpdatedHandler = middleware.Chain(monitor.WithPositionPnLUpdated, telemetry.WithPositionPnLUpdated)(middleware.NoopPosUpdHdl)
	router.OrderHandler = middleware.Chain(monitor.WithOrder, telemetry.WithOrder)(orderHandler)

	go router.Exec(ctx)

	defer router.PrintStatistics()
	defer telemetry.PrintStatistics()

	if err := <-router.Done(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal("something unexpected happened", zap.Error(err))
	}
}
