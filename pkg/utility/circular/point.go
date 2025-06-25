package circular

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Point struct {
	b *Buffer[fixed.Point]

	mean       fixed.Point
	stdDev     fixed.Point
	sum        fixed.Point
	sumSquares fixed.Point
	variance   fixed.Point
}

func CreatePointBuffer(capacity uint) *Point {
	return &Point{
		b: NewBuffer[fixed.Point](capacity),
	}
}

func (p *Point) PushUpdate(v fixed.Point) {
	if p.b.IsEmpty() {
		p.b.Push(v)
		p.sum = v
		p.sumSquares = v.Mul(v)
	} else if !p.b.IsFull() {
		p.b.Push(v)
		p.sum = p.sum.Add(v)
		p.sumSquares = p.sumSquares.Add(v.Mul(v))
	} else {
		toBeRemoved := p.b.Last()
		p.b.Push(v)
		p.sum = p.sum.Sub(toBeRemoved).Add(v)
		p.sumSquares = p.sumSquares.Sub(toBeRemoved.Mul(toBeRemoved)).Add(v.Mul(v))
	}

	p.mean = p.sum.Div(fixed.FromUint(uint64(p.b.Size()), 0))
	p.variance = p.sumSquares.Div(fixed.FromUint(uint64(p.b.Size()), 0)).Sub(p.mean.Mul(p.mean))
	if p.variance.Gt(fixed.Zero) {
		p.stdDev = p.variance.Sqrt()
	} else {
		p.stdDev = fixed.Zero
	}
}

func (p *Point) Mean() fixed.Point {
	return p.mean
}

func (p *Point) Sum() fixed.Point {
	return p.sum
}

func (p *Point) StdDev() fixed.Point {
	return p.stdDev
}

func (p *Point) Variance() fixed.Point {
	return p.variance
}
