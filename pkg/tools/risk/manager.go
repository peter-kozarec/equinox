package risk

import (
	"context"
	"errors"
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"log/slog"
	"strings"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/utility"
)

const (
	componentNameRiskManager = "tools.risk.manager"
)

var (
	ErrRouterIsNil  = errors.New("router is nil")
	ErrNoSymbols    = errors.New("symbol map is empty")
	ErrSlHandlerNil = errors.New("sl handler is nil")
	ErrTpHandlerNil = errors.New("tp handler is nil")
	ErrCfgInvalid   = errors.New("invalid configuration")

	errSignalValidation = errors.New("signal validation error")
)

type Manager struct {
	router            *bus.Router
	cfg               Configuration
	stopLossHandler   StopLossHandler
	takeProfitHandler TakeProfitHandler

	symbols      map[string]exchange.SymbolInfo
	rateProvider exchange.RateProvider

	adjustmentHandler             AdjustmentHandler
	sizeMultiplierStrategyHandler SizeMultiplierStrategyHandler
	sizeMultiplierHandlers        []SizeMultiplierHandler
	signalValidationHandlers      []SignalValidationHandler
	customOpenOrderHandler        CustomOpenOrderHandler

	ts      time.Time
	equity  fixed.Point
	balance fixed.Point

	tickCache     map[string]common.Tick
	openOrders    []common.Order
	openPositions []common.Position
}

func NewManager(router *bus.Router, cfg Configuration, slHandler StopLossHandler, tpHandler TakeProfitHandler, options ...Option) (*Manager, error) {
	if router == nil {
		return nil, ErrRouterIsNil
	}
	if slHandler == nil {
		return nil, ErrSlHandlerNil
	}
	if tpHandler == nil {
		return nil, ErrTpHandlerNil
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCfgInvalid, err)
	}

	m := &Manager{
		router:            router,
		cfg:               cfg,
		symbols:           make(map[string]exchange.SymbolInfo),
		stopLossHandler:   slHandler,
		takeProfitHandler: tpHandler,
		tickCache:         make(map[string]common.Tick),
	}

	for _, option := range options {
		option(m)
	}

	if len(m.symbols) == 0 {
		return nil, ErrNoSymbols
	}

	return m, nil
}

func (m *Manager) OnBalance(_ context.Context, balance common.Balance) {
	m.balance = balance.Value
}

func (m *Manager) OnEquity(_ context.Context, equity common.Equity) {
	m.equity = equity.Value
}

func (m *Manager) OnTick(_ context.Context, tick common.Tick) {
	m.ts = tick.TimeStamp
	m.tickCache[tick.Symbol] = tick
	m.checkForPositionAdjustment(tick)
}

func (m *Manager) OnPositionOpen(_ context.Context, position common.Position) {
	m.openPositions = append(m.openPositions, position)
}

func (m *Manager) OnPositionUpdate(_ context.Context, position common.Position) {
	for idx := range m.openPositions {
		openPosition := m.openPositions[idx]
		if openPosition.Id == position.Id {
			openPosition.GrossProfit = position.GrossProfit
			openPosition.NetProfit = position.NetProfit
			openPosition.Margin = position.Margin
			openPosition.TimeStamp = position.TimeStamp
			break
		}
	}
}

func (m *Manager) OnPositionClose(_ context.Context, position common.Position) {
	for idx := range m.openPositions {
		openPosition := &m.openPositions[idx]
		if openPosition.Id == position.Id {
			if openPosition.Size.Eq(position.Size) {
				m.openPositions = append(m.openPositions[:idx], m.openPositions[idx+1:]...)
			} else {
				remainingSize := openPosition.Size.Sub(position.Size)
				openPosition.Size = remainingSize
			}
			break
		}
	}
}

