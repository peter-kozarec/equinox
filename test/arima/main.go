package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/peter-kozarec/equinox/pkg/models/arima"

	"github.com/peter-kozarec/equinox/examples/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/data/mapper"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/simulation"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

const (
	TickBinDir = "C:\\Users\\peter\\market_data\\"

	StartTime = "2019-01-01 00:00:00"
	EndTime   = "2020-01-01 00:00:00"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("unable to initializing logger: %v", err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	router := bus.NewRouter(logger, 1000)

	mp := mapper.NewReader[mapper.BinaryTick](TickBinDir + "eurusd.bin")
	if err := mp.Open(); err != nil {
		logger.Fatal("unable to open mapper", zap.Error(err))
	}
	defer mp.Close()

	startTime, _ := time.Parse(time.DateTime, StartTime)
	endTime, _ := time.Parse(time.DateTime, EndTime)

	simConf := simulation.Configuration{
		BarPeriod:        10 * time.Minute,
		PipSize:          fixed.FromInt64(1, 4),
		ContractSize:     fixed.FromInt64(100000, 0),
		CommissionPerLot: fixed.FromInt64(3, 0),
		StartBalance:     fixed.FromInt64(10000, 0),
		PipSlippage:      fixed.FromInt64(10, 5),
	}

	audit := simulation.NewAudit(logger, time.Minute)
	sim := simulation.NewSimulator(logger, router, audit, simConf)
	exec := simulation.NewExecutor(logger, sim, mp, startTime, endTime)

	telemetry := middleware.NewTelemetry(logger)
	monitor := middleware.NewMonitor(logger, middleware.MonitorNone)
	performance := middleware.NewPerformance(logger)

	model, err := arima.NewModel(3, 1, 0, 144,
		arima.WithEstimationMethod(arima.ConditionalLeastSquares),
		arima.WithConstant(false),
		arima.WithSeasonal(1))
	if err != nil {
		logger.Fatal("unable to initialize arima model", zap.Error(err))
	}

	advisor := strategy.NewArimaAdvisor(logger, router, model)
	router.TickHandler = middleware.Chain(telemetry.WithTick, monitor.WithTick, performance.WithTick)(middleware.NoopTickHdl)
	router.BarHandler = middleware.Chain(telemetry.WithBar, monitor.WithBar, performance.WithBar)(advisor.OnNewBar)

	if err := exec.LookupStartIndex(); err != nil {
		logger.Fatal("unable to lookup start index", zap.Error(err))
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
			logger.Fatal("unexpected error during execution", zap.Error(err))
		}
	}

	sim.CloseAllOpenPositions()

	report := audit.GenerateReport()
	report.Print(logger)
}
