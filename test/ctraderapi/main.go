package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"

	"github.com/peter-kozarec/equinox/examples/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/ctrader"
	"github.com/peter-kozarec/equinox/pkg/middleware"

	"syscall"
	"time"
)

var appId = os.Getenv("CtAppId")
var appSecret = os.Getenv("CtAppSecret")
var accountId, _ = strconv.Atoi(os.Getenv("CtAccountId"))
var accessToken = os.Getenv("CtAccessToken")

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	router := bus.NewRouter(1000)
	c, err := ctrader.DialDemo()
	if err != nil {
		slog.Error("unable to connect to demo device", "error", err)
		os.Exit(1)
	}
	defer slog.Info("connection closed")
	defer c.Close()

	monitor := middleware.NewMonitor(middleware.MonitorOrders | middleware.MonitorPositionsClosed | middleware.MonitorPositionsOpened | middleware.MonitorBars)
	telemetry := middleware.NewTelemetry()
	advisor := strategy.NewMrxAdvisor(router)

	if err := ctrader.Authenticate(ctx, c, int64(accountId), accessToken, appId, appSecret); err != nil {
		slog.Error("unable to authenticate", "error", err)
		os.Exit(1)
	}
	orderHandler, err := ctrader.InitTradeSession(ctx, c, int64(accountId), "BTCUSD", time.Minute, router)
	if err != nil {
		slog.Error("unable to initialize trading session", "error", err)
		os.Exit(1)
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
		slog.Error("something unexpected happened", "error", err)
		os.Exit(1)
	}
}
