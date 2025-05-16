package main

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"peter-kozarec/equinox/cmd/mrx"
	"peter-kozarec/equinox/cmd/mrx/advisor"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/data/mapper"
	"peter-kozarec/equinox/internal/middleware"
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/simulation"
	"syscall"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	logger.Info(fmt.Sprintf("mrx %s", mrx.Version))
	defer logger.Info("done")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	m := mapper.NewReader[model.Tick](TickDataSource)
	if err := m.Open(); err != nil {
		logger.Fatal("error opening tick data reader", zap.Error(err))
	}
	defer m.Close()

	// Create
	monitor := middleware.NewMonitor(logger, MonitorFlags)
	telemetry := middleware.NewTelemetry(logger)

	router := bus.NewRouter(logger, RouterEventCapacity)

	strategy := advisor.NewStrategy(logger, router)
	simulator := simulation.NewSimulator(logger, router)
	executor := simulation.NewExecutor(logger, simulator, m, SimulationStart, SimulationEnd)

	// Initialize
	router.TickHandler = telemetry.WithTick(monitor.WithTick(strategy.OnTick))
	router.BarHandler = telemetry.WithBar(monitor.WithBar(strategy.OnBar))
	router.BalanceHandler = telemetry.WithBalance(monitor.WithBalance(strategy.OnBalance))
	router.EquityHandler = telemetry.WithEquity(monitor.WithEquity(strategy.OnEquity))
	router.PositionOpenedHandler = telemetry.WithPositionOpened(monitor.WithPositionOpened(strategy.OnPositionOpened))
	router.PositionClosedHandler = telemetry.WithPositionClosed(monitor.WithPositionClosed(strategy.OnPositionClosed))
	router.PositionPnLUpdatedHandler = telemetry.WithPositionPnLUpdated(monitor.WithPositionPnLUpdated(strategy.OnPositionPnlUpdated))
	router.OrderHandler = telemetry.WithOrder(monitor.WithOrder(simulator.OnOrder))

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
