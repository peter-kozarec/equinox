package risk

import (
	"context"
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/tools/indicators"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"log/slog"
	"time"
)

type Manager struct {
	r             *bus.Router
	instrument    common.Instrument
	configuration Configuration

	atr *indicators.Atr

	currentEquity  common.Equity
	maxEquity      common.Equity
	currentBalance common.Balance
	maxBalance     common.Balance

	openPositions   []common.Position
	closedPositions []common.Position
	pendingOrders   []common.Order

	serverTime time.Time
	lastTick   common.Tick

	totalTrades     int
	winningTrades   int
	totalWinAmount  fixed.Point
	totalLossAmount fixed.Point
}

func NewManager(r *bus.Router, instrument common.Instrument, configuration Configuration) *Manager {
	if err := validateConfiguration(configuration); err != nil {
		slog.Error("invalid configuration", slog.Any("err", err))
	}

	return &Manager{
		r:             r,
		instrument:    instrument,
		configuration: configuration,
		atr:           indicators.NewAtr(configuration.AtrPeriod),
	}
}

func validateConfiguration(cfg Configuration) error {
	if cfg.MaxRiskPercentage.Lte(fixed.Zero) || cfg.MaxRiskPercentage.Gt(fixed.FromInt(100, 0)) {
		return fmt.Errorf("MaxRiskPercentage must be between 0 and 100")
	}
	if cfg.MinRiskPercentage.Lte(fixed.Zero) || cfg.MinRiskPercentage.Gt(cfg.MaxRiskPercentage) {
		return fmt.Errorf("MinRiskPercentage must be between 0 and MaxRiskPercentage")
	}
	if cfg.BaseRiskPercentage.Lt(cfg.MinRiskPercentage) || cfg.BaseRiskPercentage.Gt(cfg.MaxRiskPercentage) {
		return fmt.Errorf("BaseRiskPercentage must be between MinRiskPercentage and MaxRiskPercentage")
	}
	if cfg.MaxOpenRiskPercentage.Lte(fixed.Zero) || cfg.MaxOpenRiskPercentage.Gt(fixed.FromInt(100, 0)) {
		return fmt.Errorf("MaxOpenRiskPercentage must be between 0 and 100")
	}
	if cfg.AtrPeriod <= 0 {
		return fmt.Errorf("AtrPeriod must be positive")
	}
	if cfg.SlAtrMultiplier.Lte(fixed.Zero) {
		return fmt.Errorf("SlAtrMultiplier must be positive")
	}
	return nil
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
		slog.Warn("atr is not ready, signal is discarded")
		return
	}

	if !m.isTimeToTrade() {
		slog.Info("it is not a time to trade, signal is discarded")
		return
	}

	maxBalanceIsZero := m.maxBalance.TimeStamp.IsZero()
	currentEquityIsZero := m.currentEquity.TimeStamp.IsZero()

	if maxBalanceIsZero {
		slog.Warn("max balance is not set, signal is discarded")
		return
	}

	if currentEquityIsZero {
		slog.Warn("current equity is not set, signal is discarded")
		return
	}

	atrValue := m.atr.Value()

	// Validate ATR is reasonable (e.g., not more than 10% of price)
	if atrValue.Gt(signal.Entry.Mul(fixed.FromFloat64(0.1))) {
		slog.Warn("ATR value seems too high relative to price",
			slog.String("atr", atrValue.String()),
			slog.String("entry", signal.Entry.String()))
		return
	}

	if signal.Entry.Sub(signal.Target).Abs().Lt(atrValue) {
		slog.Warn("signal is too close to target, signal is discarded")
		return
	}

	if err := m.assertPositionCanBeOpened(); err != nil {
		slog.Info("unable to execute signal", slog.Any("err", err))
		return
	}

	tp := signal.Target.Rescale(m.instrument.Digits)
	entry := signal.Entry.Rescale(m.instrument.Digits)
	sl := m.calculateStopLoss(entry, tp, atrValue).Rescale(m.instrument.Digits)
	size := m.calculateBasePositionSize(entry, sl)

	drawdown := fixed.One.Sub(m.currentEquity.Value.Div(m.maxEquity.Value)).MulInt(100)

	if m.totalTrades >= 20 { // Make 20 configurable
		winRate := m.getWinRate()
		avgWinLoss := m.getAvgWinLossRatio()
		size = withKellyCriterionCalcSize(size, winRate, avgWinLoss, m.configuration.BaseRiskPercentage)
	}

	size = withSignalStrengthCalcSize(size, signal.Strength)
	size = withDrawdownCalcSize(size, drawdown)
	size = withRiskRewardRatioCalcSize(size, entry, sl, tp)

	if size.IsZero() {
		slog.Info("calculated size is zero, signal is discarded",
			slog.Uint64("signal_strength", uint64(signal.Strength)),
			slog.String("drawdown", drawdown.String()))
		return
	}

	maxSize := m.calculateMaxPositionSize(entry, sl)
	minSize := m.calculateMinPositionSize(entry, sl)

	originalSize := size
	size = clamp(size, minSize, maxSize)

	size = size.Rescale(2)

	if !size.Eq(originalSize) {
		slog.Debug("position size adjusted",
			slog.String("original", originalSize.String()),
			slog.String("adjusted", size.String()),
			slog.String("min", minSize.String()),
			slog.String("max", maxSize.String()))
	}

	currentOpenRisk := m.calculateCurrentOpenRisk()
	additionalRisk := m.calculateRiskPercentage(entry, sl, size)
	totalRisk := currentOpenRisk.Add(additionalRisk)

	if totalRisk.Gt(m.configuration.MaxOpenRiskPercentage) {
		slog.Info("total risk is greater than max risk percentage, signal is discarded",
			slog.String("current_open_risk", currentOpenRisk.String()),
			slog.String("additional_risk", additionalRisk.String()),
			slog.String("total_risk", totalRisk.String()),
			slog.String("max_risk_percentage", m.configuration.MaxOpenRiskPercentage.String()))
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
	m.currentEquity = equity

	if m.maxEquity.TimeStamp.IsZero() || m.currentEquity.Value.Gt(m.maxEquity.Value) {
		m.maxEquity = m.currentEquity
	}
}

