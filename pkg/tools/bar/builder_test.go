package bar

import (
	"context"
	"testing"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBarBuilder_construct(t *testing.T) {
	tests := []struct {
		name      string
		buildMode BuildMode
		priceMode PriceMode
		ticks     []common.Tick
		expected  common.Bar
	}{
		{
			name:      "Single tick creates new bar",
			buildMode: BuildModeTickBased,
			priceMode: PriceModeMid,
			ticks: []common.Tick{
				createTick(time.Now(), 100.0, 99.0, 10.0, 10.0),
			},
			expected: common.Bar{
				Open:   fixed.FromFloat64(99.5),
				High:   fixed.FromFloat64(99.5),
				Low:    fixed.FromFloat64(99.5),
				Close:  fixed.FromFloat64(99.5),
				Volume: fixed.FromFloat64(20.0),
			},
		},
		{
			name:      "Multiple ticks update high/low/close",
			buildMode: BuildModeTickBased,
			priceMode: PriceModeAsk,
			ticks: []common.Tick{
				createTick(time.Now(), 100.0, 99.0, 10.0, 10.0),
				createTick(time.Now().Add(time.Second), 102.0, 101.0, 5.0, 5.0),
				createTick(time.Now().Add(2*time.Second), 98.0, 97.0, 15.0, 15.0),
			},
			expected: common.Bar{
				Open:   fixed.FromFloat64(100.0),
				High:   fixed.FromFloat64(102.0),
				Low:    fixed.FromFloat64(98.0),
				Close:  fixed.FromFloat64(98.0),
				Volume: fixed.FromFloat64(60.0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := bus.NewRouter(1024)
			builder := NewBuilder(router, tt.buildMode, tt.priceMode, false)
			builder.Build("EURUSD", common.BarPeriodM1)

			for _, tick := range tt.ticks {
				builder.construct("EURUSD", common.BarPeriodM1, tick)
			}

			require.Len(t, builder.barsInConstruction, 1)
			bar := builder.barsInConstruction[0]

			assert.Equal(t, tt.expected.Open, bar.Open)
			assert.Equal(t, tt.expected.High, bar.High)
			assert.Equal(t, tt.expected.Low, bar.Low)
			assert.Equal(t, tt.expected.Close, bar.Close)
			assert.Equal(t, tt.expected.Volume, bar.Volume)
		})
	}
}

func TestBarBuilder_flush(t *testing.T) {
	tests := []struct {
		name        string
		setupBars   []common.Bar
		flushSymbol string
		flushPeriod common.BarPeriod
	}{
		{
			name: "Flush existing bar successfully",
			setupBars: []common.Bar{
				{Symbol: "EURUSD", Period: common.BarPeriodM1},
			},
			flushSymbol: "EURUSD",
			flushPeriod: common.BarPeriodM1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := bus.NewRouter(1024)
			builder := NewBuilder(router, BuildModeTimeBased, PriceModeMid, false)
			builder.barsInConstruction = tt.setupBars

			err := builder.flush(tt.flushSymbol, tt.flushPeriod)
			assert.NoError(t, err)
			assert.Len(t, builder.barsInConstruction, 0)
		})
	}
}

func TestBarBuilder_getPrice(t *testing.T) {
	tests := []struct {
		name      string
		priceMode PriceMode
		tick      common.Tick
		expected  fixed.Point
	}{
		{
			name:      "Ask price mode",
			priceMode: PriceModeAsk,
			tick:      createTick(time.Now(), 100.0, 99.0, 10.0, 10.0),
			expected:  fixed.FromFloat64(100.0),
		},
		{
			name:      "Bid price mode",
			priceMode: PriceModeBid,
			tick:      createTick(time.Now(), 100.0, 99.0, 10.0, 10.0),
			expected:  fixed.FromFloat64(99.0),
		},
		{
			name:      "Mid price mode",
			priceMode: PriceModeMid,
			tick:      createTick(time.Now(), 100.0, 99.0, 10.0, 10.0),
			expected:  fixed.FromFloat64(99.5),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &Builder{priceMode: tt.priceMode}
			result := builder.getPrice(tt.tick)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBarBuilder_Run(t *testing.T) {
	t.Run("Time-based mode creates bars on schedule", func(t *testing.T) {
		router := bus.NewRouter(1024)
		builder := NewBuilder(router, BuildModeTimeBased, PriceModeMid, false)
		builder.Build("EURUSD", common.BarPeriodM1)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		errChan := builder.Exec(ctx)

		go func() {
			for i := 0; i < 5; i++ {
				builder.OnTick(ctx, createTick(time.Now(), 100.0+float64(i), 99.0+float64(i), 10.0, 10.0))
				time.Sleep(100 * time.Millisecond)
			}
		}()

		select {
		case err := <-errChan:
			assert.Equal(t, context.DeadlineExceeded, err)
		case <-time.After(4 * time.Second):
			t.Fatal("Test timeout")
		}
	})

	t.Run("Tick-based mode creates bars on period boundaries", func(t *testing.T) {
		router := bus.NewRouter(1024)
		builder := NewBuilder(router, BuildModeTickBased, PriceModeMid, false)
		builder.Build("EURUSD", common.BarPeriodM5)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		errChan := builder.Exec(ctx)

		now := time.Now()
		baseTime := now.Truncate(5 * time.Minute)

		builder.OnTick(ctx, createTick(baseTime.Add(2*time.Minute), 100.0, 99.0, 10.0, 10.0))
		builder.OnTick(ctx, createTick(baseTime.Add(6*time.Minute), 101.0, 100.0, 10.0, 10.0))

		time.Sleep(100 * time.Millisecond)
		cancel()

		select {
		case err := <-errChan:
			assert.Equal(t, context.Canceled, err)
		case <-time.After(1 * time.Second):
			t.Fatal("Test timeout")
		}
	})
}

func TestGetNextTickTime(t *testing.T) {
	tests := []struct {
		name     string
		period   common.BarPeriod
		from     time.Time
		expected time.Time
	}{
		{
			name:     "M1 at 12:34:56 -> 12:35:00",
			period:   common.BarPeriodM1,
			from:     time.Date(2024, 1, 1, 12, 34, 56, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 35, 0, 0, time.UTC),
		},
		{
			name:     "M5 at 12:33:00 -> 12:35:00",
			period:   common.BarPeriodM5,
			from:     time.Date(2024, 1, 1, 12, 33, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 35, 0, 0, time.UTC),
		},
		{
			name:     "M15 at 12:33:00 -> 12:45:00",
			period:   common.BarPeriodM15,
			from:     time.Date(2024, 1, 1, 12, 33, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 45, 0, 0, time.UTC),
		},
		{
			name:     "M30 at 12:33:00 -> 13:00:00",
			period:   common.BarPeriodM30,
			from:     time.Date(2024, 1, 1, 12, 33, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
		},
		{
			name:     "H1 at 12:33:00 -> 13:00:00",
			period:   common.BarPeriodH1,
			from:     time.Date(2024, 1, 1, 12, 33, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNextTickTime(tt.period, tt.from)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAlignedPeriodStart(t *testing.T) {
	tests := []struct {
		name     string
		period   common.BarPeriod
		time     time.Time
		expected time.Time
	}{
		{
			name:     "M1 at 12:34:56 -> 12:34:00",
			period:   common.BarPeriodM1,
			time:     time.Date(2024, 1, 1, 12, 34, 56, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 34, 0, 0, time.UTC),
		},
		{
			name:     "M5 at 12:37:30 -> 12:35:00",
			period:   common.BarPeriodM5,
			time:     time.Date(2024, 1, 1, 12, 37, 30, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 35, 0, 0, time.UTC),
		},
		{
			name:     "M15 at 12:37:30 -> 12:30:00",
			period:   common.BarPeriodM15,
			time:     time.Date(2024, 1, 1, 12, 37, 30, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			name:     "M30 at 12:37:30 -> 12:30:00",
			period:   common.BarPeriodM30,
			time:     time.Date(2024, 1, 1, 12, 37, 30, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			name:     "H1 at 12:37:30 -> 12:00:00",
			period:   common.BarPeriodH1,
			time:     time.Date(2024, 1, 1, 12, 37, 30, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAlignedPeriodStart(tt.period, tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkBarBuilder_construct(b *testing.B) {
	router := bus.NewRouter(1024)
	builder := NewBuilder(router, BuildModeTickBased, PriceModeMid, false)
	builder.Build("EURUSD", common.BarPeriodM1)

	tick := createTick(time.Now(), 100.0, 99.0, 10.0, 10.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.construct("EURUSD", common.BarPeriodM1, tick)
	}
}

func BenchmarkBarBuilder_flush(b *testing.B) {
	router := bus.NewRouter(b.N)
	builder := NewBuilder(router, BuildModeTimeBased, PriceModeMid, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.barsInConstruction = []common.Bar{
			{Symbol: "EURUSD", Period: common.BarPeriodM1},
		}

		err := builder.flush("EURUSD", common.BarPeriodM1)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkBarBuilder_getPrice(b *testing.B) {
	priceModes := []PriceMode{PriceModeAsk, PriceModeBid, PriceModeMid}
	tick := createTick(time.Now(), 100.0, 99.0, 10.0, 10.0)

	priceModeToStrings := make(map[PriceMode]string)
	priceModeToStrings[PriceModeAsk] = "Ask"
	priceModeToStrings[PriceModeBid] = "Bid"
	priceModeToStrings[PriceModeMid] = "Mid"

	for _, mode := range priceModes {
		b.Run(priceModeToStrings[mode], func(b *testing.B) {
			builder := &Builder{priceMode: mode}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = builder.getPrice(tick)
			}
		})
	}
}

func BenchmarkGetNextTickTime(b *testing.B) {
	periods := []common.BarPeriod{
		common.BarPeriodM1,
		common.BarPeriodM5,
		common.BarPeriodM15,
		common.BarPeriodM30,
		common.BarPeriodH1,
	}

	now := time.Now()

	barPeriodToStrings := make(map[common.BarPeriod]string)
	barPeriodToStrings[common.BarPeriodM1] = "M1"
	barPeriodToStrings[common.BarPeriodM5] = "M5"
	barPeriodToStrings[common.BarPeriodM15] = "M15"
	barPeriodToStrings[common.BarPeriodM30] = "M30"
	barPeriodToStrings[common.BarPeriodH1] = "H1"

	for _, period := range periods {
		b.Run(barPeriodToStrings[period], func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = getNextTickTime(period, now)
			}
		})
	}
}

func BenchmarkBarBuilder_MultipleSymbolsSeq(b *testing.B) {
	router := bus.NewRouter(1024)
	builder := NewBuilder(router, BuildModeTickBased, PriceModeMid, true)

	symbols := []string{"EURUSD", "GBPUSD", "USDJPY", "AUDUSD", "NZDUSD"}
	periods := []common.BarPeriod{common.BarPeriodM1, common.BarPeriodM5, common.BarPeriodM15}

	for _, symbol := range symbols {
		for _, period := range periods {
			builder.Build(symbol, period)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tick := createTick(time.Now(), 100.0, 99.0, 10.0, 10.0)
	router.Exec(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		symbol := symbols[i%len(symbols)]
		tick.Symbol = symbol
		tick.TimeStamp = tick.TimeStamp.Add(time.Second)
		builder.OnTick(ctx, tick)
	}
}

func createTick(timestamp time.Time, ask, bid, askVol, bidVol float64) common.Tick {
	return common.Tick{
		TimeStamp: timestamp,
		Ask:       fixed.FromFloat64(ask),
		Bid:       fixed.FromFloat64(bid),
		AskVolume: fixed.FromFloat64(askVol),
		BidVolume: fixed.FromFloat64(bidVol),
	}
}
