package main

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"peter-kozarec/equinox/cmd/mrx"
	"peter-kozarec/equinox/cmd/mrx/advisor"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/data/mapper"
	"peter-kozarec/equinox/internal/middleware"
	"peter-kozarec/equinox/internal/simulation"
	"syscall"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	logger.Info("MRX started", zap.String("environment", "backtest"), zap.String("version", mrx.Version))
	defer logger.Info("MRX finished")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	reader := mapper.NewReader[mapper.BinaryTick](TickDataSource)
	if err := reader.Open(); err != nil {
		logger.Fatal("error opening tick data reader", zap.Error(err))
	}
	defer reader.Close()

	// Create
	monitor := middleware.NewMonitor(logger, MonitorFlags)
	telemetry := middleware.NewTelemetry(logger)

	router := bus.NewRouter(logger, RouterEventCapacity)

	strategy := advisor.NewStrategy(logger, router)
	simulator := simulation.NewSimulator(logger, router)
	executor := simulation.NewExecutor(logger, simulator, reader, SimulationStart, SimulationEnd)

	// Initialize middleware
	router.TickHandler = middleware.Chain(monitor.WithTick, telemetry.WithTick)(strategy.OnTick)
	router.BarHandler = middleware.Chain(monitor.WithBar, telemetry.WithBar)(strategy.OnBar)
	router.BalanceHandler = middleware.Chain(monitor.WithBalance, telemetry.WithBalance)(strategy.OnBalance)
	router.EquityHandler = middleware.Chain(monitor.WithEquity, telemetry.WithEquity)(strategy.OnEquity)
	router.PositionOpenedHandler = middleware.Chain(monitor.WithPositionOpened, telemetry.WithPositionOpened)(strategy.OnPositionOpened)
	router.PositionClosedHandler = middleware.Chain(monitor.WithPositionClosed, telemetry.WithPositionClosed)(strategy.OnPositionClosed)
	router.PositionPnLUpdatedHandler = middleware.Chain(monitor.WithPositionPnLUpdated, telemetry.WithPositionPnLUpdated)(strategy.OnPositionPnlUpdated)
	router.OrderHandler = middleware.Chain(monitor.WithOrder, telemetry.WithOrder)(simulator.OnOrder)

	// Execute the simulation
	go router.Exec(ctx, executor.Feed)

	defer router.PrintStatistics()
	defer telemetry.PrintStatistics()

	// Wait for the simulation to complete
	if err := <-router.Done(); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, mapper.EOF) {
			logger.Error("error during simulation", zap.Error(err))
		}
	}
}
