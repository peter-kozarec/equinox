package risk

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

func calcTrueRange(lastClose fixed.Point, bar common.Bar) fixed.Point {
	_1 := bar.High.Sub(bar.Low).Abs()
	_2 := bar.High.Sub(lastClose).Abs()
	_3 := bar.Low.Sub(lastClose).Abs()

	currentTr := _1
	if _2.Gt(currentTr) {
		currentTr = _2
	}
	if _3.Gt(currentTr) {
		currentTr = _3
	}

	return currentTr
}
