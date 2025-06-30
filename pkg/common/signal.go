package common

import (
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type Signal struct {
	Target   fixed.Point
	Strength uint8
	Reason   string
}

func (s Signal) Fields() []zap.Field {
	return []zap.Field{
		zap.String("target", s.Target.String()),
		zap.Uint8("strength", s.Strength),
		zap.String("reason", s.Reason),
	}
}
