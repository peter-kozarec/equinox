package middleware

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/utility"
)

type Audit struct {
	logger *zap.Logger

	closedPositions []model.Position
}

func NewAudit(logger *zap.Logger) *Audit {
	return &Audit{
		logger: logger,
	}
}

func (audit *Audit) WithBalance(handler bus.BalanceEventHandler) bus.BalanceEventHandler {
	return func(balance *utility.Fixed) error {

		return handler(balance)
	}
}

func (audit *Audit) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position *model.Position) error {
		audit.closedPositions = append(audit.closedPositions, *position)
		return handler(position)
	}
}
