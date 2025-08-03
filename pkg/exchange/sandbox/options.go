package sandbox

import (
	"strings"

	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Option func(*Simulator)
type CommissionHandler func(exchange.SymbolInfo, common.Position) fixed.Point
type SwapHandler func(exchange.SymbolInfo, common.Position) fixed.Point
type SlippageHandler func(common.Position) fixed.Point

func WithSymbols(symbols ...exchange.SymbolInfo) Option {
	return func(s *Simulator) {
		for _, symbol := range symbols {
			s.symbolsMap[strings.ToUpper(symbol.SymbolName)] = symbol
		}
	}
}

func WithRateProvider(rateProvider exchange.RateProvider) Option {
	return func(s *Simulator) {
		s.rateProvider = rateProvider
	}
}

func WithCommissionHandler(commissionHandler CommissionHandler) Option {
	return func(s *Simulator) {
		s.commissionHandler = commissionHandler
	}
}

func WithSwapHandler(swapHandler SwapHandler) Option {
	return func(s *Simulator) {
		s.swapHandler = swapHandler
	}
}

func WithSlippageHandler(slippageHandler SlippageHandler) Option {
	return func(s *Simulator) {
		s.slippageHandler = slippageHandler
	}
}

func WithMaintenanceMargin(maintenanceMarginRate fixed.Point) Option {
	return func(s *Simulator) {
		s.maintenanceMarginRate = maintenanceMarginRate
	}
}
