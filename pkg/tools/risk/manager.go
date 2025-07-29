package risk

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/tools/indicators"
	"github.com/peter-kozarec/equinox/pkg/tools/risk/adjustment"
	"github.com/peter-kozarec/equinox/pkg/tools/risk/stoploss"
	"github.com/peter-kozarec/equinox/pkg/tools/risk/takeprofit"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	riskManagerComponentName = "tools.risk.manager"
)

type Option func(*Manager)

type Manager struct {
	r          *bus.Router
	instrument common.Instrument
	conf       Configuration
	sl         stoploss.StopLoss
	tp         takeprofit.TakeProfit
	adj        adjustment.DynamicAdjustment

	slAtr *indicators.Atr

	currentEquity  fixed.Point
	maxEquity      fixed.Point
	currentBalance fixed.Point
	maxBalance     fixed.Point

	openPositions   []common.Position
	closedPositions []common.Position
	pendingOrders   []common.Order

	serverTime           time.Time
	lastPositionOpenTime time.Time
	lastTick             common.Tick

	totalTrades     int
	winningTrades   int
	totalWinAmount  fixed.Point
	totalLossAmount fixed.Point

	trailingDistance fixed.Point
	trailingMove     fixed.Point
	nextTriggerPrice map[common.PositionId]fixed.Point

	tradeTimeHandler TradeTimeHandler
	cooldownHandler  CooldownHandler

	drawdownMulHandler       DrawdownMultiplierHandler
	signalStrengthMulHandler SignalStrengthMultiplierHandler
	rrrMulHandler            RRRMultiplierHandler
	kellyMulHandler          KellyMultiplierHandler
	martingaleMulHandler     MartingaleMultiplierHandler

	margins map[string]fixed.Point
}

func NewManager(r *bus.Router, instrument common.Instrument, conf Configuration, sl stoploss.StopLoss, tp takeprofit.TakeProfit, options ...Option) *Manager {
	m := &Manager{
		r:                r,
		instrument:       instrument,
		conf:             conf,
		sl:               sl,
		tp:               tp,
		nextTriggerPrice: make(map[common.PositionId]fixed.Point),
		margins:          make(map[string]fixed.Point),
		openPositions:    make([]common.Position, 0),
		closedPositions:  make([]common.Position, 0),
		pendingOrders:    make([]common.Order, 0),
	}

	for _, option := range options {
		option(m)
	}

	return m
}

func (m *Manager) SetMaxBalance(balance fixed.Point) {
	m.maxBalance = balance
}

func (m *Manager) SetBalance(balance fixed.Point) {
	m.currentBalance = balance
}

func (m *Manager) SetMaxEquity(equity fixed.Point) {
	m.maxEquity = equity
}

func (m *Manager) SetEquity(equity fixed.Point) {
	m.currentEquity = equity
}

func (m *Manager) OnTick(_ context.Context, tick common.Tick) {
	m.serverTime = tick.TimeStamp
	m.lastTick = tick
	m.checkOpenPositions(tick)
}

func (m *Manager) OnBar(_ context.Context, bar common.Bar) {
	if m.slAtr != nil {
		m.slAtr.OnBar(bar)
	}
}

