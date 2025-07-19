package risk

import "time"

type TradeTimeHandler func(time.Time) bool

func WithTimeHandler(h TradeTimeHandler) Option {
	return func(m *Manager) {
		if m.tradeTimeHandler != nil {
			panic("trade time handler already set")
		}
		m.tradeTimeHandler = h
	}
}

func WithEurUsdTradeTime() Option {
	return WithTimeHandler(func(t time.Time) bool {
		weekDay := t.Weekday()

		if weekDay == time.Saturday || weekDay == time.Sunday {
			return false
		}

		if weekDay == time.Monday && t.Hour() < 10 {
			return false
		}

		if weekDay == time.Friday && t.Hour() > 16 {
			return false
		}

		if t.Hour() < 8 || t.Hour() > 18 {
			return false
		}

		return true
	})
}
