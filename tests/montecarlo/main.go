package main

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/peter-kozarec/equinox/examples/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/datasource"
	"github.com/peter-kozarec/equinox/pkg/datasource/synthetic"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/exchange/sandbox"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/tools/bar"
	"github.com/peter-kozarec/equinox/pkg/tools/metrics"
	"github.com/peter-kozarec/equinox/pkg/tools/risk"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	barPeriod       = common.BarPeriodM1
	accountCurrency = "USD"
	startBalance    = fixed.FromInt(10000, 0)
	slippage        = fixed.FromFloat64(0.00002)

	symbolInfo = exchange.SymbolInfo{
		SymbolName:    "EURUSD",
		QuoteCurrency: "USD",
		Class:         exchange.Forex,
		Digits:        5,
		PipSize:       fixed.FromFloat64(0.0001),
		ContractSize:  fixed.FromFloat64(100_000),
		Leverage:      fixed.FromFloat64(30),
	}

	meanReversionWindow = 60

	genRng      = rand.New(rand.NewSource(time.Now().UnixNano()))
	genDuration = 30 * 24 * time.Hour
	genMu       = 0.1607143264
	genSigma    = 0.0698081590

	routerCapacity = 1000

	riskConf = risk.Configuration{
		MaxRiskRate:  fixed.FromFloat64(0.3),
		MinRiskRate:  fixed.FromFloat64(0.1),
		BaseRiskRate: fixed.FromFloat64(0.2),
		OpenRiskRate: fixed.Ten,
		SizeDigits:   2,
	}

	stopLossAtrWindow     = 10
	stopLossAtrMultiplier = fixed.FromInt(4, 0)
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	router := bus.NewRouter(routerCapacity)
	simulator, err := sandbox.NewSimulator(router, accountCurrency, startBalance,
		sandbox.WithSlippageHandler(func(_ common.Position) fixed.Point { return slippage }),
		sandbox.WithSymbols(symbolInfo),
		sandbox.WithMaintenanceMargin(fixed.FromFloat64(20)))
	if err != nil {
		slog.Error("unable to create simulator", "error", err)
		os.Exit(1)
	}

	builder := bar.NewBuilder(router, bar.With(symbolInfo.SymbolName, barPeriod, bar.PriceModeBid))
	generator := synthetic.NewEURUSDTickGenerator(symbolInfo.SymbolName, genRng, genDuration, genMu, genSigma)

	monitor := middleware.NewMonitor(middleware.MonitorPositionClose)
	perf := middleware.NewPerformance()

	audit := metrics.NewAudit()
	reversionStrategy := strategy.NewMeanReversion(router, meanReversionWindow)

	sl := risk.NewAtrBasedStopLoss(stopLossAtrWindow, stopLossAtrMultiplier)
	tp := risk.NewFixedTakeProfit()

	riskManager, err := risk.NewManager(router, riskConf, sl, tp, risk.WithSymbols(symbolInfo))
	if err != nil {
		slog.Error("unable to create risk manager", "error", err)
		os.Exit(1)
	}

	router.OnTick = middleware.Chain(monitor.WithTick, perf.WithTick)(bus.MergeHandlers(simulator.OnTick, riskManager.OnTick, builder.OnTick, reversionStrategy.OnTick))
	router.OnBar = middleware.Chain(monitor.WithBar, perf.WithBar)(bus.MergeHandlers(sl.OnBar, reversionStrategy.OnBar))
	router.OnOrder = middleware.Chain(monitor.WithOrder, perf.WithOrder)(simulator.OnOrder)
	router.OnOrderAcceptance = middleware.Chain(monitor.WithOrderAcceptance, perf.WithOrderAcceptance)(middleware.NoopOrderAcceptanceHandler)
	router.OnOrderRejection = middleware.Chain(monitor.WithOrderRejection, perf.WithOrderRejection)(riskManager.OnOrderRejected)
	router.OnOrderFilled = middleware.Chain(monitor.WithOrderFilled, perf.WithOrderFilled)(riskManager.OnOrderFilled)
	router.OnOrderCancel = middleware.Chain(monitor.WithOrderCancelled, perf.WithOrderCancelled)(middleware.NoopOrderCancelledHandler)
	router.OnPositionOpen = middleware.Chain(monitor.WithPositionOpen, perf.WithPositionOpen)(riskManager.OnPositionOpen)
	router.OnPositionClose = middleware.Chain(monitor.WithPositionClose, perf.WithPositionClose)(bus.MergeHandlers(riskManager.OnPositionClose, audit.OnPositionClosed))
	router.OnPositionUpdate = middleware.Chain(monitor.WithPositionUpdate, perf.WithPositionUpdate)(riskManager.OnPositionUpdate)
	router.OnEquity = middleware.Chain(monitor.WithEquity, perf.WithEquity)(bus.MergeHandlers(riskManager.OnEquity, audit.OnEquity))
	router.OnBalance = middleware.Chain(monitor.WithBalance, perf.WithBalance)(riskManager.OnBalance)
	router.OnSignal = middleware.Chain(monitor.WithSignal, perf.WithSignal)(riskManager.OnSignal)
	router.OnSignalAcceptance = middleware.Chain(monitor.WithSignalAcceptance, perf.WithSignalAcceptance)(middleware.NoopSignalAcceptanceHandler)
	router.OnSignalRejection = middleware.Chain(monitor.WithSignalRejection, perf.WithSignalRejection)(middleware.NoopSignalRejectionHandler)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	if err := <-router.ExecLoop(ctx, datasource.CreateTickDispatcher(router, generator)); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, synthetic.ErrEof) {
			slog.Error("unexpected error during execution", "error", err)
			os.Exit(1)
		}
	}

	simulator.CloseAllOpenPositions()

	perf.PrintStatistics()
	router.GetStatistics().Print()
	audit.GenerateReport().Print()
}
