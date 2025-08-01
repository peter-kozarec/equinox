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

func WithSymbolInfo(symbols ...exchange.SymbolInfo) Option {
	return func(s *Simulator) {
		for _, symbol := range symbols {
			s.symbolsMap[strings.ToUpper(symbol.SymbolName)] = symbol
		}
	}
}

func WithRateProvider(rateProvider RateProvider) Option {
	return func(s *Simulator) {
		s.rateProvider = rateProvider
	}
}

func WithSlippage(slippage fixed.Point) Option {
	return func(s *Simulator) {
		s.slippage = slippage
	}
}

func WithTotalCommissionHandler(commissionHandler CommissionHandler) Option {
	return func(s *Simulator) {
		s.commissionHandler = commissionHandler
	}
}

func WithTotalSwapHandler(swapHandler SwapHandler) Option {
	return func(s *Simulator) {
		s.swapHandler = swapHandler
	}
}
