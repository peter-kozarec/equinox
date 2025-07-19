package risk

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

type SignalStrengthMultiplierHandler func(signalStrength uint8) fixed.Point

func WithSignalStrengthMultiplier(h SignalStrengthMultiplierHandler) Option {
	return func(m *Manager) {
		if m.signalStrengthMulHandler != nil {
			panic("signal strength multiplier handler already set")
		}
		m.signalStrengthMulHandler = h
	}
}

func WithDefaultSignalStrengthMul() Option {
	const (
		highSignalStrengthThreshold   uint8 = 90
		mediumSignalStrengthThreshold uint8 = 70
		lowSignalStrengthThreshold    uint8 = 50
	)

	var (
		highSignalStrengthMultiplier   = fixed.FromFloat64(1.0)
		mediumSignalStrengthMultiplier = fixed.FromFloat64(0.7)
		lowSignalStrengthMultiplier    = fixed.FromFloat64(0.5)
		noSignalStrengthMultiplier     = fixed.FromFloat64(0.0)
	)

	return WithSignalStrengthMultiplier(func(signalStrength uint8) fixed.Point {
		if signalStrength >= highSignalStrengthThreshold {
			return highSignalStrengthMultiplier
		} else if signalStrength >= mediumSignalStrengthThreshold {
			return mediumSignalStrengthMultiplier
		} else if signalStrength >= lowSignalStrengthThreshold {
			return lowSignalStrengthMultiplier
		} else {
			return noSignalStrengthMultiplier
		}
	})
}
