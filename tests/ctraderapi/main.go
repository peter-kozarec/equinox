package main

import (
	"context"
	"errors"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/tools/bar"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"os"
	"os/signal"
	"strconv"

	"github.com/peter-kozarec/equinox/examples/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/ctrader"
	"github.com/peter-kozarec/equinox/pkg/middleware"

	"syscall"
)

var appId = os.Getenv("CtAppId")
var appSecret = os.Getenv("CtAppSecret")
var accountId, _ = strconv.Atoi(os.Getenv("CtAccountId"))
var accessToken = os.Getenv("CtAccessToken")

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

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

	monitor := middleware.NewMonitor(middleware.MonitorAll)
	advisor := strategy.NewMrxAdvisor(router)
	barBuilder := bar.NewBuilder(router, bar.BuildModeTickBased, bar.PriceModeBid, false)
	barBuilder.Build("BTCUSD", common.BarPeriodM1)

	if err := ctrader.Authenticate(ctx, c, int64(accountId), accessToken, appId, appSecret); err != nil {
		slog.Error("unable to authenticate", "error", err)
		os.Exit(1)
	}
	orderHandler, err := ctrader.InitTradeSession(ctx, c, int64(accountId), "BTCUSD", router)
	if err != nil {
		slog.Error("unable to initialize trading session", "error", err)
		os.Exit(1)
	}

	// Initialize middleware
	router.TickHandler = middleware.Chain(monitor.WithTick)(func(ctx context.Context, tick common.Tick) {
		barBuilder.OnTick(ctx, tick)
		advisor.OnTick(ctx, tick)
	})
	router.BarHandler = middleware.Chain(monitor.WithBar)(advisor.OnBar)
	router.BalanceHandler = middleware.Chain(monitor.WithBalance)(middleware.NoopBalanceHdl)
	router.EquityHandler = middleware.Chain(monitor.WithEquity)(middleware.NoopEquityHdl)
	router.PositionOpenedHandler = middleware.Chain(monitor.WithPositionOpened)(middleware.NoopPosOpnHdl)
	router.PositionClosedHandler = middleware.Chain(monitor.WithPositionClosed)(middleware.NoopPosClsHdl)
	router.PositionPnLUpdatedHandler = middleware.Chain(monitor.WithPositionPnLUpdated)(middleware.NoopPosUpdHdl)
	router.OrderHandler = middleware.Chain(monitor.WithOrder)(orderHandler)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		errCh := router.Exec(ctx)
		select {
		case e := <-errCh:
			return e
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	g.Go(func() error {
		errCh := barBuilder.Exec(ctx)
		select {
		case e := <-errCh:
			return e
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	defer router.PrintStatistics()

	if e := g.Wait(); e != nil && !errors.Is(e, context.Canceled) {
		slog.Error("something unexpected happened", "error", e)
		os.Exit(1)
	}
}
