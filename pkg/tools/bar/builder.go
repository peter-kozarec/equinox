package bar

import (
	"context"
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"log/slog"
	"time"
)

type BuildMode int
type PriceMode int

const (
	BuildModeTimeBased BuildMode = iota
	BuildModeTickBased
)

const (
	PriceModeAsk PriceMode = iota
	PriceModeBid
	PriceModeMid
)

type Builder struct {
	router             *bus.Router
	buildMode          BuildMode
	priceMode          PriceMode
	seqMode            bool
	tickChan           chan common.Tick
	barsInConstruction []common.Bar

	stf []struct {
		symbol string
		period common.BarPeriod
	}
}

func NewBuilder(router *bus.Router, buildMode BuildMode, priceMode PriceMode, seqMode bool) *Builder {
	return &Builder{
		router:    router,
		buildMode: buildMode,
		priceMode: priceMode,
		seqMode:   seqMode,
	}
}

func (b *Builder) OnTick(ctx context.Context, tick common.Tick) {
	if b.seqMode {
		for _, stf := range b.stf {
			b.construct(stf.symbol, stf.period, tick)
		}
		return
	}

	select {
	case <-ctx.Done():
	case b.tickChan <- tick:
	default:
		slog.Warn("tick channel is full")
	}
}

func (b *Builder) Build(symbol string, period common.BarPeriod) {
	b.stf = append(b.stf, struct {
		symbol string
		period common.BarPeriod
	}{symbol, period})
}

func (b *Builder) Exec(ctx context.Context) <-chan error {
	if b.seqMode {
		panic("seq mode is not supported to run in parallel")
	}

	errChan := make(chan error, 1)
	b.tickChan = make(chan common.Tick, 100)

	go func() {
		defer close(errChan)
		defer close(b.tickChan)

		nextTicks := make(map[common.BarPeriod]time.Time)
		if b.buildMode == BuildModeTimeBased {
			now := time.Now()
			for _, stf := range b.stf {
				nextTicks[stf.period] = getNextTickTime(stf.period, now)
			}
		}

		// Find the earliest next tick time
		getEarliestTick := func() (time.Time, bool) {
			if len(nextTicks) == 0 {
				return time.Time{}, false
			}
			earliest := time.Now().Add(24 * time.Hour)
			for _, t := range nextTicks {
				if t.Before(earliest) {
					earliest = t
				}
			}
			return earliest, true
		}

		for {
			var timer *time.Timer
			var timerChan <-chan time.Time

			if b.buildMode == BuildModeTimeBased {
				if earliest, ok := getEarliestTick(); ok {
					timer = time.NewTimer(time.Until(earliest))
					timerChan = timer.C
				}
			}

			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				errChan <- ctx.Err()
				return
			case tick, ok := <-b.tickChan:
				if !ok {
					slog.Error("tick channel is closed")
					return
				}

				for _, stf := range b.stf {
					b.construct(stf.symbol, stf.period, tick)
				}
			case <-timerChan:
				now := time.Now()
				for _, stf := range b.stf {
					if nextTime, ok := nextTicks[stf.period]; ok && !now.Before(nextTime) {
						if err := b.flush(stf.symbol, stf.period); err != nil {
							errChan <- fmt.Errorf("unable to flush bar: %w", err)
							return
						}
						nextTicks[stf.period] = getNextTickTime(stf.period, now)
					}
				}
			}
		}
	}()

	return errChan
}

