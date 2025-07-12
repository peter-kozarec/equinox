package main

import (
	"context"
	"errors"
	"github.com/peter-kozarec/equinox/pkg/tools/bar"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"math/rand"
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

var (
	barPeriod    = common.BarPeriodM1
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

	audit := simulation.NewAudit(time.Minute)
	sim := simulation.NewSimulator(router, audit, instrument, startBalance)
	barBuilder := bar.NewBuilder(router, bar.BuildModeTickBased, bar.PriceModeBid, true)
	barBuilder.Build(instrument.Symbol, barPeriod)

	exec := simulation.NewEurUsdMonteCarloTickSimulator(
		router,
		instrument.Symbol,
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

	monitor := middleware.NewMonitor(middleware.MonitorBars)
	performance := middleware.NewPerformance()
	//ledger := middleware.NewLedger(ctx, logger, db, 13456789, 987654321)

	advisor := strategy.NewMrxAdvisor(router)

	router.TickHandler = middleware.Chain(monitor.WithTick, performance.WithTick)(bus.MergeHandlers(sim.OnTick, barBuilder.OnTick, advisor.OnTick))
	router.BarHandler = middleware.Chain(monitor.WithBar, performance.WithBar)(advisor.OnBar)
	router.OrderHandler = middleware.Chain(monitor.WithOrder, performance.WithOrder)(sim.OnOrder)
	router.OrderAcceptedHandler = middleware.Chain(monitor.WithOrderAccepted, performance.WithOrderAccepted)(middleware.NoopOrderAccHdl)
	router.OrderRejectedHandler = middleware.Chain(monitor.WithOrderRejected, performance.WithOrderRejected)(middleware.NoopOrderRjctHdl)
	router.PositionOpenedHandler = middleware.Chain(monitor.WithPositionOpened, performance.WithPositionOpened)(middleware.NoopPosOpnHdl)
	router.PositionClosedHandler = middleware.Chain(monitor.WithPositionClosed, performance.WithPositionClosed)(advisor.OnPositionClosed)
	router.PositionPnLUpdatedHandler = middleware.Chain(monitor.WithPositionPnLUpdated, performance.WithPositionPnLUpdated)(middleware.NoopPosUpdHdl)
	router.EquityHandler = middleware.Chain(monitor.WithEquity, performance.WithEquity)(middleware.NoopEquityHdl)
	router.BalanceHandler = middleware.Chain(monitor.WithBalance, performance.WithBalance)(middleware.NoopBalanceHdl)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		errCh := router.ExecLoop(ctx, exec.DoOnce)
		select {
		case e := <-errCh:
			return e
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	defer performance.PrintStatistics()
	defer router.PrintStatistics()

	if e := g.Wait(); e != nil {
		if !errors.Is(e, context.Canceled) && !errors.Is(e, mapper.ErrEof) {
			slog.Error("unexpected error during execution", "error", e)
			os.Exit(1)
		}
	}

	sim.CloseAllOpenPositions()

	report := audit.GenerateReport()
	report.Print()
}
