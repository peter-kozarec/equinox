package main

import (
	"context"
	"errors"
	"github.com/peter-kozarec/equinox/pkg/tools/metrics"
	"github.com/peter-kozarec/equinox/pkg/tools/risk"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/peter-kozarec/equinox/examples/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/data/mapper"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/simulation"
	"github.com/peter-kozarec/equinox/pkg/tools/bar"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	barPeriod    = common.BarPeriodM1
	startBalance = fixed.FromInt(10000, 0)

	instrument = common.Instrument{
		Symbol:           "EURUSD",
		Digits:           5,
		PipSize:          fixed.FromInt(1, 4),
		ContractSize:     fixed.FromInt(100000, 0),
		CommissionPerLot: fixed.Zero,
		PipSlippage:      fixed.Zero,
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
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	router := bus.NewRouter(1000)

	sim := simulation.NewSimulator(router, instrument, startBalance)
	barBuilder := bar.NewBuilder(router, bar.With(instrument.Symbol, barPeriod, bar.PriceModeBid))

	exec := simulation.NewEurUsdMonteCarloTickSimulator(
		router,
		instrument.Symbol,
		rand.New(rand.NewSource(time.Now().UnixNano())),
		30*24*time.Hour,
		0.1607143264,
		0.0698081590,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	monitor := middleware.NewMonitor(middleware.MonitorSignals | middleware.MonitorOrders | middleware.MonitorPositionsClosed)
	audit := metrics.NewAudit()
	performance := middleware.NewPerformance()

	advisor := strategy.NewMrxAdvisor(router)
	riskManager := risk.NewManager(router, instrument, riskConf,
		risk.WithDefaultKellyMultiplier(),
		risk.WithDefaultDrawdownMultiplier(),
		risk.WithDefaultRRRMultiplier(),
		risk.WithOnHourCooldown())

	router.TickHandler = middleware.Chain(monitor.WithTick, performance.WithTick)(bus.MergeHandlers(sim.OnTick, riskManager.OnTick, barBuilder.OnTick, advisor.OnTick))
	router.BarHandler = middleware.Chain(monitor.WithBar, performance.WithBar)(bus.MergeHandlers(riskManager.OnBar, advisor.OnBar))
	router.OrderHandler = middleware.Chain(monitor.WithOrder, performance.WithOrder)(sim.OnOrder)
	router.OrderAcceptedHandler = middleware.Chain(monitor.WithOrderAccepted, performance.WithOrderAccepted)(riskManager.OnOrderAccepted)
	router.OrderRejectedHandler = middleware.Chain(monitor.WithOrderRejected, performance.WithOrderRejected)(riskManager.OnOrderRejected)
	router.PositionOpenedHandler = middleware.Chain(monitor.WithPositionOpened, performance.WithPositionOpened)(riskManager.OnPositionOpened)
	router.PositionClosedHandler = middleware.Chain(monitor.WithPositionClosed, performance.WithPositionClosed)(bus.MergeHandlers(riskManager.OnPositionClosed, audit.OnPositionClosed))
	router.PositionPnLUpdatedHandler = middleware.Chain(monitor.WithPositionPnLUpdated, performance.WithPositionPnLUpdated)(riskManager.OnPositionUpdated)
	router.EquityHandler = middleware.Chain(monitor.WithEquity, performance.WithEquity)(bus.MergeHandlers(riskManager.OnEquity, audit.OnEquity))
	router.BalanceHandler = middleware.Chain(monitor.WithBalance, performance.WithBalance)(riskManager.OnBalance)
	router.SignalHandler = middleware.Chain(monitor.WithSignal, performance.WithSignal)(riskManager.OnSignal)

	riskManager.OnEquity(ctx, common.Equity{
		Source:      "main",
		Account:     "",
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   time.Now(),
		Value:       startBalance,
	})

	riskManager.OnBalance(ctx, common.Balance{
		Source:      "main",
		Account:     "",
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   time.Now(),
		Value:       startBalance,
	})

	defer performance.PrintStatistics()
	defer router.PrintStatistics()

	if err := <-router.ExecLoop(ctx, exec.DoOnce); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, mapper.ErrEof) {
			slog.Error("unexpected error during execution", "error", err)
			os.Exit(1)
		}
	}

	sim.CloseAllOpenPositions()

	report := audit.GenerateReport()
	report.Print()
}
