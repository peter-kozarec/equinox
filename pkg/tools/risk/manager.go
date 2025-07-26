package risk

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/tools/indicators"
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

	atr *indicators.Atr

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

	tradeTimeHandler TradeTimeHandler
	cooldownHandler  CooldownHandler

	drawdownMulHandler       DrawdownMultiplierHandler
	signalStrengthMulHandler SignalStrengthMultiplierHandler
	rrrMulHandler            RRRMultiplierHandler
	kellyMulHandler          KellyMultiplierHandler

	margins map[string]fixed.Point
}

func NewManager(r *bus.Router, instrument common.Instrument, conf Configuration, options ...Option) *Manager {
	m := &Manager{
		r:          r,
		instrument: instrument,
		conf:       conf,
		atr:        indicators.NewAtr(conf.AtrPeriod),
		margins:    make(map[string]fixed.Point),
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
	m.checkOpenPositions()
}

func (m *Manager) OnBar(_ context.Context, bar common.Bar) {
	m.atr.OnBar(bar)
}

func (m *Manager) OnSignal(_ context.Context, signal common.Signal) {
	if !m.atr.Ready() {
		if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
			Reason:         "atr is not ready",
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal rejection event", slog.Any("err", err))
		}
		return
	}

	if !m.isTimeToTrade() {
		if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
			Reason:         "it is not time to trade",
			Comment:        fmt.Sprintf("server time: %s", m.serverTime.String()),
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal rejection event", slog.Any("err", err))
		}
		return
	}

	if m.cooldownHandler != nil && !m.lastPositionOpenTime.IsZero() {
		if !m.cooldownHandler(m.lastPositionOpenTime, m.serverTime) {
			if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
				Reason:         "cooldown period is not over",
				OriginalSignal: signal,
				Source:         riskManagerComponentName,
				ExecutionID:    utility.GetExecutionID(),
				TraceID:        utility.CreateTraceID(),
				TimeStamp:      m.serverTime,
			}); err != nil {
				slog.Warn("unable to post signal rejection event", slog.Any("err", err))
			}
			return
		}
	}

	if m.maxBalance.IsZero() || m.currentBalance.IsZero() {
		if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
			Reason:         "balance is not set",
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal rejection event", slog.Any("err", err))
		}
		return
	}

	if m.maxEquity.IsZero() || m.currentEquity.IsZero() {
		if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
			Reason:         "equity is not set",
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal rejection event", slog.Any("err", err))
		}
		return
	}

	atrValue := m.atr.Value()

	if signal.Entry.Sub(signal.Target).Abs().Lt(atrValue.Mul(m.conf.AtrTakeProfitMinMultiplier)) {
		if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
			Reason:         "target is too close to entry",
			Comment:        fmt.Sprintf("atr: %s; adjustedAtr: %s", atrValue.String(), atrValue.Mul(m.conf.AtrTakeProfitMinMultiplier).String()),
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal rejection event", slog.Any("err", err))
		}
		return
	}

	tp := signal.Target.Rescale(m.instrument.Digits)
	entry := signal.Entry.Rescale(m.instrument.Digits)
	sl := m.calculateStopLoss(entry, tp, atrValue).Rescale(m.instrument.Digits)
	size, err := m.calculateBasePositionSize(signal.Symbol, entry, sl)
	if err != nil {
		if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
			Reason:         "unable to calculate base position size",
			Comment:        fmt.Sprintf("error: %s", err.Error()),
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal rejection event", slog.Any("err", err))
		}
		return
	}

	drawdown := fixed.One.Sub(m.currentEquity.Div(m.maxEquity)).MulInt(100)

	comment := ""
	if m.kellyMulHandler != nil {
		multiplier := m.kellyMulHandler(m.totalTrades, m.getWinRate(), m.getAvgWinLossRatio())
		size = size.Mul(multiplier)
		comment += fmt.Sprintf("kelly multiplier: %s;", multiplier.String())
	}
	if m.signalStrengthMulHandler != nil {
		multiplier := m.signalStrengthMulHandler(signal.Strength)
		size = size.Mul(multiplier)
		comment += fmt.Sprintf("signal strength multiplier: %s;", multiplier.String())
	}
	if m.drawdownMulHandler != nil {
		multiplier := m.drawdownMulHandler(drawdown)
		size = size.Mul(multiplier)
		comment += fmt.Sprintf("drawdown multiplier: %s;", multiplier.String())
	}
	if m.rrrMulHandler != nil {
		risk := entry.Sub(sl).Abs()
		reward := tp.Sub(entry).Abs()
		ratio := reward.Div(risk)
		multiplier := m.rrrMulHandler(ratio)
		size = size.Mul(multiplier)
		comment += fmt.Sprintf("rrr multiplier: %s;rrr risk %s; rrr reward%s; rrr ratio %s;", multiplier.String(), risk.String(), reward.String(), ratio.String())
	}

	if size.IsZero() {
		if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
			Reason:         "position size is zero",
			Comment:        fmt.Sprintf("drawdown: %s; %s", drawdown.String(), comment),
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal rejection event", slog.Any("err", err))
		}
		return
	} else {
		if err := m.r.Post(bus.SignalAcceptanceEvent, common.SignalAccepted{
			Comment:        comment,
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal acceptance event", slog.Any("err", err))
		}
	}

	maxSize, err := m.calculateMaxPositionSize(signal.Symbol, entry, sl)
	if err != nil {
		if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
			Reason:         "unable to calculate max position size",
			Comment:        fmt.Sprintf("error: %s", err.Error()),
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal rejection event", slog.Any("err", err))
		}
		return
	}
	minSize, err := m.calculateMinPositionSize(signal.Symbol, entry, sl)
	if err != nil {
		if err := m.r.Post(bus.SignalRejectionEvent, common.SignalRejected{
			Reason:         "unable to calculate min position size",
			Comment:        fmt.Sprintf("error: %s", err.Error()),
			OriginalSignal: signal,
			Source:         riskManagerComponentName,
			ExecutionID:    utility.GetExecutionID(),
			TraceID:        utility.CreateTraceID(),
			TimeStamp:      m.serverTime,
		}); err != nil {
			slog.Warn("unable to post signal rejection event", slog.Any("err", err))
		}
		return
	}

	originalSize := size
	size = clamp(size, minSize, maxSize)

	if !size.Eq(originalSize) {
		slog.Debug("position size adjusted",
			slog.String("original", originalSize.String()),
			slog.String("adjusted", size.String()),
			slog.String("min", minSize.String()),
			slog.String("max", maxSize.String()))
	}

	size = size.Rescale(2)

	currentOpenRisk := m.calculateCurrentOpenRisk()
	additionalRisk := m.calculateRiskPercentage(signal.Symbol, entry, sl, size)
	totalRisk := currentOpenRisk.Add(additionalRisk)

	if totalRisk.Gt(m.conf.RiskOpen) {
		slog.Debug("total risk is greater than max risk percentage, signal is discarded",
			slog.String("current_open_risk", currentOpenRisk.String()),
			slog.String("additional_risk", additionalRisk.String()),
			slog.String("total_risk", totalRisk.String()),
			slog.String("max_risk_percentage", m.conf.RiskMax.String()))
		return
	}

	order, err := m.createMarketOrder(entry, tp, sl, size)
	if err != nil {
		slog.Warn("unable to create market order", slog.Any("err", err))
		return
	}

	if err := m.r.Post(bus.OrderEvent, order); err != nil {
		slog.Warn("unable to post order event", slog.Any("err", err))
		return
	}

	m.pendingOrders = append(m.pendingOrders, order)
}

