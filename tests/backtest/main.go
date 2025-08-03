package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/peter-kozarec/equinox/examples/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/datasource"
	"github.com/peter-kozarec/equinox/pkg/datasource/historical"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/exchange/sandbox"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/tools/bar"
	"github.com/peter-kozarec/equinox/pkg/tools/metrics"
	"github.com/peter-kozarec/equinox/pkg/tools/risk"
	"github.com/peter-kozarec/equinox/pkg/tools/risk/stoploss"
	"github.com/peter-kozarec/equinox/pkg/tools/risk/takeprofit"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	TickBinDir = "/home/pko/Workspace/market_data/"

	StartTime = "2018-01-01 00:00:00"
	EndTime   = "2025-01-01 00:00:00"

	meanReversionWindow = 60
)

var (
	barPeriod       = common.BarPeriodM10
	accountCurrency = "USD"
	startBalance    = fixed.FromInt(10000, 0)
	slippage        = fixed.FromFloat64(0.00002)

	routerCapacity = 1000

	symbolInfo = exchange.SymbolInfo{
		SymbolName:    "EURUSD",
		QuoteCurrency: "USD",
		Digits:        5,
		PipSize:       fixed.FromFloat64(0.0001),
		ContractSize:  fixed.FromFloat64(100_000),
		Leverage:      fixed.FromFloat64(30),
	}

	riskConf = risk.Configuration{
		RiskMax:  fixed.FromFloat64(0.3),
		RiskMin:  fixed.FromFloat64(0.1),
		RiskBase: fixed.FromFloat64(0.2),
		RiskOpen: fixed.Ten,
	}

	stopLossAtrWindow     = 1
	stopLossAtrMultiplier = fixed.FromInt(2, 0)
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	src := historical.NewSource[historical.BinaryTick](TickBinDir + "eurusd.bin")
	if err := src.Open(); err != nil {
		slog.Error("unable to open mapper", "error", err)
	}
	defer src.Close()

	startTime, _ := time.Parse(time.DateTime, StartTime)
	endTime, _ := time.Parse(time.DateTime, EndTime)

	router := bus.NewRouter(routerCapacity)

	simulator, err := sandbox.NewSimulator(router, accountCurrency, startBalance,
		sandbox.WithSlippageHandler(func(_ common.Position) fixed.Point { return slippage }),
		sandbox.WithSymbols(symbolInfo))
	if err != nil {
		slog.Error("unable to create simulator", "error", err)
		os.Exit(1)
	}

	tickReader := historical.NewTickReader(src, symbolInfo.SymbolName, startTime, endTime)
	barBuilder := bar.NewBuilder(router, bar.With(symbolInfo.SymbolName, barPeriod, bar.PriceModeBid))

	flags := middleware.MonitorPositionClose
	monitor := middleware.NewMonitor(flags)
	perf := middleware.NewPerformance()

	audit := metrics.NewAudit()
	reversionStrategy := strategy.NewMeanReversion(router, meanReversionWindow)

	sl := stoploss.NewAtrBasedStopLoss(stopLossAtrWindow, stopLossAtrMultiplier)
	tp := takeprofit.NewFixedTakeProfit()

	riskOptions := []risk.Option{risk.WithDefaultKellyMultiplier(), risk.WithDefaultDrawdownMultiplier(), risk.WithDefaultRRRMultiplier(), risk.WithOnHourCooldown()}
	riskManager := risk.NewManager(router, symbolInfo, riskConf, sl, tp, riskOptions...)

	riskManager.SetMaxEquity(startBalance)
	riskManager.SetEquity(startBalance)
	riskManager.SetMaxBalance(startBalance)
	riskManager.SetBalance(startBalance)

	router.OnTick = middleware.Chain(monitor.WithTick, perf.WithTick)(bus.MergeHandlers(simulator.OnTick, riskManager.OnTick, barBuilder.OnTick, reversionStrategy.OnTick))
	router.OnBar = middleware.Chain(monitor.WithBar, perf.WithBar)(bus.MergeHandlers(sl.OnBar, riskManager.OnBar, reversionStrategy.OnBar))
	router.OnOrder = middleware.Chain(monitor.WithOrder, perf.WithOrder)(simulator.OnOrder)
	router.OnOrderAcceptance = middleware.Chain(monitor.WithOrderAcceptance, perf.WithOrderAcceptance)(riskManager.OnOrderAccepted)
	router.OnOrderRejection = middleware.Chain(monitor.WithOrderRejection, perf.WithOrderRejection)(riskManager.OnOrderRejected)
	router.OnOrderFilled = middleware.Chain(monitor.WithOrderFilled, perf.WithOrderFilled)(middleware.NoopOrderFilledHandler)
	router.OnOrderCancel = middleware.Chain(monitor.WithOrderCancelled, perf.WithOrderCancelled)(middleware.NoopOrderCancelledHandler)
	router.OnPositionOpen = middleware.Chain(monitor.WithPositionOpen, perf.WithPositionOpen)(riskManager.OnPositionOpened)
	router.OnPositionClose = middleware.Chain(monitor.WithPositionClose, perf.WithPositionClose)(bus.MergeHandlers(riskManager.OnPositionClosed, audit.OnPositionClosed))
	router.OnPositionUpdate = middleware.Chain(monitor.WithPositionUpdate, perf.WithPositionUpdate)(riskManager.OnPositionUpdated)
	router.OnEquity = middleware.Chain(monitor.WithEquity, perf.WithEquity)(bus.MergeHandlers(riskManager.OnEquity, audit.OnEquity))
	router.OnBalance = middleware.Chain(monitor.WithBalance, perf.WithBalance)(riskManager.OnBalance)
	router.OnSignal = middleware.Chain(monitor.WithSignal, perf.WithSignal)(riskManager.OnSignal)
	router.OnSignalAcceptance = middleware.Chain(monitor.WithSignalAcceptance, perf.WithSignalAcceptance)(middleware.NoopSignalAcceptanceHandler)
	router.OnSignalRejection = middleware.Chain(monitor.WithSignalRejection, perf.WithSignalRejection)(middleware.NoopSignalRejectionHandler)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	if e := <-router.ExecLoop(ctx, datasource.CreateTickDispatcher(router, tickReader)); e != nil {
		if !errors.Is(e, context.Canceled) && !errors.Is(e, historical.ErrEof) {
			slog.Error("unexpected error during execution", "error", e)
			os.Exit(1)
		}
	}

	simulator.CloseAllOpenPositions()

	perf.PrintStatistics()
	router.GetStatistics().Print()
	audit.GenerateReport().Print()
}
