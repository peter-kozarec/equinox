package risk

import (
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type SizeMultiplierHandler interface {
	Id() string
	CalcSizeMultiplier(common.Signal) fixed.Point
}

type Factor struct {
	HandlerId  string
	Multiplier fixed.Point
}

type SizeMultiplierStrategyHandler func(fixed.Point, common.Signal, ...SizeMultiplierHandler) (fixed.Point, fixed.Point, string, []Factor)

func WithSizeMultipliers(applicationHandler SizeMultiplierStrategyHandler, handlers ...SizeMultiplierHandler) Option {
	return func(m *Manager) {
		m.sizeMultiplierStrategyHandler = applicationHandler
		m.sizeMultiplierHandlers = handlers
	}
}

func WithChainedSizeMultipliers(handlers ...SizeMultiplierHandler) Option {
	return WithSizeMultipliers(func(baseSize fixed.Point, signal common.Signal, sizeHandlers ...SizeMultiplierHandler) (fixed.Point, fixed.Point, string, []Factor) {
		var factors []Factor
		for _, sizeHandler := range sizeHandlers {
			multiplier := sizeHandler.CalcSizeMultiplier(signal)
			baseSize = baseSize.Mul(multiplier)
			id := sizeHandler.Id()
			factors = append(factors, Factor{id, multiplier})
		}
		if len(factors) == 0 {
			return baseSize, fixed.One, "ChainedSizeMultipliers", factors
		}
		return baseSize, baseSize.DivInt(len(factors)), "ChainedSizeMultipliers", factors
	}, handlers...)
}

func WithCombinedSizeMultipliers(handlers ...SizeMultiplierHandler) Option {
	return WithSizeMultipliers(func(baseSize fixed.Point, signal common.Signal, sizeHandlers ...SizeMultiplierHandler) (fixed.Point, fixed.Point, string, []Factor) {
		finalMultiplier := fixed.One
		var factors []Factor
		for _, sizeHandler := range sizeHandlers {
			multiplier := sizeHandler.CalcSizeMultiplier(signal)
			finalMultiplier = finalMultiplier.Mul(multiplier)
			id := sizeHandler.Id()
			factors = append(factors, Factor{id, multiplier})
		}
		return baseSize.Mul(finalMultiplier), finalMultiplier, "CombinedSizeMultipliers", factors
	}, handlers...)
}

func WithSummedSizeMultipliers(handlers ...SizeMultiplierHandler) Option {
	return WithSizeMultipliers(func(baseSize fixed.Point, signal common.Signal, sizeHandlers ...SizeMultiplierHandler) (fixed.Point, fixed.Point, string, []Factor) {
		finalMultiplier := fixed.One
		var factors []Factor
		for _, sizeHandler := range sizeHandlers {
			multiplier := sizeHandler.CalcSizeMultiplier(signal)
			finalMultiplier = finalMultiplier.Add(multiplier)
			id := sizeHandler.Id()
			factors = append(factors, Factor{id, multiplier})
		}
		return baseSize.Mul(finalMultiplier), finalMultiplier, "SummedSizeMultipliers", factors
	}, handlers...)
}
