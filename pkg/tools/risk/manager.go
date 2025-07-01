package risk

import (
	"errors"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/indicators"
	"github.com/peter-kozarec/equinox/pkg/tools/position"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type volRegime int

const (
	lowVolRegime volRegime = iota
	normVolRegime
	highVolRegime
)

type Manager struct {
	logger *zap.Logger
	router *bus.Router

	positionHoldings *position.Holdings
	configuration    Configuration

	account struct {
		leverage fixed.Point

		currentBalance  fixed.Point
		currentEquity   fixed.Point
		currentDrawdown fixed.Point

		maxBalance fixed.Point
		minBalance fixed.Point
		maxEquity  fixed.Point
		minEquity  fixed.Point

		drawdownFunc DrawdownMultiplierFunc
	}

	fxSession struct {
		sydneyMultiplier        fixed.Point
		tokyoMultiplier         fixed.Point
		londonMultiplier        fixed.Point
		newYorkMultiplier       fixed.Point
		sydneyTokyoMultiplier   fixed.Point
		tokyoLondonMultiplier   fixed.Point
		londonNewYorkMultiplier fixed.Point
	}

	volatility struct {
		atr              *indicators.Atr
		lastClose        fixed.Point
		currentTrueRange fixed.Point

		lowVolMultiplier  fixed.Point
		normVolMultiplier fixed.Point
		highVolMultiplier fixed.Point
	}
}

func NewManager(logger *zap.Logger, router *bus.Router, configuration Configuration, options ...ManagerOption) *Manager {
	m := &Manager{
		logger:           logger,
		router:           router,
		positionHoldings: position.NewHoldings(),
		configuration:    configuration,
	}

	for _, option := range options {
		option(m)
	}

	m.volatility.atr = indicators.NewAtr(configuration.AtrWindowSize)

	return m
}

func (m *Manager) ProcessSignal(signal common.Signal) {

	if !m.account.currentBalance.IsSet() {
		m.logger.Warn("current balance is not set")
		return
	}

	stopLoss, err := m.calcStopLoss(signal.Entry, signal.Target)
	if err != nil {
		m.logger.Warn("unable to calculate stop loss", zap.Error(err))
		return
	}

	signalMul := m.getNormalizedSignalStrength(signal.Strength)
	sessionMul := m.getFxSessionMultiplier()
	drawdownMul := m.getDrawdownMultiplier()
	volatilityMul := m.getVolatilityMultiplier()

	risk := stopLoss.Sub(signal.Entry).Abs()
	baseSize := m.getBaseSize(risk)
	maxSize := m.getMaxSize(risk)
	size := baseSize.Mul(signalMul).Mul(sessionMul).Mul(drawdownMul).Mul(volatilityMul)

	_ = fixed.ClampPoint(size, fixed.New(1, 2), maxSize).Rescale(m.configuration.SizeScale)
}

func (m *Manager) ProcessBar(bar common.Bar) {
	if m.volatility.atr != nil {
		m.volatility.atr.OnBar(bar)

		if m.volatility.lastClose.IsSet() {
			m.volatility.currentTrueRange = calcTrueRange(m.volatility.lastClose, bar)
		}

		m.volatility.lastClose = bar.Close
	}
}

func (m *Manager) UpdateBalance(balance fixed.Point) {
	m.account.currentBalance = balance

	if m.account.maxBalance.IsSet() || m.account.currentBalance.Gt(m.account.maxBalance) {
		m.account.maxBalance = balance
	}

	if m.account.minBalance.IsSet() || m.account.currentEquity.Lt(m.account.minBalance) {
		m.account.minBalance = balance
	}
}

func (m *Manager) UpdateEquity(equity fixed.Point) {
	m.account.currentEquity = equity

	if m.account.maxEquity.IsSet() || m.account.currentEquity.Gt(m.account.maxEquity) {
		m.account.maxEquity = equity
	}

	if m.account.minEquity.IsSet() || m.account.currentEquity.Lt(m.account.minEquity) {
		m.account.minEquity = equity
	}

	if m.account.maxBalance.IsSet() {
		drawdown := fixed.One.Sub(m.account.currentEquity.Div(m.account.maxBalance)).MulInt(100)

		if m.account.currentDrawdown.IsSet() || drawdown.Gt(m.account.currentDrawdown) {
			m.account.currentDrawdown = drawdown
		}
	}
}

func (m *Manager) NewPositionOpened(position common.Position) {
	m.positionHoldings.OnPositionOpen(position)
}

func (m *Manager) NewPositionClosed(position common.Position) {
	m.positionHoldings.OnPositionClose(position)
}

func (m *Manager) PositionUpdated(position common.Position) {
	m.positionHoldings.OnPositionUpdate(position)
}

func (m *Manager) currentVolatility() volRegime {
	if !m.volatility.currentTrueRange.IsSet() || !m.volatility.atr.Ready() {
		return normVolRegime
	}

	diff := m.volatility.currentTrueRange.Sub(m.volatility.atr.Value()).Abs()
	percentDiff := diff.Div(m.volatility.atr.Value())

	if percentDiff.Lte(m.configuration.LowVolatilityAtrThreshold) {
		return lowVolRegime
	} else if percentDiff.Lte(m.configuration.HighVolatilityAtrThreshold) {
		return normVolRegime
	}

	return highVolRegime
}

func (m *Manager) getNormalizedSignalStrength(signalStrength uint8) fixed.Point {
	return fixed.FromFloat(float64(signalStrength / 100))
}

func (m *Manager) getDrawdownMultiplier() fixed.Point {
	if !m.account.currentDrawdown.IsSet() {
		return fixed.One
	}

	if m.account.drawdownFunc != nil {
		return m.account.drawdownFunc(m.account.currentDrawdown)
	}

	return fixed.One
}

func (m *Manager) getFxSessionMultiplier() fixed.Point {

	session := GetCurrentSession()

	switch session.Session {
	case SessionTokyo:
		if m.fxSession.tokyoMultiplier.IsSet() {
			return m.fxSession.tokyoMultiplier
		}
	case SessionSydney:
		if m.fxSession.sydneyMultiplier.IsSet() {
			return m.fxSession.sydneyMultiplier
		}
	case SessionLondon:
		if m.fxSession.londonMultiplier.IsSet() {
			return m.fxSession.londonMultiplier
		}
	case SessionNewYork:
		if m.fxSession.newYorkMultiplier.IsSet() {
			return m.fxSession.newYorkMultiplier
		}
	case SessionSydneyTokyo:
		if m.fxSession.sydneyTokyoMultiplier.IsSet() {
			return m.fxSession.sydneyTokyoMultiplier
		}
	case SessionTokyoLondon:
		if m.fxSession.tokyoLondonMultiplier.IsSet() {
			return m.fxSession.tokyoLondonMultiplier
		}
	case SessionLondonNewYork:
		if m.fxSession.londonNewYorkMultiplier.IsSet() {
			return m.fxSession.londonNewYorkMultiplier
		}
	default:
	}

	return fixed.One
}

func (m *Manager) getVolatilityMultiplier() fixed.Point {

	volatility := m.currentVolatility()

	switch volatility {
	case lowVolRegime:
		return m.volatility.lowVolMultiplier
	case normVolRegime:
		return m.volatility.normVolMultiplier
	case highVolRegime:
		return m.volatility.highVolMultiplier
	default:
		return fixed.One
	}
}

func (m *Manager) calcStopLoss(entry, target fixed.Point) (fixed.Point, error) {

	if m.volatility.atr == nil || !m.volatility.atr.Ready() {
		return fixed.Point{}, errors.New("unable to get atr")
	}

	slAtrMul := m.getStopLossAtrMultiplier()
	atr := m.volatility.atr.Value()

	if entry.Gt(target) {
		return entry.Add(atr.Mul(slAtrMul)), nil
	}
	return entry.Sub(atr.Mul(slAtrMul)), nil
}

func (m *Manager) getStopLossAtrMultiplier() fixed.Point {

	volatility := m.currentVolatility()

	switch volatility {
	case lowVolRegime:
		return m.configuration.LowVolatilityStopLossAtrMultiplier
	case normVolRegime:
		return m.configuration.NormalVolatilityStopLossAtrMultiplier
	case highVolRegime:
		return m.configuration.HighVolatilityStopLossAtrMultiplier
	default:
		return fixed.One
	}
}

func (m *Manager) getBaseSize(risk fixed.Point) fixed.Point {
	return fixed.One
}

func (m *Manager) getMaxSize(risk fixed.Point) fixed.Point {
	return fixed.One
}
