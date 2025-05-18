package utility

import (
	"fmt"
)

type Fixed struct {
	Value     int64
	Precision int32
}

var pow10 = [...]int64{
	1, 10, 100, 1_000, 10_000,
	100_000, 1_000_000, 10_000_000, 100_000_000,
	1_000_000_000, 10_000_000_000, 100_000_000_000,
	1_000_000_000_000, 10_000_000_000_000,
	100_000_000_000_000, 1_000_000_000_000_000,
	10_000_000_000_000_000, 100_000_000_000_000_000,
	1_000_000_000_000_000_000,
}

func NewFixed(value int64, precision int32) Fixed {
	return Fixed{
		Value:     value,
		Precision: precision,
	}
}

func (fixed Fixed) normalizeTo(exp int32) int64 {
	diff := int(exp - fixed.Precision)
	if diff == 0 {
		return fixed.Value
	} else if diff > 0 {
		return fixed.Value * pow10[diff]
	}
	return fixed.Value / pow10[-diff]
}

func (fixed Fixed) Abs() Fixed {
	if fixed.Value > 0 {
		return fixed
	}
	return fixed.MulInt(-1)
}

func (fixed Fixed) Add(o Fixed) Fixed {
	exp := max(fixed.Precision, o.Precision)
	return Fixed{
		Value:     fixed.normalizeTo(exp) + o.normalizeTo(exp),
		Precision: exp,
	}
}

func (fixed Fixed) Sub(o Fixed) Fixed {
	exp := max(fixed.Precision, o.Precision)
	return Fixed{
		Value:     fixed.normalizeTo(exp) - o.normalizeTo(exp),
		Precision: exp,
	}
}

func (fixed Fixed) Mul(o Fixed) Fixed {
	return Fixed{
		Value:     fixed.Value * o.Value,
		Precision: fixed.Precision + o.Precision,
	}
}

func (fixed Fixed) Div(o Fixed) Fixed {
	if o.Value == 0 {
		panic("division by zero")
	}
	adjusted := fixed.Value * pow10[o.Precision]
	return Fixed{
		Value:     adjusted / o.Value,
		Precision: fixed.Precision,
	}
}

func (fixed Fixed) AddInt(v int64) Fixed {
	return Fixed{
		Value:     fixed.Value + v*pow10[fixed.Precision],
		Precision: fixed.Precision,
	}
}

func (fixed Fixed) SubInt(v int64) Fixed {
	return Fixed{
		Value:     fixed.Value - v*pow10[fixed.Precision],
		Precision: fixed.Precision,
	}
}

func (fixed Fixed) MulInt(v int64) Fixed {
	return Fixed{
		Value:     fixed.Value * v,
		Precision: fixed.Precision,
	}
}

func (fixed Fixed) DivInt(v int64) Fixed {
	if v == 0 {
		panic("division by zero")
	}
	return Fixed{
		Value:     fixed.Value / v,
		Precision: fixed.Precision,
	}
}

func (fixed Fixed) Float64() float64 {
	return float64(fixed.Value) / float64(pow10[fixed.Precision])
}

func (fixed Fixed) String() string {
	intPart := fixed.Value / pow10[fixed.Precision]
	fracPart := fixed.Value % pow10[fixed.Precision]

	if fracPart < 0 {
		fracPart = -fracPart
	}

	fracStr := fmt.Sprintf("%0*d", fixed.Precision, fracPart)
	return fmt.Sprintf("%d.%s", intPart, fracStr)
}

func (fixed Fixed) Eq(o Fixed) bool {
	exp := max(fixed.Precision, o.Precision)
	return fixed.normalizeTo(exp) == o.normalizeTo(exp)
}

func (fixed Fixed) Gt(o Fixed) bool {
	exp := max(fixed.Precision, o.Precision)
	return fixed.normalizeTo(exp) > o.normalizeTo(exp)
}

func (fixed Fixed) Lt(o Fixed) bool {
	exp := max(fixed.Precision, o.Precision)
	return fixed.normalizeTo(exp) < o.normalizeTo(exp)
}

func (fixed Fixed) Gte(o Fixed) bool {
	return fixed.Eq(o) || fixed.Gt(o)
}

func (fixed Fixed) Lte(o Fixed) bool {
	return fixed.Eq(o) || fixed.Lt(o)
}
