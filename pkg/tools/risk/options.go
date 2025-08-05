package risk

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Option func(*Manager)

type SignalValidationHandler interface {
	ValidateSignal(common.Signal) error
}

func WithSignalValidation(handlers ...SignalValidationHandler) Option {
	return func(m *Manager) {
		m.signalValidationHandlers = handlers
	}
}

func WithRateProvider(provider exchange.RateProvider) Option {
	return func(m *Manager) {
		m.rateProvider = provider
	}
}

type CustomOpenOrderHandler func(entry, sl, tp, size fixed.Point, symbol string) common.Order

func WithCustomOpenOrder(handler CustomOpenOrderHandler) Option {
	return func(m *Manager) {
		m.customOpenOrderHandler = handler
	}
}
