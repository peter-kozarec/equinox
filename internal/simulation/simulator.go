package simulation

import (
	"github.com/govalues/decimal"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
)

type Simulator struct {
	logger *zap.Logger
	router *bus.Router

	equity  model.Equity
	balance model.Balance
}

func NewSimulator(logger *zap.Logger, router *bus.Router) *Simulator {
	simulator := &Simulator{
		logger: logger,
		router: router,
	}

	balance, err := decimal.NewFromFloat64(10000.0)
	if err != nil {
		panic(err)
	}

	simulator.balance = model.Balance(balance)
	simulator.equity = model.Equity(balance)

	return simulator
}

func (simulator *Simulator) OnTick(tick *model.Tick) error {
	return nil
}
