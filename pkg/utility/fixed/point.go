package fixed

import (
	"github.com/govalues/decimal"
	"go.uber.org/zap/zapcore"
)

var (
	NegOne = New(-1, 0)
	Zero   = New(0, 0)
	One    = New(1, 0)

	Sqrt252 = New(1587450786638754, 14)
)

type Point struct {
	v decimal.Decimal
}

func New(v int64, precision int) Point {
	return Point{must(decimal.New(v, precision))}
}

func FromUint(v uint64, precision int) Point {
	return Point{must(decimal.New(int64(v), precision))}
}

func FromFloat(v float64) Point {
	return Point{must(decimal.NewFromFloat64(v))}
}

func (p Point) Add(o Point) Point { return Point{must(p.v.Add(o.v))} }
func (p Point) Sub(o Point) Point { return Point{must(p.v.Sub(o.v))} }
func (p Point) Mul(o Point) Point { return Point{must(p.v.Mul(o.v))} }
func (p Point) Div(o Point) Point { return Point{must(p.v.Quo(o.v))} }

func (p Point) AddInt64(v int64) Point { return p.Add(New(v, 0)) }
func (p Point) AddInt(v int) Point     { return p.Add(New(int64(v), 0)) }
func (p Point) SubInt64(v int64) Point { return p.Sub(New(v, 0)) }
func (p Point) SubInt(v int) Point     { return p.Sub(New(int64(v), 0)) }
func (p Point) MulInt64(v int64) Point { return p.Mul(New(v, 0)) }
func (p Point) MulInt(v int) Point     { return p.Mul(New(int64(v), 0)) }
func (p Point) DivInt64(v int64) Point { return p.Div(New(v, 0)) }
func (p Point) DivInt(v int) Point     { return p.Div(New(int64(v), 0)) }

func (p Point) Eq(o Point) bool  { return p.v.Cmp(o.v) == 0 }
func (p Point) Gt(o Point) bool  { return p.v.Cmp(o.v) > 0 }
func (p Point) Lt(o Point) bool  { return p.v.Cmp(o.v) < 0 }
func (p Point) Gte(o Point) bool { return p.v.Cmp(o.v) >= 0 }
func (p Point) Lte(o Point) bool { return p.v.Cmp(o.v) <= 0 }

func (p Point) Abs() Point     { return Point{p.v.Abs()} }
func (p Point) Neg() Point     { return Point{p.v.Neg()} }
func (p Point) String() string { return p.v.String() }
func (p Point) Precision() int { return p.v.Prec() }
func (p Point) IsZero() bool   { return p.v.IsZero() }

func (p Point) Pow(o Point) Point { return Point{must(p.v.Pow(o.v))} }
func (p Point) Sqrt() Point       { return Point{must(p.v.Sqrt())} }

func (p Point) Rescale(scale int) Point { return Point{p.v.Rescale(scale)} }
func (p Point) Float64() float64        { return mustFloat64(p.v.Float64()) }

func (p Point) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("decimal", p.v.String())
	return nil
}

// internal helper
func must(decimal decimal.Decimal, err error) decimal.Decimal {
	if err != nil {
		panic(err)
	}
	return decimal
}

func mustFloat64(float float64, ok bool) float64 {
	if !ok {
		panic("unable to compute float64")
	}
	return float
}
