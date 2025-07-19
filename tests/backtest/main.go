package main

import (
	"context"
	"errors"
	"github.com/peter-kozarec/equinox/pkg/tools/bar"
	"github.com/peter-kozarec/equinox/pkg/tools/risk"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"

	"github.com/peter-kozarec/equinox/examples/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/data/mapper"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/simulation"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	TickBinDir = "C:\\Users\\peter\\market_data\\"

	StartTime = "2019-01-01 00:00:00"
	EndTime   = "2020-01-01 00:00:00"
)

var (
	barPeriod    = common.BarPeriodM1
	startBalance = fixed.FromInt(10000, 0)

	instrument = common.Instrument{
		Symbol:           "EURUSD",
		Digits:           5,
		PipSize:          fixed.FromInt(1, 4),
		ContractSize:     fixed.FromInt(100000, 0),
		CommissionPerLot: fixed.FromInt(3, 0),
		PipSlippage:      fixed.FromInt(4, 5),
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

	mp := mapper.NewReader[mapper.BinaryTick](TickBinDir + "eurusd.bin")
	if err := mp.Open(); err != nil {
		slog.Error("unable to open mapper", "error", err)
	}
	defer mp.Close()

	startTime, _ := time.Parse(time.DateTime, StartTime)
	endTime, _ := time.Parse(time.DateTime, EndTime)

	audit := simulation.NewAudit(time.Minute)
	sim := simulation.NewSimulator(router, audit, instrument, startBalance)
	exec := simulation.NewExecutor(router, mp, instrument.Symbol, startTime, endTime)
	barBuilder := bar.NewBuilder(router, bar.With(instrument.Symbol, barPeriod, bar.PriceModeBid))

	monitor := middleware.NewMonitor(middleware.MonitorOrders | middleware.MonitorPositionsOpened | middleware.MonitorPositionsClosed)
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
	router.PositionClosedHandler = middleware.Chain(monitor.WithPositionClosed, performance.WithPositionClosed)(riskManager.OnPositionClosed)
	router.PositionPnLUpdatedHandler = middleware.Chain(monitor.WithPositionPnLUpdated, performance.WithPositionPnLUpdated)(riskManager.OnPositionUpdated)
	router.EquityHandler = middleware.Chain(monitor.WithEquity, performance.WithEquity)(riskManager.OnEquity)
	router.BalanceHandler = middleware.Chain(monitor.WithBalance, performance.WithBalance)(riskManager.OnBalance)
	router.SignalHandler = middleware.Chain(monitor.WithSignal, performance.WithSignal)(riskManager.OnSignal)

	if err := exec.LookupStartIndex(); err != nil {
		slog.Error("unable to lookup start index", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

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

	if e := <-router.ExecLoop(ctx, exec.DoOnce); e != nil {
		if !errors.Is(e, context.Canceled) && !errors.Is(e, mapper.ErrEof) {
			slog.Error("unexpected error during execution", "error", e)
			os.Exit(1)
		}
	}

	sim.CloseAllOpenPositions()

	report := audit.GenerateReport()
	report.Print()
}