func (m *Manager) OnPositionOpened(_ context.Context, position common.Position) {
	m.lastPositionOpenTime = m.serverTime
	m.openPositions = append(m.openPositions, position)
}

func (m *Manager) OnPositionClosed(_ context.Context, position common.Position) {
	found := false
	for idx := range m.openPositions {
		openPosition := &m.openPositions[idx]
		if openPosition.TraceID == position.TraceID {
			m.openPositions = append(m.openPositions[:idx], m.openPositions[idx+1:]...)
			found = true
			break
		}
	}
	if !found {
		slog.Warn("position is not open, closed position is discarded")
		return
	}

	m.closedPositions = append(m.closedPositions, position)
	m.updatePerformanceStats(position)
}

func (m *Manager) OnPositionUpdated(_ context.Context, position common.Position) {
	found := false
	for idx := range m.openPositions {
		openPosition := &m.openPositions[idx]
		if openPosition.TraceID == position.TraceID {
			*openPosition = position
			found = true
			break
		}
	}

	if !found {
		slog.Warn("position is not open, updated position is discarded")
		return
	}
}

func (m *Manager) OnOrderAccepted(_ context.Context, acceptedOrder common.OrderAccepted) {
	found := false
	for idx := range m.pendingOrders {
		pendingOrder := &m.pendingOrders[idx]
		if pendingOrder.TraceID == acceptedOrder.OriginalOrder.TraceID {
			m.pendingOrders = append(m.pendingOrders[:idx], m.pendingOrders[idx+1:]...)
			found = true
			break
		}
	}

	if !found {
		slog.Warn("order is not pending, accepted order is discarded")
		return
	}
}