func (m *Manager) OnOrderFilled(_ context.Context, filledOrder common.OrderFilled) {
	for idx := range m.openOrders {
		openOrder := &m.openOrders[idx]
		if openOrder.TraceID == filledOrder.OriginalOrder.TraceID {
			if openOrder.Size.Eq(filledOrder.OriginalOrder.FilledSize) {
				m.openOrders = append(m.openOrders[:idx], m.openOrders[idx+1:]...)
			} else {
				remainingSize := openOrder.Size.Sub(filledOrder.OriginalOrder.FilledSize)
				openOrder.Size = remainingSize
			}
			break
		}
	}
}

func (m *Manager) OnOrderCancelled(_ context.Context, filledOrder common.OrderCancelled) {
	for idx := range m.openOrders {
		openOrder := &m.openOrders[idx]
		if openOrder.TraceID == filledOrder.OriginalOrder.TraceID {
			if openOrder.Size.Eq(filledOrder.CancelledSize) {
				m.openOrders = append(m.openOrders[:idx], m.openOrders[idx+1:]...)
			} else {
				remainingSize := openOrder.Size.Sub(filledOrder.CancelledSize)
				openOrder.Size = remainingSize
			}
			break
		}
	}
}

func (m *Manager) OnOrderRejected(_ context.Context, rejectedOrder common.OrderRejected) {
	for idx := range m.openOrders {
		openOrder := &m.openOrders[idx]
		if openOrder.TraceID == rejectedOrder.OriginalOrder.TraceID {
			m.openOrders = append(m.openOrders[:idx], m.openOrders[idx+1:]...)
			break
		}
	}
}

func (m *Manager) OnSignal(_ context.Context, signal common.Signal) {
	if err := m.validateSignal(signal); err != nil {
		m.postSignalRejected(signal, err.Error(), "original signal dropped")
		return
	}

	sl, err := m.stopLossHandler.CalcStopLoss(signal)
	if err != nil {
		m.postSignalRejected(signal, err.Error(), "original signal dropped")
		return
	}

	tp, err := m.takeProfitHandler.CalcTakeProfit(signal)
	if err != nil {
		m.postSignalRejected(signal, err.Error(), "original signal dropped")
		return
	}

	pipDiff, pipVal := m.calcPipDiffAndVal(signal.Entry, sl, signal.Symbol)
	baseSize := m.calcSizeForBaseRiskRate(pipDiff, pipVal)
	minSize := m.calcSizeForMinRiskRate(pipDiff, pipVal)
	maxSize := m.calcSizeForMaxRiskRate(pipDiff, pipVal)

	sizeAfterMultipliers, multiplierComment := m.applyMultipliers(baseSize, signal)
	if sizeAfterMultipliers.Lte(fixed.Zero) {
		m.postSignalRejected(signal, multiplierComment, "original signal dropped")
		return
	}

	finalSize := clamp(sizeAfterMultipliers, minSize, maxSize)

	sl = m.rescalePrice(sl, signal.Symbol)
	tp = m.rescalePrice(tp, signal.Symbol)
	finalSize = m.rescaleSize(finalSize)

	if err := m.checkMarginRequirementsForSize(pipDiff, pipVal, finalSize); err != nil {
		m.postSignalRejected(signal, err.Error(), "original signal dropped")
		return
	}

	m.postSignalAccepted(signal, multiplierComment)
	order := m.createOpenOrder(signal.Entry, sl, tp, finalSize, signal.Symbol)
	m.postOrder(order)
}

func (m *Manager) checkForPositionAdjustment(tick common.Tick) {
	if m.adjustmentHandler != nil {
		for _, openPosition := range m.openPositions {
			if !strings.EqualFold(openPosition.Symbol, tick.Symbol) {
				continue
			}

			order, shouldAdjust := m.adjustmentHandler.AdjustPosition(openPosition)
			if !shouldAdjust {
				continue
			}
			m.postOrder(order)
		}
	}
}

func (m *Manager) validateSignal(signal common.Signal) error {
	_, ok := m.symbols[strings.ToUpper(signal.Symbol)]
	if !ok {
		return fmt.Errorf("%s symbol is not supported: %w", signal.Symbol, errSignalValidation)
	}
	for _, handler := range m.signalValidationHandlers {
		if err := handler.ValidateSignal(signal); err != nil {
			return fmt.Errorf("%w: %v", errSignalValidation, err)
		}
	}
	return nil
}

