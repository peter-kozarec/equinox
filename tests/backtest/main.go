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
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/datasource"
	"github.com/peter-kozarec/equinox/pkg/datasource/historical"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/middleware"
	"github.com/peter-kozarec/equinox/pkg/tools/bar"
	"github.com/peter-kozarec/equinox/pkg/tools/metrics"
	"github.com/peter-kozarec/equinox/pkg/tools/risk"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	TickBinDir = "C:\\Users\\peter\\market_data\\"

	StartTime = "2019-01-01 00:00:00"
	EndTime   = "2020-01-01 00:00:00"
)

var (
	symbol       = "EURUSD"
	barPeriod    = common.BarPeriodM1
	startBalance = fixed.FromInt(10000, 0)

	routerCapacity = 1000

	instrument = common.Instrument{
		Symbol:           symbol,
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

	src := historical.NewSource[historical.BinaryTick](TickBinDir + "eurusd.bin")
	if err := src.Open(); err != nil {
		slog.Error("unable to open mapper", "error", err)
	}
	defer src.Close()

	startTime, _ := time.Parse(time.DateTime, StartTime)
	endTime, _ := time.Parse(time.DateTime, EndTime)

	r := bus.NewRouter(routerCapacity)

	s := exchange.NewSimulator(r, instrument, startBalance)
	h := historical.NewTickReader(src, symbol, startTime, endTime)
	b := bar.NewBuilder(r, bar.With(symbol, barPeriod, bar.PriceModeBid))

	m := middleware.NewMonitor(middleware.MonitorOrders | middleware.MonitorPositionsOpened | middleware.MonitorPositionsClosed)
	p := middleware.NewPerformance()

	a := metrics.NewAudit()
	ea := strategy.NewMrxAdvisor(r)
	rm := risk.NewManager(r, instrument, riskConf,
		risk.WithDefaultKellyMultiplier(),
		risk.WithDefaultDrawdownMultiplier(),
		risk.WithDefaultRRRMultiplier(),
		risk.WithOnHourCooldown())

	r.TickHandler = middleware.Chain(m.WithTick, p.WithTick)(bus.MergeHandlers(s.OnTick, rm.OnTick, b.OnTick, ea.OnTick))
	r.BarHandler = middleware.Chain(m.WithBar, p.WithBar)(bus.MergeHandlers(rm.OnBar, ea.OnBar))
	r.OrderHandler = middleware.Chain(m.WithOrder, p.WithOrder)(s.OnOrder)
	r.OrderAcceptedHandler = middleware.Chain(m.WithOrderAccepted, p.WithOrderAccepted)(rm.OnOrderAccepted)
	r.OrderRejectedHandler = middleware.Chain(m.WithOrderRejected, p.WithOrderRejected)(rm.OnOrderRejected)
	r.PositionOpenedHandler = middleware.Chain(m.WithPositionOpened, p.WithPositionOpened)(rm.OnPositionOpened)
	r.PositionClosedHandler = middleware.Chain(m.WithPositionClosed, p.WithPositionClosed)(bus.MergeHandlers(rm.OnPositionClosed, a.OnPositionClosed))
	r.PositionPnLUpdatedHandler = middleware.Chain(m.WithPositionPnLUpdated, p.WithPositionPnLUpdated)(rm.OnPositionUpdated)
	r.EquityHandler = middleware.Chain(m.WithEquity, p.WithEquity)(bus.MergeHandlers(rm.OnEquity, a.OnEquity))
	r.BalanceHandler = middleware.Chain(m.WithBalance, p.WithBalance)(rm.OnBalance)
	r.SignalHandler = middleware.Chain(m.WithSignal, p.WithSignal)(rm.OnSignal)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	rm.OnEquity(ctx, common.Equity{
		Source:      "main",
		Account:     "",
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   time.Now(),
		Value:       startBalance,
	})

	rm.OnBalance(ctx, common.Balance{
		Source:      "main",
		Account:     "",
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   time.Now(),
		Value:       startBalance,
	})

	defer p.PrintStatistics()
	defer r.PrintStatistics()

	if e := <-r.ExecLoop(ctx, datasource.CreateTickDispatcher(r, h)); e != nil {
		if !errors.Is(e, context.Canceled) && !errors.Is(e, historical.ErrEof) {
			slog.Error("unexpected error during execution", "error", e)
			os.Exit(1)
		}
	}

	s.CloseAllOpenPositions()

	report := a.GenerateReport()
	report.Print()
}