func (m *Manager) OnOrderRejected(_ context.Context, rejectedOrder common.OrderRejected) {
	found := false
	for idx := range m.pendingOrders {
		pendingOrder := &m.pendingOrders[idx]
		if pendingOrder.TraceID == rejectedOrder.OriginalOrder.TraceID {
			m.pendingOrders = append(m.pendingOrders[:idx], m.pendingOrders[idx+1:]...)
			found = true
			break
		}
	}

	if !found {
		slog.Warn("order is not pending, rejected order is discarded")
		return
	}
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

func (m *Manager) checkOpenPositions() {
	for _, position := range m.openPositions {
		pendingOrderFound := false

		for _, pendingOrder := range m.pendingOrders {
			if pendingOrder.PositionId == position.Id {
				slog.Debug("position has an open orders, break even is not checked")
				pendingOrderFound = true
			}
		}

		if !pendingOrderFound {
			if err := m.checkForBreakEven(position); err != nil {
				slog.Warn("unable to check for break even", slog.Any("err", err))
			}
		}
	}
}

func (m *Manager) checkForBreakEven(position common.Position) error {
	if m.conf.BreakEvenThreshold.IsZero() || position.TakeProfit.IsZero() {
		return nil
	}

	lastTick := m.lastTick
	serverTime := m.serverTime

	var moved, takeProfitPriceDiff fixed.Point
	if position.Side == common.PositionSideLong {
		if position.StopLoss.Gte(position.OpenPrice) || position.TakeProfit.Lte(lastTick.Bid) {
			return nil
		}
		moved = lastTick.Bid.Sub(position.OpenPrice)
		takeProfitPriceDiff = position.TakeProfit.Sub(position.OpenPrice)
	} else {
		if position.StopLoss.Lte(position.OpenPrice) || position.TakeProfit.Gte(lastTick.Ask) {
			return nil
		}
		moved = position.OpenPrice.Sub(lastTick.Ask)
		takeProfitPriceDiff = position.OpenPrice.Sub(position.TakeProfit)
	}

	if moved.Lt(fixed.Zero) {
		return nil
	}

	movePercentage := moved.Div(takeProfitPriceDiff).MulInt(100)

	if movePercentage.Gte(m.conf.BreakEvenThreshold) {
		newStopLossMove := takeProfitPriceDiff.Mul(m.conf.BreakEvenMove.DivInt(100))

		var newStopLoss fixed.Point
		if position.Side == common.PositionSideLong {
			newStopLoss = position.OpenPrice.Add(newStopLossMove)
		} else {
			newStopLoss = position.OpenPrice.Sub(newStopLossMove)
		}

		order := common.Order{
			Source:      riskManagerComponentName,
			Symbol:      m.instrument.Symbol,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   serverTime,
			Command:     common.OrderCommandPositionModify,
			StopLoss:    newStopLoss,
			TakeProfit:  position.TakeProfit,
			PositionId:  position.Id,
			Comment:     fmt.Sprintf("Break even triggered at %s%% move", movePercentage.String()),
		}

		if err := m.r.Post(bus.OrderEvent, order); err != nil {
			return fmt.Errorf("unable to post order event: %v", err)
		}

		m.pendingOrders = append(m.pendingOrders, order)
	}

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
	margin, ok := m.margins[symbol]
	if !ok {
		margin = fixed.FromInt(100, 0)
	}
	priceDiff := entry.Sub(sl).Abs()
	monetaryRisk := priceDiff.Mul(size).Mul(m.instrument.ContractSize)
	leverageMultiplier := fixed.FromInt(100, 1).Div(margin)
	effectiveRisk := monetaryRisk.Mul(leverageMultiplier)
	riskPercentage := effectiveRisk.Div(m.currentEquity).MulInt(100)
	return riskPercentage
}

func (m *Manager) calculateMaxPositionSize(symbol string, entry, sl fixed.Point) (fixed.Point, error) {
	return m.calculatePositionSize(symbol, entry, sl, m.conf.RiskMax)
}

func (m *Manager) calculateMinPositionSize(symbol string, entry, sl fixed.Point) (fixed.Point, error) {
	return m.calculatePositionSize(symbol, entry, sl, m.conf.RiskMin)
}

func (m *Manager) calculateBasePositionSize(symbol string, entry, sl fixed.Point) (fixed.Point, error) {
	return m.calculatePositionSize(symbol, entry, sl, m.conf.RiskBase)
}

func (m *Manager) calculatePositionSize(symbol string, entry, sl, riskPercentage fixed.Point) (fixed.Point, error) {
	if m.hasEnoughMargin(symbol, entry, riskPercentage) {
		return fixed.Zero, fmt.Errorf("not enough margin")
	}
	priceDiff := entry.Sub(sl).Abs()
	pipDiff := priceDiff.Div(m.instrument.PipSize)
	riskAmount := m.currentEquity.Mul(riskPercentage.DivInt(100))
	pipValue := m.instrument.ContractSize.Mul(m.instrument.PipSize)
	positionSize := riskAmount.Div(pipDiff.Mul(pipValue))
	return positionSize, nil
}

func (m *Manager) hasEnoughMargin(symbol string, entry, size fixed.Point) bool {
	margin, ok := m.margins[symbol]
	if !ok {
		margin = fixed.FromInt(100, 0)
	}

	positionValue := entry.Mul(size).Mul(m.instrument.ContractSize)
	requiredMargin := positionValue.Mul(margin).DivInt(100)
	return m.currentEquity.Gte(requiredMargin)
}

func (m *Manager) calculateStopLoss(entry, target, atr fixed.Point) fixed.Point {
	if entry.Gt(target) {
		return entry.Add(atr.Mul(m.conf.AtrStopLossMultiplier))
	}
	return entry.Sub(atr.Mul(m.conf.AtrStopLossMultiplier))
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

	avgWin := m.totalWinAmount.Div(fixed.FromInt(m.winningTrades, 0))
	avgLoss := m.totalLossAmount.Div(fixed.FromInt(m.totalTrades-m.winningTrades, 0))

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

func (m *Manager) createMarketOrder(entry, tp, sl, size fixed.Point) (common.Order, error) {
	var order common.Order
	var err error

	if entry.Gt(tp) {
		order, err = m.createSellOrder(entry, tp, sl, size)
		if err != nil {
			return common.Order{}, fmt.Errorf("unable to create sell order: %v", err)
		}
	} else {
		order, err = m.createBuyOrder(entry, tp, sl, size)
		if err != nil {
			return common.Order{}, fmt.Errorf("unable to create buy order: %v", err)
		}
	}

	order.Command = common.OrderCommandPositionOpen
	order.Type = common.OrderTypeMarket
	return order, nil
}

func (m *Manager) createSellOrder(entry, tp, sl, size fixed.Point) (common.Order, error) {
	if tp.Gt(entry) {
		return common.Order{}, fmt.Errorf("target price is greater than entry price, unable to create sell order")
	}
	if sl.Lt(entry) {
		return common.Order{}, fmt.Errorf("stop loss is less than entry price, unable to create sell order")
	}

	return common.Order{
		Source:      riskManagerComponentName,
		Symbol:      m.instrument.Symbol,
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   time.Now(),
		Side:        common.OrderSideSell,
		Price:       entry,
		TakeProfit:  tp,
		StopLoss:    sl,
		Size:        size,
	}, nil
}

func (m *Manager) createBuyOrder(entry, tp, sl, size fixed.Point) (common.Order, error) {
	if tp.Lt(entry) {
		return common.Order{}, fmt.Errorf("target price is less than entry price, unable to create buy order")
	}
	if sl.Gt(entry) {
		return common.Order{}, fmt.Errorf("stop loss is greater than entry price, unable to create buy order")
	}

	return common.Order{
		Source:      riskManagerComponentName,
		Symbol:      m.instrument.Symbol,
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   time.Now(),
		Side:        common.OrderSideBuy,
		Price:       entry,
		TakeProfit:  tp,
		StopLoss:    sl,
		Size:        size,
	}, nil
}