func (b *Builder) construct(symbol string, period common.BarPeriod, tick common.Tick) {
	if b.buildMode == BuildModeTickBased {

		for i, bar := range b.barsInConstruction {
			if bar.Symbol == symbol && bar.Period == period {
				nextPeriodStart := getNextTickTime(period, bar.OpenTime)

				// If tick belongs to next period, flush current bar
				if !tick.TimeStamp.Before(nextPeriodStart) {
					if err := b.router.Post(bus.BarEvent, bar); err != nil {
						slog.Error("unable to post bar", "error", err)
					}
					// Remove the flushed bar
					b.barsInConstruction = append(b.barsInConstruction[:i], b.barsInConstruction[i+1:]...)
					break
				}
			}
		}
	}

	found := false
	price := b.getPrice(tick)
	volume := tick.AskVolume.Add(tick.BidVolume)

	for i := range b.barsInConstruction {
		bar := &b.barsInConstruction[i]

		if bar.Symbol == symbol && bar.Period == period {

			if price.Gt(bar.High) {
				bar.High = price
			}

			if price.Lt(bar.Low) {
				bar.Low = price
			}

			bar.Close = price
			bar.TimeStamp = tick.TimeStamp
			bar.Volume = bar.Volume.Add(volume)

			found = true
			break
		}
	}

	if !found {
		openTime := tick.TimeStamp
		if b.buildMode == BuildModeTickBased {
			openTime = getAlignedPeriodStart(period, tick.TimeStamp)
		}

		bar := common.Bar{
			Source:      "bar-builder",
			Symbol:      symbol,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			Period:      period,
			TimeStamp:   tick.TimeStamp,
			OpenTime:    openTime,
			Open:        price,
			Close:       price,
			High:        price,
			Low:         price,
			Volume:      volume,
		}

		b.barsInConstruction = append(b.barsInConstruction, bar)
	}
}

func (b *Builder) flush(symbol string, period common.BarPeriod) error {
	for idx, bar := range b.barsInConstruction {
		if bar.Symbol == symbol && bar.Period == period {
			if err := b.router.Post(bus.BarEvent, bar); err != nil {
				return fmt.Errorf("unable to post bar: %w", err)
			}
			b.barsInConstruction = append(b.barsInConstruction[:idx], b.barsInConstruction[idx+1:]...)
			return nil
		}
	}
	return nil
}

func (b *Builder) getPrice(tick common.Tick) fixed.Point {
	switch b.priceMode {
	case PriceModeAsk:
		return tick.Ask
	case PriceModeBid:
		return tick.Bid
	case PriceModeMid:
		return tick.Ask.Add(tick.Bid).DivInt(2)
	default:
		panic("invalid price mode")
	}
}

func getNextTickTime(period common.BarPeriod, from time.Time) time.Time {
	hourStart := from.Truncate(time.Hour)

	switch period {
	case common.BarPeriodM1:
		// Every minute
		next := from.Truncate(time.Minute).Add(time.Minute)
		if next.After(from) {
			return next
		}
		return next.Add(time.Minute)

	case common.BarPeriodM5:
		// :00, :05, :10, :15, :20, :25, :30, :35, :40, :45, :50, :55
		minutesSinceHour := int(from.Sub(hourStart).Minutes())
		nextMinute := ((minutesSinceHour / 5) + 1) * 5
		if nextMinute >= 60 {
			return hourStart.Add(time.Hour)
		}
		return hourStart.Add(time.Duration(nextMinute) * time.Minute)

	case common.BarPeriodM15:
		// :00, :15, :30, :45
		minutesSinceHour := int(from.Sub(hourStart).Minutes())
		nextMinute := ((minutesSinceHour / 15) + 1) * 15
		if nextMinute >= 60 {
			return hourStart.Add(time.Hour)
		}
		return hourStart.Add(time.Duration(nextMinute) * time.Minute)

	case common.BarPeriodM30:
		// :00, :30
		minutesSinceHour := int(from.Sub(hourStart).Minutes())
		if minutesSinceHour < 30 {
			return hourStart.Add(30 * time.Minute)
		}
		return hourStart.Add(time.Hour)

	case common.BarPeriodH1:
		// :00
		return hourStart.Add(time.Hour)

	default:
		panic("unsupported period")
	}
}

func getAlignedPeriodStart(period common.BarPeriod, t time.Time) time.Time {
	hourStart := t.Truncate(time.Hour)

	switch period {
	case common.BarPeriodM1:
		return t.Truncate(time.Minute)
	case common.BarPeriodM5:
		minute := t.Minute()
		alignedMinute := (minute / 5) * 5
		return hourStart.Add(time.Duration(alignedMinute) * time.Minute)
	case common.BarPeriodM15:
		minute := t.Minute()
		alignedMinute := (minute / 15) * 15
		return hourStart.Add(time.Duration(alignedMinute) * time.Minute)
	case common.BarPeriodM30:
		if t.Minute() < 30 {
			return hourStart
		}
		return hourStart.Add(30 * time.Minute)
	case common.BarPeriodH1:
		return hourStart
	default:
		return t.Truncate(time.Minute)
	}
}
