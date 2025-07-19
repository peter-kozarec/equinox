package risk

import "time"

type CooldownHandler func(lastTradeTime, currentTime time.Time) bool

func WithCooldown(h CooldownHandler) Option {
	return func(m *Manager) {
		if m.cooldownHandler != nil {
			panic("cooldown handler already set")
		}
		m.cooldownHandler = h
	}
}

func WithOnHourCooldown() Option {
	return WithCooldown(func(lastTradeTime, currentTime time.Time) bool {
		return currentTime.Sub(lastTradeTime).Hours() > 1
	})
}
