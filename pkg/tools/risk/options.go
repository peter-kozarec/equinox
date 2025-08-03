package risk

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"strings"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange"
)

type Option func(*Manager)

func WithSymbols(symbols ...exchange.SymbolInfo) Option {
	return func(m *Manager) {
		for _, symbol := range symbols {
			m.symbols[strings.ToUpper(symbol.SymbolName)] = symbol
		}
	}
}

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
