package model

import "peter-kozarec/equinox/internal/utility"

type SymbolInfo struct {
	Id               int64
	Digits           int
	LotSize          utility.Fixed
	DenominationUnit string
}
