package middleware

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
)

type Audit struct {
	logger *zap.Logger
}

func NewAudit(logger *zap.Logger) *Audit {
	return &Audit{
		logger: logger,
	}
}

func (audit *Audit) WithTick(handler bus.TickEventHandler) bus.TickEventHandler {
	return func(tick *model.Tick) error {
		audit.logger.Debug("audit", zap.Any("tick", tick))
		return handler(tick)
	}
}

func (audit *Audit) WithBar(handler bus.BarEventHandler) bus.BarEventHandler {
	return func(bar *model.Bar) error {
		audit.logger.Debug("audit", zap.Any("bar", bar))
		return handler(bar)
	}
}
