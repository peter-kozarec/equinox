package exchange

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type SymbolClass string

const (
	Forex SymbolClass = "forex"
)

type SymbolInfo struct {
	SymbolName    string
	SymbolId      int64
	Class         SymbolClass
	QuoteCurrency string
	Digits        int
	PipSize       fixed.Point
	ContractSize  fixed.Point
	Leverage      fixed.Point
}
