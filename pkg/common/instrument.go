package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type Instrument struct {
	Id               int64
	Digits           int
	LotSize          fixed.Point
	DenominationUnit string
}

func (i Instrument) Fields() []zap.Field {
	return []zap.Field{
		zap.Int64("id", i.Id),
		zap.Int("digits", i.Digits),
		zap.String("lot_size", i.LotSize.String()),
		zap.String("denomination_unit", i.DenominationUnit),
	}
}
