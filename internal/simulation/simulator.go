package simulation

import (
	"fmt"
	"github.com/govalues/decimal"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
	"time"
)

type Simulator struct {
	logger     *zap.Logger
	router     *bus.Router
	aggregator *Aggregator

	equity  model.Equity
	balance model.Balance

	simulationTime time.Time
}

func NewSimulator(logger *zap.Logger, router *bus.Router) *Simulator {
	simulator := &Simulator{
		logger:     logger,
		router:     router,
		aggregator: NewAggregator(BarPeriod, router),
	}

	balance, err := decimal.NewFromFloat64(StartingBalance)
	if err != nil {
		panic(err)
	}

	simulator.balance = model.Balance(balance)
	simulator.equity = model.Equity(balance)

	return simulator
}

func (simulator *Simulator) OnTick(tick *model.Tick) error {
	// Set simulation time from processed tick
	simulator.simulationTime = time.Unix(0, tick.TimeStamp)

	// Store balance and equity before processing the tick
	lastBalance := simulator.balance
	lastEquity := simulator.equity

	// Post balance event if the current balance changed after the tick was processed
	if lastBalance != simulator.balance {
		if err := simulator.router.Post(bus.BalanceEvent, &simulator.balance); err != nil {
			return fmt.Errorf("post balance event: %w", err)
		}
	}
	// Post equity event if the current equity changed after the tick was processed
	if lastEquity != simulator.equity {
		if err := simulator.router.Post(bus.EquityEvent, &simulator.equity); err != nil {
			return fmt.Errorf("post equity event: %w", err)
		}
	}

	// Snapshot the balance and equity

	// Post the tick
	if err := simulator.router.Post(bus.TickEvent, tick); err != nil {
		return fmt.Errorf("post tick event: %w", err)
	}

	// Aggregate into bars
	if err := simulator.aggregator.OnTick(tick); err != nil {
		return fmt.Errorf("error aggregating ticks: %w", err)
	}

	return nil
}
