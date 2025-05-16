package advisor

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
)

type Strategy struct {
	logger *zap.Logger
	router *bus.Router
}

func NewStrategy(logger *zap.Logger, router *bus.Router) *Strategy {
	return &Strategy{
		logger: logger,
		router: router,
	}
}

func (strategy *Strategy) OnBar(bar *model.Bar) error {
	return nil
}

func (strategy *Strategy) OnTick(tick *model.Tick) error {
	return nil
}
