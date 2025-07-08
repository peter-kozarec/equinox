package fixed

import (
	"github.com/govalues/decimal"
	"github.com/peter-kozarec/equinox/pkg/utility"
)

// Point is an unsafe wrapper around decimal implementation. Caller must make sure the calculations
// are correct and will not result in an error state, otherwise it will panic
type Point struct {
	v decimal.Decimal
}

func FromInt(value int, scale int) Point {
	return Point{must(decimal.New(int64(value), scale))}
}

func FromInt64(value int64, scale int) Point {
	return Point{must(decimal.New(value, scale))}
}

func FromUint64(value uint64, scale int) Point {
	return Point{must(decimal.New(utility.U64ToI64Unsafe(value), scale))}
}

func FromFloat64(value float64) Point {
	return Point{must(decimal.NewFromFloat64(value))}
}

func (p Point) String() string           { return p.v.String() }
func (p Point) Float64() (float64, bool) { return p.v.Float64() }

func (p Point) Abs() Point { return Point{p.v.Abs()} }
func (p Point) Neg() Point { return Point{p.v.Neg()} }

func (p Point) Add(o Point) Point { return Point{must(p.v.Add(o.v))} }
func (p Point) Sub(o Point) Point { return Point{must(p.v.Sub(o.v))} }
func (p Point) Mul(o Point) Point { return Point{must(p.v.Mul(o.v))} }
func (p Point) Div(o Point) Point { return Point{must(p.v.Quo(o.v))} }

func (p Point) MulInt64(o int64) Point { return Point{must(p.v.Mul(decimal.MustNew(o, 0)))} }
func (p Point) MulInt(o int) Point     { return Point{must(p.v.Mul(decimal.MustNew(int64(o), 0)))} }
func (p Point) DivInt64(o int64) Point { return Point{must(p.v.Quo(decimal.MustNew(o, 0)))} }
func (p Point) DivInt(o int) Point     { return Point{must(p.v.Quo(decimal.MustNew(int64(o), 0)))} }

func (p Point) Eq(o Point) bool  { return p.v.Cmp(o.v) == 0 }
func (p Point) Gt(o Point) bool  { return p.v.Cmp(o.v) > 0 }
func (p Point) Lt(o Point) bool  { return p.v.Cmp(o.v) < 0 }
func (p Point) Gte(o Point) bool { return p.v.Cmp(o.v) >= 0 }
func (p Point) Lte(o Point) bool { return p.v.Cmp(o.v) <= 0 }

func (p Point) IsZero() bool            { return p.v.IsZero() }
func (p Point) Rescale(scale int) Point { return Point{p.v.Rescale(scale)} }

func (p Point) Pow(o Point) Point { return Point{must(p.v.Pow(o.v))} }
func (p Point) Sqrt() Point       { return Point{must(p.v.Sqrt())} }

func (p Point) Exp() Point { return Point{must(p.v.Exp())} }
func (p Point) Log() Point { return Point{must(p.v.Log())} }

func (p Point) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func must(v decimal.Decimal, err error) decimal.Decimal {
	if err == nil {
		// Return in the happy path
		return v
	}
	panic(err)
}