func (m *Manager) OnBalance(_ context.Context, balance common.Balance) {
	m.currentBalance = balance

	if m.maxBalance.TimeStamp.IsZero() || m.currentBalance.Value.Gt(m.maxBalance.Value) {
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
		if err := m.checkForBreakEven(position); err != nil {
			slog.Warn("unable to check for break even", slog.Any("err", err))
		}
	}
}

func (m *Manager) checkForBreakEven(position common.Position) error {
	if m.configuration.BreakEvenThresholdPercentage.IsZero() || position.TakeProfit.IsZero() {
		return nil
	}

	lastTick := m.lastTick
	serverTime := m.serverTime

	var moved, takeProfitPriceDiff fixed.Point
	if position.Side == common.PositionSideLong {
		if position.StopLoss.Gte(position.OpenPrice) {
			// Stop loss is already at or above the open price
			return nil
		}
		moved = lastTick.Bid.Sub(position.OpenPrice)
		takeProfitPriceDiff = position.TakeProfit.Sub(position.OpenPrice)
	} else {
		if position.StopLoss.Lte(position.OpenPrice) {
			// Stop loss is already at or below open price
			return nil
		}
		moved = position.OpenPrice.Sub(lastTick.Ask)
		takeProfitPriceDiff = position.OpenPrice.Sub(position.TakeProfit)
	}

	if moved.Lt(fixed.Zero) {
		// Price hasn't moved in a favorable direction
		return nil
	}

	movePercentage := moved.Div(takeProfitPriceDiff).MulInt(100)

	if movePercentage.Gte(m.configuration.BreakEvenThresholdPercentage) {
		newStopLossMove := takeProfitPriceDiff.Mul(m.configuration.BreakEvenMovePercentage.DivInt(100))

		var newStopLoss fixed.Point
		if position.Side == common.PositionSideLong {
			newStopLoss = position.OpenPrice.Add(newStopLossMove)
		} else {
			newStopLoss = position.OpenPrice.Sub(newStopLossMove)
		}

		order := common.Order{
			Source:      "risk-manager",
			Symbol:      m.instrument.Symbol,
			ExecutionId: utility.ExecutionID{},
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   serverTime,
			Command:     common.OrderCommandPositionClose,
			StopLoss:    newStopLoss,
			TakeProfit:  position.TakeProfit,
			PositionId:  position.Id,
			Comment:     fmt.Sprintf("Break even triggered at %s move", movePercentage.String()),
		}

		if err := m.r.Post(bus.OrderEvent, order); err != nil {
			return fmt.Errorf("unable to post order event: %v", err)
		}
	}

	return nil
}

