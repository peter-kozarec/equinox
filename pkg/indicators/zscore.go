package indicators

import (
	"errors"

	"github.com/peter-kozarec/equinox/pkg/utility/circular"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type ZScore struct {
	windowSize int
	data       *circular.PointBuffer
}

func NewZScore(windowSize int) *ZScore {
	return &ZScore{
		windowSize: windowSize,
		data:       circular.NewPointBuffer(uint(windowSize)),
	}
}

func (z *ZScore) AddPoint(p fixed.Point) {
	z.data.PushUpdate(p)
}

func (z *ZScore) Value() (fixed.Point, error) {
	if !z.IsReady() {
		return fixed.Point{}, errors.New("not enough data")
	}

	lastPoint := z.data.B.First()
	mean := z.data.Mean()
	stdDev := z.data.StdDev()

	return lastPoint.Sub(mean).Div(stdDev), nil
}

func (z *ZScore) IsReady() bool {
	return z.data.B.IsFull()
}
