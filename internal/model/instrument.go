package model

import (
	"peter-kozarec/equinox/internal/utility/fixed"
)

type Instrument struct {
	Id               int64
	Digits           int
	LotSize          fixed.Point
	DenominationUnit string
}