func (m *Manager) OnSignal(_ context.Context, signal common.Signal) {
	rejection := m.validateSignal(signal)
	if rejection != nil {
		m.rejectSignal(signal, rejection.reason, rejection.comment)
		return
	}

	entry := signal.Entry.Rescale(m.instrument.Digits)

	tp, err := m.tp.GetInitialTakeProfit(signal)
	if err != nil {
		m.rejectSignal(signal, "take profit is not set", err.Error())
		return
	}
	tp = tp.Rescale(m.instrument.Digits)

	sl, err := m.sl.GetInitialStopLoss(signal)
	if err != nil {
		m.rejectSignal(signal, "stop loss is not set", err.Error())
		return
	}
	sl = sl.Rescale(m.instrument.Digits)

	size, comment := m.applySizeMultipliers(signal, entry, sl)
	if size.IsZero() {
		drawdown := m.calculateDrawdown()
		m.rejectSignal(signal,
			"position size is zero",
			fmt.Sprintf("drawdown: %s; %s", drawdown.String(), comment))
		return
	}

	size = m.clampPositionSize(size, entry, sl)

	if !m.hasEnoughMargin(signal.Symbol, entry, size) {
		margin := m.getMargin(signal.Symbol)
		m.rejectSignal(signal,
			"not enough margin",
			fmt.Sprintf("margin_requirement: %s%%; equity: %s; expected_size: %s", margin.String(), m.currentEquity.String(), size.String()))
		return
	}

	if !m.validateRiskLimits(signal, entry, sl, size) {
		m.rejectSignal(signal,
			"risk limit is exceeded",
			fmt.Sprintf("risk_limit: %s%%; equity: %s; expected_size: %s", m.conf.RiskOpen.String(), m.currentEquity.String(), size.String()))
		return
	}

	order, err := m.createMarketOrder(signal, tp, sl, size)
	if err != nil {
		m.rejectSignal(signal, "unable to create market order", err.Error())
		return
	}

	if err := m.r.Post(bus.OrderEvent, order); err != nil {
		m.rejectSignal(signal, "unable to post order event", err.Error())
		return
	}

	m.acceptSignal(signal, comment)
	m.pendingOrders = append(m.pendingOrders, order)
}

func (m *Manager) OnPositionOpened(_ context.Context, position common.Position) {
	m.lastPositionOpenTime = m.serverTime
	m.openPositions = append(m.openPositions, position)
}

func (m *Manager) OnPositionClosed(_ context.Context, position common.Position) {
	idx := m.findOpenPosition(position.TraceID)
	if idx == -1 {
		slog.Warn("position not found in open positions", slog.Uint64("traceId", position.TraceID))
		return
	}

	delete(m.nextTriggerPrice, position.Id)
	m.openPositions = append(m.openPositions[:idx], m.openPositions[idx+1:]...)
	m.closedPositions = append(m.closedPositions, position)
	m.updatePerformanceStats(position)
}

func (m *Manager) OnPositionUpdated(_ context.Context, position common.Position) {
	idx := m.findOpenPosition(position.TraceID)
	if idx == -1 {
		slog.Warn("position not found in open positions", slog.Uint64("traceId", position.TraceID))
		return
	}
	m.openPositions[idx] = position
}

func (m *Manager) OnOrderAccepted(_ context.Context, acceptedOrder common.OrderAccepted) {
	m.removePendingOrder(acceptedOrder.OriginalOrder.TraceID)
}

func (m *Manager) OnOrderRejected(_ context.Context, rejectedOrder common.OrderRejected) {
	m.removePendingOrder(rejectedOrder.OriginalOrder.TraceID)
}

func (m *Manager) OnEquity(_ context.Context, equity common.Equity) {
	m.currentEquity = equity.Value
	if m.maxEquity.IsZero() || m.currentEquity.Gt(m.maxEquity) {
		m.maxEquity = m.currentEquity
	}
}

func (m *Manager) OnBalance(_ context.Context, balance common.Balance) {
	m.currentBalance = balance.Value
	if m.maxBalance.IsZero() || m.currentBalance.Gt(m.maxBalance) {
		m.maxBalance = m.currentBalance
	}
}

func (m *Manager) SetPerformanceMetrics(totalTrades, winningTrades int, totalWinAmount, totalLossAmount fixed.Point) {
	m.totalTrades = totalTrades
	m.winningTrades = winningTrades
	m.totalWinAmount = totalWinAmount
	m.totalLossAmount = totalLossAmount
}

func (m *Manager) SetMargin(symbol string, margin fixed.Point) {
	m.margins[symbol] = margin
}

type signalRejection struct {
	reason  string
	comment string
}

func (m *Manager) validateSignal(_ common.Signal) *signalRejection {
	if !m.isTimeToTrade() {
		return &signalRejection{
			reason:  "it is not time to trade",
			comment: fmt.Sprintf("server time: %s", m.serverTime.String()),
		}
	}

	if m.cooldownHandler != nil && !m.lastPositionOpenTime.IsZero() {
		if !m.cooldownHandler(m.lastPositionOpenTime, m.serverTime) {
			return &signalRejection{reason: "cooldown period is not over"}
		}
	}

	if m.maxBalance.IsZero() || m.currentBalance.IsZero() {
		return &signalRejection{reason: "balance is not set"}
	}

	if m.maxEquity.IsZero() || m.currentEquity.IsZero() {
		return &signalRejection{reason: "equity is not set"}
	}

	return nil
}