func (m *Manager) applyMultipliers(baseSize fixed.Point, signal common.Signal) (fixed.Point, string) {
	if m.sizeMultiplierStrategyHandler != nil {
		newSize, finalMultiplier, strategyName, factors := m.sizeMultiplierStrategyHandler(baseSize, signal)
		var commentBuilder strings.Builder
		commentBuilder.WriteString("Strategy: " + strategyName)
		commentBuilder.WriteString("; Final Multiplier: " + finalMultiplier.String())
		for _, factor := range factors {
			commentBuilder.WriteString("; " + factor.HandlerId + ": " + factor.Multiplier.String())
		}
		return newSize, commentBuilder.String()
	}
	return baseSize, "No size multiplier strategy applied."
}

func (m *Manager) calcPipDiffAndVal(entry, closePrice fixed.Point, symbol string) (fixed.Point, fixed.Point) {
	symbolInfo := m.symbols[strings.ToUpper(symbol)]
	return closePrice.Sub(entry).Abs().Div(symbolInfo.PipSize), symbolInfo.ContractSize.Mul(symbolInfo.PipSize)
}

func (m *Manager) calcSizeForBaseRiskRate(pipDiff, pipValue fixed.Point) fixed.Point {
	return m.calcSizeForRiskRate(pipDiff, pipValue, m.cfg.BaseRiskRate)
}

func (m *Manager) calcSizeForMinRiskRate(pipDiff, pipValue fixed.Point) fixed.Point {
	return m.calcSizeForRiskRate(pipDiff, pipValue, m.cfg.MinRiskRate)
}

func (m *Manager) calcSizeForMaxRiskRate(pipDiff, pipValue fixed.Point) fixed.Point {
	return m.calcSizeForRiskRate(pipDiff, pipValue, m.cfg.MaxRiskRate)
}

func (m *Manager) calcSizeForRiskRate(pipDiff, pipValue, riskRate fixed.Point) fixed.Point {
	return m.equity.Mul(riskRate.DivInt(100)).Div(pipDiff.Mul(pipValue))
}

func (m *Manager) calcOpenRiskRate() (fixed.Point, error) {
	openRiskRate := fixed.Zero
	for _, position := range m.openPositions {
		closePrice, err := m.getClosePrice(m.isLongPosition(position), position.Symbol)
		if err != nil {
			return fixed.Point{}, fmt.Errorf("unable to get close price: %w", err)
		}
		openPrice := position.OpenPrice
		pipDiff, pipVal := m.calcPipDiffAndVal(openPrice, closePrice, position.Symbol)
		riskRate := m.calcRiskRateForSize(pipDiff, pipVal, position.Size)
		openRiskRate = openRiskRate.Add(riskRate)
	}
	return openRiskRate, nil
}

func (m *Manager) calcRiskRateForSize(pipDiff, pipValue, size fixed.Point) fixed.Point {
	if m.equity.IsZero() {
		return fixed.Zero
	}
	return size.Mul(pipDiff.Mul(pipValue)).Div(m.equity).MulInt(100)
}

func (m *Manager) getLastTick(symbol string) (common.Tick, error) {
	tick, ok := m.tickCache[symbol]
	if !ok {
		return common.Tick{}, fmt.Errorf("tick %s not found", symbol)
	}
	return tick, nil
}

func (m *Manager) getOpenPrice(isLong bool, symbol string) (fixed.Point, error) {
	tick, err := m.getLastTick(symbol)
	if err != nil {
		return fixed.Point{}, err
	}
	if isLong {
		return tick.Ask, nil
	}
	return tick.Bid, nil
}

func (m *Manager) getClosePrice(isLong bool, symbol string) (fixed.Point, error) {
	tick, err := m.getLastTick(symbol)
	if err != nil {
		return fixed.Point{}, err
	}
	if isLong {
		return tick.Bid, nil
	}
	return tick.Ask, nil
}

