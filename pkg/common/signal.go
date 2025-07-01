package common

import (
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type Signal struct {
	Source    string // Signal source identifier
	Entry     fixed.Point
	Target    fixed.Point
	Strength  uint8         // Signal strength (0-100)
	TimeFrame time.Duration // Bar timeframe (0 for tick)
	Comment   string        // Additional comment about the signal
}

func (s Signal) Fields() []zap.Field {
	return []zap.Field{
		zap.String("source", s.Source),
		zap.String("target", s.Target.String()),
		zap.Uint8("strength", s.Strength),
		zap.Duration("timeframe", s.TimeFrame),
		zap.String("comment", s.Comment),
	}
}
