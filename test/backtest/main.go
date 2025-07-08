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

func main() {
	router := bus.NewRouter(1000)

	mp := mapper.NewReader[mapper.BinaryTick](TickBinDir + "eurusd.bin")
	if err := mp.Open(); err != nil {
		slog.Error("unable to open mapper", "error", err)
	}
	defer mp.Close()

	startTime, _ := time.Parse(time.DateTime, StartTime)
	endTime, _ := time.Parse(time.DateTime, EndTime)

	simConf := simulation.Configuration{
		BarPeriod:        time.Minute,
		PipSize:          fixed.FromInt(1, 4),
		ContractSize:     fixed.FromInt(100000, 0),
		CommissionPerLot: fixed.FromInt(3, 0),
		StartBalance:     fixed.FromInt(10000, 0),
		PipSlippage:      fixed.FromInt(10, 5),
	}

	audit := simulation.NewAudit(time.Minute)
	sim := simulation.NewSimulator(router, audit, simConf)
	exec := simulation.NewExecutor(sim, mp, startTime, endTime)

	telemetry := middleware.NewTelemetry()
	monitor := middleware.NewMonitor(middleware.MonitorPositionsClosed)
	performance := middleware.NewPerformance()

	advisor := strategy.NewMrxAdvisor(router)
	router.TickHandler = middleware.Chain(telemetry.WithTick, monitor.WithTick, performance.WithTick)(advisor.NewTick)
	router.BarHandler = middleware.Chain(telemetry.WithBar, monitor.WithBar, performance.WithBar)(advisor.NewBar)
	router.OrderHandler = middleware.Chain(telemetry.WithOrder, monitor.WithOrder, performance.WithOrder)(sim.OnOrder)
	router.PositionOpenedHandler = middleware.Chain(telemetry.WithPositionOpened, monitor.WithPositionOpened, performance.WithPositionOpened)(middleware.NoopPosOpnHdl)
	router.PositionClosedHandler = middleware.Chain(telemetry.WithPositionClosed, monitor.WithPositionClosed, performance.WithPositionClosed)(advisor.PositionClosed)
	router.PositionPnLUpdatedHandler = middleware.Chain(telemetry.WithPositionPnLUpdated, monitor.WithPositionPnLUpdated, performance.WithPositionPnLUpdated)(middleware.NoopPosUpdHdl)
	router.EquityHandler = middleware.Chain(telemetry.WithEquity, monitor.WithEquity, performance.WithEquity)(middleware.NoopEquityHdl)
	router.BalanceHandler = middleware.Chain(telemetry.WithBalance, monitor.WithBalance, performance.WithBalance)(middleware.NoopBalanceHdl)

	if err := exec.LookupStartIndex(); err != nil {
		slog.Error("unable to lookup start index", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	go router.ExecLoop(ctx, exec.DoOnce)

	defer performance.PrintStatistics(telemetry)
	defer router.PrintStatistics()
	defer telemetry.PrintStatistics()
	defer sim.PrintDetails()

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
