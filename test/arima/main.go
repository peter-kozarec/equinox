package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"

	"github.com/peter-kozarec/equinox/pkg/models/arima"

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
	router := bus.NewRouter(1000)

	mp := mapper.NewReader[mapper.BinaryTick](TickBinDir + "eurusd.bin")
	if err := mp.Open(); err != nil {
		slog.Error("unable to open mapper", "error", err)
		os.Exit(1)
	}
	defer mp.Close()

	startTime, _ := time.Parse(time.DateTime, StartTime)
	endTime, _ := time.Parse(time.DateTime, EndTime)

	audit := simulation.NewAudit(time.Minute)
	sim := simulation.NewSimulator(router, audit, instrument, startBalance)
	aggregator := simulation.NewAggregator(barPeriod, instrument, router)
	exec := simulation.NewExecutor(router, mp, instrument.Symbol, startTime, endTime)

	monitor := middleware.NewMonitor(middleware.MonitorNone)
	performance := middleware.NewPerformance()

	model, err := arima.NewModel(3, 1, 0, 144,
		arima.WithEstimationMethod(arima.ConditionalLeastSquares),
		arima.WithConstant(false),
		arima.WithSeasonal(1))
	if err != nil {
		slog.Error("unable to initialize arima model", "error", err)
		os.Exit(1)
	}

	advisor := strategy.NewArimaAdvisor(router, model)
	router.TickHandler = middleware.Chain(monitor.WithTick, performance.WithTick)(func(ctx context.Context, tick common.Tick) {
		sim.OnTick(ctx, tick)
		aggregator.OnTick(ctx, tick)
	})
	router.BarHandler = middleware.Chain(monitor.WithBar, performance.WithBar)(advisor.OnNewBar)

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
