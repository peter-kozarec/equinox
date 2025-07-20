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
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/tools/bar"
	"github.com/peter-kozarec/equinox/pkg/tools/metrics"
	"github.com/peter-kozarec/equinox/pkg/tools/risk"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	symbol       = "EURUSD"
	barPeriod    = common.BarPeriodM1
	startBalance = fixed.FromInt(10000, 0)

	meanReversionWindow = 60

	genRng      = rand.New(rand.NewSource(time.Now().UnixNano()))
	genDuration = 30 * 24 * time.Hour
	genMu       = 0.1607143264
	genSigma    = 0.0698081590

	routerCapacity = 1000

	instrument = common.Instrument{
		Symbol:           symbol,
		Digits:           5,
		PipSize:          fixed.FromInt(1, 4),
		ContractSize:     fixed.FromInt(100000, 0),
		CommissionPerLot: fixed.FromInt(3, 0),
		PipSlippage:      fixed.FromInt(2, 5),
	}

	riskConf = risk.Configuration{
		RiskMax:  fixed.FromFloat64(0.3),
		RiskMin:  fixed.FromFloat64(0.1),
		RiskBase: fixed.FromFloat64(0.2),
		RiskOpen: fixed.Ten,

		AtrPeriod:                  44,
		AtrStopLossMultiplier:      fixed.Five,
		AtrTakeProfitMinMultiplier: fixed.Two,

		BreakEvenMove:      fixed.FromFloat64(20),
		BreakEvenThreshold: fixed.FromFloat64(60),
	}
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	router := bus.NewRouter(routerCapacity)
	simulator := exchange.NewSimulator(router, instrument, startBalance)

	builder := bar.NewBuilder(router, bar.With(symbol, barPeriod, bar.PriceModeBid))
	generator := synthetic.NewEURUSDTickGenerator(symbol, genRng, genDuration, genMu, genSigma)

	flags := middleware.MonitorSignal | middleware.MonitorOrder | middleware.MonitorSignalAcceptance | middleware.MonitorSignalRejection | middleware.MonitorPositionClose
	monitor := middleware.NewMonitor(flags)
	perf := middleware.NewPerformance()

	audit := metrics.NewAudit()
	reversionStrategy := strategy.NewMeanReversion(router, meanReversionWindow)

	riskOptions := []risk.Option{risk.WithDefaultKellyMultiplier(), risk.WithDefaultDrawdownMultiplier(), risk.WithDefaultRRRMultiplier(), risk.WithOnHourCooldown()}
	riskManager := risk.NewManager(router, instrument, riskConf, riskOptions...)

	router.OnTick = middleware.Chain(monitor.WithTick, perf.WithTick)(bus.MergeHandlers(simulator.OnTick, riskManager.OnTick, builder.OnTick, reversionStrategy.OnTick))
	router.OnBar = middleware.Chain(monitor.WithBar, perf.WithBar)(bus.MergeHandlers(riskManager.OnBar, reversionStrategy.OnBar))
	router.OnOrder = middleware.Chain(monitor.WithOrder, perf.WithOrder)(simulator.OnOrder)
	router.OnOrderAcceptance = middleware.Chain(monitor.WithOrderAcceptance, perf.WithOrderAcceptance)(riskManager.OnOrderAccepted)
	router.OnOrderRejection = middleware.Chain(monitor.WithOrderRejection, perf.WithOrderRejection)(riskManager.OnOrderRejected)
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
