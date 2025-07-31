package sandbox

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"strings"
)

type Option func(*Simulator)
type TotalCommissionHandler func(exchange.SymbolInfo, common.Position) fixed.Point
type TotalSwapHandler func(exchange.SymbolInfo, common.Position) fixed.Point

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

func WithTotalCommissionHandler(h TotalCommissionHandler) Option {
	return func(s *Simulator) {
		s.totalCommissionHandler = h
	}
}

func WithTotalSwapHandler(h TotalSwapHandler) Option {
	return func(s *Simulator) {
		s.totalSwapHandler = h
	}
}
