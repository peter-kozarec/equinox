package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/peter-kozarec/equinox/examples/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange/ctrader"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/tools/bar"
)

const (
	symbol    = "BTCUSD"
	barPeriod = common.BarPeriodM1
	barMode   = bar.PriceModeBid

	routerCapacity = 1000

	meanReversionWindow = 60
)

var appId = os.Getenv("CtAppId")
var appSecret = os.Getenv("CtAppSecret")
var accountId, _ = strconv.Atoi(os.Getenv("CtAccountId"))
var accessToken = os.Getenv("CtAccessToken")

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	router := bus.NewRouter(routerCapacity)
	c, err := ctrader.DialDemo()
	if err != nil {
		slog.Error("unable to connect to demo device", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	monitor := middleware.NewMonitor(middleware.MonitorAll)
	advisor := strategy.NewMeanReversion(router, meanReversionWindow)
	barBuilder := bar.NewBuilder(router, bar.With(symbol, barPeriod, barMode))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	if err := ctrader.Authenticate(ctx, c, int64(accountId), accessToken, appId, appSecret); err != nil {
		slog.Error("unable to authenticate", "error", err)
		os.Exit(1)
	}
	orderHandler, err := ctrader.InitTradeSession(ctx, c, int64(accountId), symbol, router)
	if err != nil {
		slog.Error("unable to initialize trading session", "error", err)
		os.Exit(1)
	}

	router.OnTick = middleware.Chain(monitor.WithTick)(bus.MergeHandlers(barBuilder.OnTick, advisor.OnTick))
	router.OnBar = middleware.Chain(monitor.WithBar)(advisor.OnBar)
	router.OnBalance = middleware.Chain(monitor.WithBalance)(middleware.NoopBalanceHandler)
	router.OnEquity = middleware.Chain(monitor.WithEquity)(middleware.NoopEquityHandler)
	router.OnPositionOpen = middleware.Chain(monitor.WithPositionOpen)(middleware.NoopPositionUpdateHandler)
	router.OnPositionClose = middleware.Chain(monitor.WithPositionClose)(middleware.NoopPositionUpdateHandler)
	router.OnPositionUpdate = middleware.Chain(monitor.WithPositionUpdate)(middleware.NoopPositionUpdateHandler)
	router.OnOrder = middleware.Chain(monitor.WithOrder)(orderHandler)

	if e := <-router.Exec(ctx); e != nil && !errors.Is(e, context.Canceled) {
		slog.Error("something unexpected happened", "error", e)
		os.Exit(1)
	}

	router.GetStatistics().Print()
}
