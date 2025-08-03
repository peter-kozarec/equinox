package risk

import (
	"errors"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

var (
	ErrMaxRiskInvalid    = errors.New("max risk lower or equal to zero")
	ErrMinRiskInvalid    = errors.New("min risk lower or equal to zero")
	ErrBaseRiskIsInvalid = errors.New("base risk lower or equal to zero")
	ErrOpenRiskIsInvalid = errors.New("open risk lower or equal to zero")
	ErrOpenRiskSmall     = errors.New("open risk is lower then max risk")
)

type Configuration struct {
	MaxRiskRate  fixed.Point
	MinRiskRate  fixed.Point
	BaseRiskRate fixed.Point
	OpenRiskRate fixed.Point

	SizeDigits int
}

func (c Configuration) validate() error {
	if c.MaxRiskRate.Lte(fixed.Zero) {
		return ErrMaxRiskInvalid
	}
	if c.MinRiskRate.Lte(fixed.Zero) {
		return ErrMinRiskInvalid
	}
	if c.BaseRiskRate.Lte(fixed.Zero) {
		return ErrBaseRiskIsInvalid
	}
	if c.OpenRiskRate.Lte(fixed.Zero) {
		return ErrOpenRiskIsInvalid
	}
	if c.OpenRiskRate.Lte(c.MinRiskRate) {
		return ErrOpenRiskSmall
	}
	return nil
}
