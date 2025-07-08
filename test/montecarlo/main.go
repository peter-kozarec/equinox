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
	"github.com/peter-kozarec/equinox/pkg/data/mapper"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/simulation"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	StartTime = "2020-01-01 00:00:00"
)

func main() {
	router := bus.NewRouter(1000)

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

	exec := simulation.NewEurUsdMonteCarloTickSimulator(
		sim,
		rand.New(rand.NewSource(time.Now().UnixNano())),
		30*24*time.Hour, // Duration
		0.1607143264,    // Your mu
		0.0698081590,    // Your sigma
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	//db, err := psql.Connect(ctx, "", "", "", "", "")
	//if err != nil {
	//	logger.Fatal("unable to connect to postgres", zap.Error(err))
	//}

	monitor := middleware.NewMonitor(middleware.MonitorPositionsClosed)
	performance := middleware.NewPerformance()
	//ledger := middleware.NewLedger(ctx, logger, db, 13456789, 987654321)

	advisor := strategy.NewMrxAdvisor(router)
	router.TickHandler = middleware.Chain(monitor.WithTick, performance.WithTick)(advisor.NewTick)
	router.BarHandler = middleware.Chain(monitor.WithBar, performance.WithBar)(advisor.NewBar)
	router.OrderHandler = middleware.Chain(monitor.WithOrder, performance.WithOrder)(sim.OnOrder)
	router.PositionOpenedHandler = middleware.Chain(monitor.WithPositionOpened, performance.WithPositionOpened)(middleware.NoopPosOpnHdl)
	router.PositionClosedHandler = middleware.Chain(monitor.WithPositionClosed, performance.WithPositionClosed)(advisor.PositionClosed)
	router.PositionPnLUpdatedHandler = middleware.Chain(monitor.WithPositionPnLUpdated, performance.WithPositionPnLUpdated)(middleware.NoopPosUpdHdl)
	router.EquityHandler = middleware.Chain(monitor.WithEquity, performance.WithEquity)(middleware.NoopEquityHdl)
	router.BalanceHandler = middleware.Chain(monitor.WithBalance, performance.WithBalance)(middleware.NoopBalanceHdl)

	go router.ExecLoop(ctx, exec.DoOnce)

	defer performance.PrintStatistics()
	defer router.PrintStatistics()
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