func (m *Manager) applySizeMultipliers(signal common.Signal, entry, sl fixed.Point) (fixed.Point, string) {
	size := m.calculateBasePositionSize(entry, sl)
	drawdown := m.calculateDrawdown()
	comment := ""

	if m.kellyMulHandler != nil {
		multiplier := m.kellyMulHandler(m.totalTrades, m.getWinRate(), m.getAvgWinLossRatio())
		size = size.Mul(multiplier)
		comment += fmt.Sprintf("kelly: %s; ", multiplier.String())
	}

	if m.signalStrengthMulHandler != nil {
		multiplier := m.signalStrengthMulHandler(signal.Strength)
		size = size.Mul(multiplier)
		comment += fmt.Sprintf("strength: %s; ", multiplier.String())
	}

	if m.drawdownMulHandler != nil {
		multiplier := m.drawdownMulHandler(drawdown)
		size = size.Mul(multiplier)
		comment += fmt.Sprintf("drawdown: %s; ", multiplier.String())
	}

	if m.rrrMulHandler != nil {
		ratio := m.calculateRRR(entry, sl, signal.Target)
		multiplier := m.rrrMulHandler(ratio)
		size = size.Mul(multiplier)
		comment += fmt.Sprintf("rrr: %s; ", multiplier.String())
	}

	if m.martingaleMulHandler != nil {
		multiplier := m.martingaleMulHandler(m.closedPositions)
		size = size.Mul(multiplier)
		comment += fmt.Sprintf("martingale: %s; ", multiplier.String())
	}

	return size, comment
}

func (m *Manager) clampPositionSize(size, entry, sl fixed.Point) fixed.Point {
	maxSize := m.calculateMaxPositionSize(entry, sl).Rescale(2)
	minSize := m.calculateMinPositionSize(entry, sl).Rescale(2)
	size = size.Rescale(2)

	originalSize := size
	size = clamp(size, minSize, maxSize)

	if !size.Eq(originalSize) {
		slog.Debug("position size adjusted",
			slog.String("original", originalSize.String()),
			slog.String("adjusted", size.String()),
			slog.String("min", minSize.String()),
			slog.String("max", maxSize.String()))
	}

	return size
}

func (m *Manager) validateRiskLimits(signal common.Signal, entry, sl, size fixed.Point) bool {
	currentOpenRisk := m.calculateCurrentOpenRisk()
	additionalRisk := m.calculateRiskPercentage(signal.Symbol, entry, sl, size)
	totalRisk := currentOpenRisk.Add(additionalRisk)
	return !totalRisk.Gt(m.conf.RiskOpen)
}

func (m *Manager) rejectSignal(signal common.Signal, reason, comment string) {
	rejection := common.SignalRejected{
		Reason:         reason,
		Comment:        comment,
		OriginalSignal: signal,
		Source:         riskManagerComponentName,
		ExecutionID:    utility.GetExecutionID(),
		TraceID:        utility.CreateTraceID(),
		TimeStamp:      m.serverTime,
	}

	if err := m.r.Post(bus.SignalRejectionEvent, rejection); err != nil {
		slog.Error("unable to post signal rejection event",
			slog.String("reason", reason),
			slog.Any("err", err))
	}
}

func (m *Manager) acceptSignal(signal common.Signal, comment string) {
	acceptance := common.SignalAccepted{
		Comment:        comment,
		OriginalSignal: signal,
		Source:         riskManagerComponentName,
		ExecutionID:    utility.GetExecutionID(),
		TraceID:        utility.CreateTraceID(),
		TimeStamp:      m.serverTime,
	}

	if err := m.r.Post(bus.SignalAcceptanceEvent, acceptance); err != nil {
		slog.Error("unable to post signal acceptance event", slog.Any("err", err))
	}
}

func (m *Manager) checkOpenPositions(tick common.Tick) {
	for _, position := range m.openPositions {
		if m.hasOpenOrdersForPosition(position.Id) {
			continue
		}

		if err := m.checkDynamicPositionAdjustment(position, tick); err != nil {
			slog.Warn("unable to check for dynamic position adjustment",
				slog.Int64("positionId", position.Id),
				slog.Any("err", err))
		}
	}
}