func (m *Manager) isTimeToTrade() bool {
	serverTime := m.serverTime

	if serverTime.IsZero() {
		return false
	}

	weekDay := serverTime.Weekday()

	// No trading on weekends
	if weekDay == time.Saturday || weekDay == time.Sunday {
		return false
	}

	// No trading Monday before 10:00
	if weekDay == time.Monday && serverTime.Hour() < 10 {
		return false
	}

	// No trading Friday after 16:00
	if weekDay == time.Friday && serverTime.Hour() > 16 {
		return false
	}

	// Only trade between 8:00 and 18:00
	if serverTime.Hour() < 8 || serverTime.Hour() > 18 {
		return false
	}

	return true
}

func (m *Manager) calculateCurrentOpenRisk() fixed.Point {

	risk := fixed.Zero
	for _, position := range m.openPositions {
		risk = risk.Add(m.calculateRiskPercentage(position.OpenPrice, position.StopLoss, position.Size))
	}

	return risk
}

// calculateRiskPercentage returns the risk as a percentage of equity
func (m *Manager) calculateRiskPercentage(entry, sl, size fixed.Point) fixed.Point {
	equity := m.currentEquity.Value

	priceDiff := entry.Sub(sl).Abs()
	// Calculate monetary risk: price difference * size * contract size * pip size
	monetaryRisk := priceDiff.Mul(size).Mul(m.instrument.ContractSize)

	// Convert to percentage of equity
	riskPercentage := monetaryRisk.Div(equity).MulInt(100)
	return riskPercentage
}

func (m *Manager) calculateMaxPositionSize(entry, sl fixed.Point) fixed.Point {
	return m.calculatePositionSize(entry, sl, m.configuration.MaxRiskPercentage)
}

func (m *Manager) calculateMinPositionSize(entry, sl fixed.Point) fixed.Point {
	return m.calculatePositionSize(entry, sl, m.configuration.MinRiskPercentage)
}

func (m *Manager) calculateBasePositionSize(entry, sl fixed.Point) fixed.Point {
	return m.calculatePositionSize(entry, sl, m.configuration.BaseRiskPercentage)
}

// calculatePositionSize calculates position size based on risk percentage
func (m *Manager) calculatePositionSize(entry, sl, riskPercentage fixed.Point) fixed.Point {
	equity := m.currentEquity.Value

	// Calculate pip difference
	priceDiff := entry.Sub(sl).Abs()
	pipDiff := priceDiff.Div(m.instrument.PipSize)

	// Calculate risk amount in account currency
	riskAmount := equity.Mul(riskPercentage.DivInt(100))

	// Calculate position size
	// Formula: Position Size = Risk Amount / (Pip Difference * Pip Value)
	// Where Pip Value = Contract Size * Pip Size
	pipValue := m.instrument.ContractSize.Mul(m.instrument.PipSize)
	positionSize := riskAmount.Div(pipDiff.Mul(pipValue))
	return positionSize
}

func (m *Manager) calculateStopLoss(entry, target, atr fixed.Point) fixed.Point {
	if entry.Gt(target) {
		// Sell signal
		return entry.Add(atr.Mul(m.configuration.SlAtrMultiplier))
	}
	// Buy signal
	return entry.Sub(atr.Mul(m.configuration.SlAtrMultiplier))
}

func (m *Manager) assertPositionCanBeOpened() error {
	pendingCount := len(m.pendingOrders)

	if pendingCount > 0 {
		return fmt.Errorf("pending orders exist (%d), unable to open position", pendingCount)
	}
	return nil
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
		Source:      "risk-manager",
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
		Source:      "risk-manager",
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
