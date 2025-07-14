package indicators

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type ZScore struct {
	windowSize int
	data       *fixed.RingBuffer
}

func NewZScore(windowSize int) *ZScore {
	return &ZScore{
		windowSize: windowSize,
		data:       fixed.NewRingBuffer(windowSize),
	}
}

func (z *ZScore) AddPoint(p fixed.Point) {
	z.data.Add(p)
}

func (z *ZScore) Value() fixed.Point {

	lastPoint := z.data.Latest()
	mean := z.data.Mean()
	stdDev := z.data.SampleStdDev()

	return lastPoint.Sub(mean).Div(stdDev)
}

func (z *ZScore) IsReady() bool {
	return z.data.IsFull()
}
