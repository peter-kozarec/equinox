package main

import (
	"context"
	"errors"
	"github.com/peter-kozarec/equinox/pkg/common"
	"log/slog"
	"os"
	"os/signal"
	"time"

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
	barPeriod    = time.Minute
	startBalance = fixed.FromInt(10000, 0)

	instrument = common.Instrument{
		Symbol:           "EURUSD",
		PipSize:          fixed.FromInt(1, 4),
		ContractSize:     fixed.FromInt(100000, 0),
		CommissionPerLot: fixed.FromInt(3, 0),
		PipSlippage:      fixed.FromInt(10, 5),
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
	aggregator := simulation.NewAggregator(barPeriod, instrument, router)
	exec := simulation.NewExecutor(router, mp, instrument.Symbol, startTime, endTime)

	monitor := middleware.NewMonitor(middleware.MonitorPositionsClosed)
	performance := middleware.NewPerformance()

	advisor := strategy.NewMrxAdvisor(router)
	router.TickHandler = middleware.Chain(monitor.WithTick, performance.WithTick)(func(tick common.Tick) {
		sim.OnTick(tick)
		aggregator.OnTick(tick)
		advisor.OnTick(tick)
	})
	router.BarHandler = middleware.Chain(monitor.WithBar, performance.WithBar)(advisor.NewBar)
	router.OrderHandler = middleware.Chain(monitor.WithOrder, performance.WithOrder)(sim.OnOrder)
	router.PositionOpenedHandler = middleware.Chain(monitor.WithPositionOpened, performance.WithPositionOpened)(middleware.NoopPosOpnHdl)
	router.PositionClosedHandler = middleware.Chain(monitor.WithPositionClosed, performance.WithPositionClosed)(advisor.PositionClosed)
	router.PositionPnLUpdatedHandler = middleware.Chain(monitor.WithPositionPnLUpdated, performance.WithPositionPnLUpdated)(middleware.NoopPosUpdHdl)
	router.EquityHandler = middleware.Chain(monitor.WithEquity, performance.WithEquity)(middleware.NoopEquityHdl)
	router.BalanceHandler = middleware.Chain(monitor.WithBalance, performance.WithBalance)(middleware.NoopBalanceHdl)

	if err := exec.LookupStartIndex(); err != nil {
		slog.Error("unable to lookup start index", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	go router.ExecLoop(ctx, exec.DoOnce)

	defer performance.PrintStatistics()
	defer router.PrintStatistics()

	if err := <-router.Done(); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, mapper.ErrEof) {
			slog.Error("unexpected error during execution", "error", err)
			os.Exit(1)
		}
	}

	sim.CloseAllOpenPositions()

	report := audit.GenerateReport()
	report.Print()
}
