package main

import (
	"context"
	"errors"
	"github.com/peter-kozarec/equinox/internal/strategy"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/data/mapper"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/simulation"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"time"
)

const (
	StartTime = "2020-01-01 00:00:00"
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

	simConf := simulation.Configuration{
		BarPeriod:        time.Minute,
		PipSize:          fixed.New(1, 4),
		ContractSize:     fixed.New(100000, 0),
		CommissionPerLot: fixed.New(3, 0),
		StartBalance:     fixed.New(10000, 0),
		PipSlippage:      fixed.New(10, 5),
	}

	audit := simulation.NewAudit(logger, time.Minute)
	sim := simulation.NewSimulator(logger, router, audit, simConf)

	exec := simulation.NewEurUsdMonteCarloTickSimulator(
		logger,
		sim,
		30*24*time.Hour, // Duration
		0.1607143264,    // Your mu
		0.0698081590,    // Your sigma
	)

	telemetry := middleware.NewTelemetry(logger)
	monitor := middleware.NewMonitor(logger, middleware.MonitorPositionsClosed|middleware.MonitorBalance)

	advisor := strategy.NewAdvisor(logger, router)
	router.TickHandler = middleware.Chain(telemetry.WithTick, monitor.WithTick)(advisor.NewTick)
	router.BarHandler = middleware.Chain(telemetry.WithBar, monitor.WithBar)(advisor.NewBar)
	router.OrderHandler = middleware.Chain(telemetry.WithOrder, monitor.WithOrder)(sim.OnOrder)
	router.PositionOpenedHandler = middleware.Chain(telemetry.WithPositionOpened, monitor.WithPositionOpened)(middleware.NoopPosOpnHdl)
	router.PositionClosedHandler = middleware.Chain(telemetry.WithPositionClosed, monitor.WithPositionClosed)(advisor.PositionClosed)
	router.PositionPnLUpdatedHandler = middleware.Chain(telemetry.WithPositionPnLUpdated, monitor.WithPositionPnLUpdated)(middleware.NoopPosUpdHdl)
	router.EquityHandler = middleware.Chain(telemetry.WithEquity, monitor.WithEquity)(middleware.NoopEquityHdl)
	router.BalanceHandler = middleware.Chain(telemetry.WithBalance, monitor.WithBalance)(middleware.NoopBalanceHdl)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	go router.ExecLoop(ctx, exec.DoOnce)

	defer router.PrintStatistics()
	defer telemetry.PrintStatistics()
	defer sim.PrintDetails()

	if err := <-router.Done(); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, mapper.EOF) {
			logger.Fatal("unexpected error during execution", zap.Error(err))
		}
	}

	sim.CloseAllOpenPositions()

	report := audit.GenerateReport()
	report.Print(logger)
}