func (m *Manager) isLongPosition(position common.Position) bool {
	return position.Side == common.PositionSideLong
}

func (m *Manager) determineOrderSide(entry, tp fixed.Point) common.OrderSide {
	if entry.Lt(tp) {
		return common.OrderSideBuy
	}
	return common.OrderSideSell
}

func (m *Manager) rescalePrice(price fixed.Point, symbolName string) fixed.Point {
	return price.Rescale(m.symbols[strings.ToUpper(symbolName)].Digits)
}

func (m *Manager) rescaleSize(size fixed.Point) fixed.Point {
	return size.Rescale(m.cfg.SizeDigits)
}

func (m *Manager) checkMarginRequirementsForSize(pipDiff, pipValue, size fixed.Point) error {
	riskRate := m.calcRiskRateForSize(pipDiff, pipValue, size)
	openRiskRate, err := m.calcOpenRiskRate()
	if err != nil {
		return fmt.Errorf("unable to calculate open risk rate: %w", err)
	}
	totalRiskRate := openRiskRate.Add(riskRate)
	if totalRiskRate.Gt(m.cfg.OpenRiskRate) {
		return fmt.Errorf("max open risk rate %s would be exceeded by %s%%",
			m.cfg.OpenRiskRate.String(), totalRiskRate.Sub(m.cfg.OpenRiskRate).String())
	}
	return nil
}

func (m *Manager) createOpenOrder(entry, sl, tp, size fixed.Point, symbol string) common.Order {
	if m.customOpenOrderHandler != nil {
		return m.customOpenOrderHandler(entry, sl, tp, size, symbol)
	}
	return m.defaultOpenOrderHandler(entry, sl, tp, size, symbol)
}

func (m *Manager) defaultOpenOrderHandler(entry, sl, tp, size fixed.Point, symbol string) common.Order {
	return common.Order{
		Command:     common.OrderCommandPositionOpen,
		Type:        common.OrderTypeMarket,
		TimeInForce: common.TimeInForceImmediateOrCancel,
		Side:        m.determineOrderSide(entry, tp),
		Price:       entry,
		Size:        size,
		StopLoss:    sl,
		TakeProfit:  tp,
		Source:      componentNameRiskManager,
		Symbol:      symbol,
		ExecutionId: utility.GetExecutionID(),
		TraceID:     utility.CreateTraceID(),
		TimeStamp:   m.ts,
	}
}

func (m *Manager) postOrder(order common.Order) {
	if err := m.router.Post(bus.OrderEvent, order); err != nil {
		slog.Error("unable to post order",
			"error", err, "order", order)
	} else {
		m.openOrders = append(m.openOrders, order)
	}
}

func (m *Manager) postSignalAccepted(signal common.Signal, comment string) {
	signalAccepted := common.SignalAccepted{
		Comment:        comment,
		OriginalSignal: signal,
		Source:         componentNameRiskManager,
		ExecutionID:    utility.GetExecutionID(),
		TraceID:        utility.CreateTraceID(),
		TimeStamp:      m.ts,
	}
	if err := m.router.Post(bus.SignalAcceptanceEvent, signalAccepted); err != nil {
		slog.Error("unable to post signal acceptance event",
			"error", err, "signal", signal, "accepted_signal", signalAccepted)
	}
}

func (m *Manager) postSignalRejected(signal common.Signal, reason, comment string) {
	signalRejected := common.SignalRejected{
		Reason:         reason,
		Comment:        comment,
		OriginalSignal: signal,
		Source:         componentNameRiskManager,
		ExecutionID:    utility.GetExecutionID(),
		TraceID:        utility.CreateTraceID(),
		TimeStamp:      m.ts,
	}
	if err := m.router.Post(bus.SignalRejectionEvent, signalRejected); err != nil {
		slog.Error("unable to post signal rejection event",
			"error", err, "signal", signal, "rejected_signal", signalRejected)
	}
}