func (m *Manager) checkDynamicPositionAdjustment(position common.Position, tick common.Tick) error {
	if m.adj == nil {
		return nil
	}

	newStopLoss, newTakeProfit, wasChanged := m.adj.AdjustPosition(position, tick)

	if !wasChanged {
		return nil
	}

	if newStopLoss.IsZero() || newTakeProfit.IsZero() {
		return fmt.Errorf("new stop loss or take profit is zero")
	}

	order := common.Order{
		Command:     common.OrderCommandPositionModify,
		StopLoss:    newStopLoss,
		TakeProfit:  newTakeProfit,
		PositionId:  position.Id,
		Source:      riskManagerComponentName,
		Symbol:      position.Symbol,
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   m.serverTime,
	}

	if err := m.r.Post(bus.OrderEvent, order); err != nil {
		return fmt.Errorf("unable to post order event: %w", err)
	}

	m.pendingOrders = append(m.pendingOrders, order)
	return nil
}

func (m *Manager) isTimeToTrade() bool {
	if m.serverTime.IsZero() {
		return false
	}
	if m.tradeTimeHandler != nil {
		return m.tradeTimeHandler(m.serverTime)
	}
	return true
}

func (m *Manager) calculateCurrentOpenRisk() fixed.Point {
	risk := fixed.Zero
	for _, position := range m.openPositions {
		risk = risk.Add(m.calculateRiskPercentage(position.Symbol, position.OpenPrice, position.StopLoss, position.Size))
	}
	return risk
}

func (m *Manager) calculateRiskPercentage(symbol string, entry, sl, size fixed.Point) fixed.Point {
	margin := m.getMargin(symbol)
	priceDiff := entry.Sub(sl).Abs()
	monetaryRisk := priceDiff.Mul(size).Mul(m.instrument.ContractSize)
	leverageMultiplier := fixed.FromInt(100, 0).Div(margin)
	effectiveRisk := monetaryRisk.Mul(leverageMultiplier)
	riskPercentage := effectiveRisk.Div(m.currentEquity).MulInt(100)
	return riskPercentage
}

func (m *Manager) calculateMaxPositionSize(entry, sl fixed.Point) fixed.Point {
	return m.calculatePositionSize(entry, sl, m.conf.RiskMax)
}

func (m *Manager) calculateMinPositionSize(entry, sl fixed.Point) fixed.Point {
	return m.calculatePositionSize(entry, sl, m.conf.RiskMin)
}

func (m *Manager) calculateBasePositionSize(entry, sl fixed.Point) fixed.Point {
	return m.calculatePositionSize(entry, sl, m.conf.RiskBase)
}

func (m *Manager) calculatePositionSize(entry, sl, riskPercentage fixed.Point) fixed.Point {
	priceDiff := entry.Sub(sl).Abs()
	pipDiff := priceDiff.Div(m.instrument.PipSize)
	riskAmount := m.currentEquity.Mul(riskPercentage.DivInt(100))
	pipValue := m.instrument.ContractSize.Mul(m.instrument.PipSize)
	positionSize := riskAmount.Div(pipDiff.Mul(pipValue))
	return positionSize
}

func (m *Manager) hasEnoughMargin(symbol string, entry, size fixed.Point) bool {
	margin := m.getMargin(symbol)
	positionValue := entry.Mul(size).Mul(m.instrument.ContractSize)
	requiredMargin := positionValue.Mul(margin.DivInt(100))
	return m.currentEquity.Gte(requiredMargin)
}

func (m *Manager) calculateDrawdown() fixed.Point {
	if m.maxEquity.IsZero() {
		return fixed.Zero
	}
	return fixed.One.Sub(m.currentEquity.Div(m.maxEquity)).MulInt(100)
}

func (m *Manager) calculateRRR(entry, sl, tp fixed.Point) fixed.Point {
	risk := entry.Sub(sl).Abs()
	reward := tp.Sub(entry).Abs()
	if risk.IsZero() {
		return fixed.Zero
	}
	return reward.Div(risk)
}

