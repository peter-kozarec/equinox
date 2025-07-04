package circular

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type PointBuffer struct {
	B        *Buffer[fixed.Point]
	capacity uint

	mean       fixed.Point
	stdDev     fixed.Point
	sum        fixed.Point
	sumSquares fixed.Point
	variance   fixed.Point
}

func NewPointBuffer(capacity uint) *PointBuffer {
	return &PointBuffer{
		capacity: capacity,
		B:        NewBuffer[fixed.Point](capacity),
	}
}

func (p *PointBuffer) PushUpdate(v fixed.Point) {
	if p.B.IsEmpty() {
		p.B.Push(v)
		p.sum = v
		p.sumSquares = v.Mul(v)
	} else if !p.B.IsFull() {
		p.B.Push(v)
		p.sum = p.sum.Add(v)
		p.sumSquares = p.sumSquares.Add(v.Mul(v))
	} else {
		toBeRemoved := p.B.Last()
		p.B.Push(v)
		p.sum = p.sum.Sub(toBeRemoved).Add(v)
		p.sumSquares = p.sumSquares.Sub(toBeRemoved.Mul(toBeRemoved)).Add(v.Mul(v))
	}

	p.mean = p.sum.Div(fixed.FromInt64(int64(p.B.Size()), 0))
	p.variance = p.sumSquares.Div(fixed.FromInt64(int64(p.B.Size()), 0)).Sub(p.mean.Mul(p.mean))
	if p.variance.Gt(fixed.Zero) {
		p.stdDev = p.variance.Sqrt()
	} else {
		p.stdDev = fixed.Zero
	}
}

func (p *PointBuffer) Clear() {
	p.B = nil
	p.B = NewBuffer[fixed.Point](p.capacity)

	p.mean = fixed.Zero
	p.stdDev = fixed.Zero
	p.sum = fixed.Zero
	p.sumSquares = fixed.Zero
	p.variance = fixed.Zero
}

func (p *PointBuffer) Mean() fixed.Point {
	return p.mean
}

func (p *PointBuffer) Sum() fixed.Point {
	return p.sum
}

func (p *PointBuffer) StdDev() fixed.Point {
	return p.stdDev
}

func (p *PointBuffer) Variance() fixed.Point {
	return p.variance
}
