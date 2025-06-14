package model

import (
	"peter-kozarec/equinox/pkg/utility/fixed"
)

type Instrument struct {
	Id               int64
	Digits           int
	LotSize          fixed.Point
	DenominationUnit string
}
