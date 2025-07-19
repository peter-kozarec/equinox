package bar

import (
	"context"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"log/slog"
	"time"
)

type PriceMode int

const (
	PriceModeAsk PriceMode = iota
	PriceModeBid
	PriceModeMid
)

type Option func(*Builder)

func With(symbol string, period common.BarPeriod, priceMode PriceMode) Option {
	return func(b *Builder) {
		for _, c := range b.configs {
			if c.symbol == symbol && c.period == period {
				panic("bar config already exists")
			}
		}

		b.configs = append(b.configs, struct {
			symbol string
			period common.BarPeriod
			mode   PriceMode
		}{symbol, period, priceMode})
	}
}

type Builder struct {
	router         *bus.Router
	inConstruction []common.Bar

	configs []struct {
		symbol string
		period common.BarPeriod
		mode   PriceMode
	}
}

func NewBuilder(router *bus.Router, options ...Option) *Builder {
	b := &Builder{
		router: router,
	}

	for _, option := range options {
		option(b)
	}

	return b
}

func (b *Builder) OnTick(_ context.Context, tick common.Tick) {
	for _, c := range b.configs {
		b.construct(c.symbol, c.period, c.mode, tick)
	}
}

func (b *Builder) construct(symbol string, period common.BarPeriod, mode PriceMode, tick common.Tick) {

	// Check if the tick belongs to another period, if so, close bar in construction by flushing it to the router
	for i, bar := range b.inConstruction {
		if bar.Symbol == symbol && bar.Period == period {
			nextPeriodStart := getNextTickTime(period, bar.OpenTime)
			if !tick.TimeStamp.Before(nextPeriodStart) {
				if err := b.router.Post(bus.BarEvent, bar); err != nil {
					slog.Error("unable to post bar", "error", err)
				}
				b.inConstruction = append(b.inConstruction[:i], b.inConstruction[i+1:]...)
				break
			}
		}
	}

	found := false
	price := b.getPrice(tick, mode)
	volume := tick.AskVolume.Add(tick.BidVolume)

	for i := range b.inConstruction {
		bar := &b.inConstruction[i]

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
		bar := common.Bar{
			Source:      "bar-builder",
			Symbol:      symbol,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			Period:      period,
			TimeStamp:   tick.TimeStamp,
			OpenTime:    getAlignedPeriodStart(period, tick.TimeStamp),
			Open:        price,
			Close:       price,
			High:        price,
			Low:         price,
			Volume:      volume,
		}

		b.inConstruction = append(b.inConstruction, bar)
	}
}

func (b *Builder) getPrice(tick common.Tick, mode PriceMode) fixed.Point {
	switch mode {
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

	case common.BarPeriodM10:
		// :00, :10, :20, :30, :40, :50
		minutesSinceHour := int(from.Sub(hourStart).Minutes())
		nextMinute := ((minutesSinceHour / 10) + 1) * 10
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
	case common.BarPeriodM10:
		minute := t.Minute()
		alignedMinute := (minute / 10) * 10
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
