package utility

import (
	"github.com/govalues/decimal"
	"go.uber.org/zap/zapcore"
)

var (
	ZeroFixed        = MustNewFixed(0, 0)
	TenThousandFixed = MustNewFixed(10000, 0)
	Sqrt252          = MustNewFixed(1587450786638754, 14)
)

type Fixed struct {
	d decimal.Decimal
}

func (f Fixed) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("decimal", f.d.String())
	return nil
}

// internal helper
func must(d decimal.Decimal, err error) decimal.Decimal {
	if err != nil {
		panic(err)
	}
	return d
}

func NewFixedFromInt(value int64, precision int) Fixed {
	return Fixed{must(decimal.New(value, precision))}
}

func NewFixedFromUInt(value uint64, precision int32) Fixed {
	return Fixed{must(decimal.New(int64(value), int(precision)))}
}

func MustNewFixed(value int64, precision int) Fixed {
	return Fixed{decimal.MustNew(value, precision)}
}

func (f Fixed) Add(o Fixed) Fixed { return Fixed{must(f.d.Add(o.d))} }
func (f Fixed) Sub(o Fixed) Fixed { return Fixed{must(f.d.Sub(o.d))} }
func (f Fixed) Mul(o Fixed) Fixed { return Fixed{must(f.d.Mul(o.d))} }
func (f Fixed) Div(o Fixed) Fixed { return Fixed{must(f.d.Quo(o.d))} }

func (f Fixed) AddInt64(v int64) Fixed { return f.Add(NewFixedFromInt(v, 0)) }
func (f Fixed) AddInt(v int) Fixed     { return f.Add(NewFixedFromInt(int64(v), 0)) }
func (f Fixed) SubInt64(v int64) Fixed { return f.Sub(NewFixedFromInt(v, 0)) }
func (f Fixed) SubInt(v int) Fixed     { return f.Sub(NewFixedFromInt(int64(v), 0)) }
func (f Fixed) MulInt64(v int64) Fixed { return f.Mul(NewFixedFromInt(v, 0)) }
func (f Fixed) MulInt(v int) Fixed     { return f.Mul(NewFixedFromInt(int64(v), 0)) }
func (f Fixed) DivInt64(v int64) Fixed { return f.Div(NewFixedFromInt(v, 0)) }
func (f Fixed) DivInt(v int) Fixed     { return f.Div(NewFixedFromInt(int64(v), 0)) }

func (f Fixed) Eq(o Fixed) bool  { return f.d.Cmp(o.d) == 0 }
func (f Fixed) Gt(o Fixed) bool  { return f.d.Cmp(o.d) > 0 }
func (f Fixed) Lt(o Fixed) bool  { return f.d.Cmp(o.d) < 0 }
func (f Fixed) Gte(o Fixed) bool { return f.d.Cmp(o.d) >= 0 }
func (f Fixed) Lte(o Fixed) bool { return f.d.Cmp(o.d) <= 0 }

func (f Fixed) Abs() Fixed     { return Fixed{f.d.Abs()} }
func (f Fixed) Neg() Fixed     { return Fixed{f.d.Neg()} }
func (f Fixed) String() string { return f.d.String() }
func (f Fixed) Precision() int { return f.d.Prec() }
func (f Fixed) IsZero() bool   { return f.d.IsZero() }

func (f Fixed) Pow(o Fixed) Fixed { return Fixed{must(f.d.Pow(o.d))} }
func (f Fixed) Sqrt() Fixed       { return Fixed{must(f.d.Sqrt())} }

func (f Fixed) Rescale(scale int) Fixed { return Fixed{f.d.Rescale(scale)} }
