package model

import (
	"peter-kozarec/equinox/internal/utility/fixed"
)

type SymbolInfo struct {
	Id               int64
	Digits           int
	LotSize          fixed.Point
	DenominationUnit string
}