func (m *Manager) getWinRate() fixed.Point {
	if m.totalTrades == 0 {
		return fixed.FromFloat64(0.5)
	}
	return fixed.FromInt(m.winningTrades, 0).Div(fixed.FromInt(m.totalTrades, 0))
}

func (m *Manager) getAvgWinLossRatio() fixed.Point {
	if m.totalLossAmount.IsZero() || m.winningTrades == 0 {
		return fixed.FromFloat64(1.5)
	}

	losingTrades := m.totalTrades - m.winningTrades
	if losingTrades == 0 {
		return fixed.FromFloat64(2.0)
	}

	avgWin := m.totalWinAmount.Div(fixed.FromInt(m.winningTrades, 0))
	avgLoss := m.totalLossAmount.Div(fixed.FromInt(losingTrades, 0))

	if avgLoss.IsZero() {
		return fixed.FromFloat64(2.0)
	}
	return avgWin.Div(avgLoss)
}

func (m *Manager) updatePerformanceStats(position common.Position) {
	m.totalTrades++

	if position.GrossProfit.Gt(fixed.Zero) {
		m.winningTrades++
		m.totalWinAmount = m.totalWinAmount.Add(position.GrossProfit)
	} else {
		m.totalLossAmount = m.totalLossAmount.Add(position.GrossProfit.Abs())
	}
}

func (m *Manager) createMarketOrder(signal common.Signal, tp, sl, size fixed.Point) (common.Order, error) {
	var order common.Order
	var err error

	if signal.Entry.Gt(signal.Target) {
		order, err = m.createSellOrder(signal.Entry, tp, sl, size)
	} else {
		order, err = m.createBuyOrder(signal.Entry, tp, sl, size)
	}

	if err != nil {
		return common.Order{}, err
	}

	order.Command = common.OrderCommandPositionOpen
	order.Type = common.OrderTypeMarket
	return order, nil
}

func (m *Manager) createSellOrder(entry, tp, sl, size fixed.Point) (common.Order, error) {
	if tp.Gte(entry) && !tp.IsZero() {
		return common.Order{}, fmt.Errorf("target price must be less than entry price for sell order")
	}
	if sl.Lte(entry) {
		return common.Order{}, fmt.Errorf("stop loss must be greater than entry price for sell order")
	}

	return common.Order{
		Source:      riskManagerComponentName,
		Symbol:      m.instrument.Symbol,
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   m.serverTime,
		Side:        common.OrderSideSell,
		Price:       entry,
		TakeProfit:  tp,
		StopLoss:    sl,
		Size:        size,
	}, nil
}

func (m *Manager) createBuyOrder(entry, tp, sl, size fixed.Point) (common.Order, error) {
	if tp.Lte(entry) && !tp.IsZero() {
		return common.Order{}, fmt.Errorf("target price must be greater than entry price for buy order")
	}
	if sl.Gte(entry) {
		return common.Order{}, fmt.Errorf("stop loss must be less than entry price for buy order")
	}

	return common.Order{
		Source:      riskManagerComponentName,
		Symbol:      m.instrument.Symbol,
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   m.serverTime,
		Side:        common.OrderSideBuy,
		Price:       entry,
		TakeProfit:  tp,
		StopLoss:    sl,
		Size:        size,
	}, nil
}

func (m *Manager) findOpenPosition(traceID utility.TraceID) int {
	for idx, position := range m.openPositions {
		if position.TraceID == traceID {
			return idx
		}
	}
	return -1
}

func (m *Manager) hasOpenOrdersForPosition(positionID common.PositionId) bool {
	for _, order := range m.pendingOrders {
		if order.PositionId == positionID {
			return true
		}
	}
	return false
}

func (m *Manager) removePendingOrder(traceID utility.TraceID) {
	for idx, order := range m.pendingOrders {
		if order.TraceID == traceID {
			m.pendingOrders = append(m.pendingOrders[:idx], m.pendingOrders[idx+1:]...)
			return
		}
	}
	slog.Warn("pending order not found", slog.Uint64("traceId", traceID))
}

func (m *Manager) getMargin(symbol string) fixed.Point {
	margin, ok := m.margins[symbol]
	if !ok {
		return fixed.FromInt(100, 0)
	}
	return margin
}
