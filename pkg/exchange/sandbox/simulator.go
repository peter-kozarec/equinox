package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	positionStatusPendingOpen  common.PositionStatus = "pending-open"
	positionStatusPendingClose common.PositionStatus = "pending-close"

	simulatorComponentName = "exchange.sandbox.simulator"
)

type Simulator struct {
	router          *bus.Router
	accountCurrency string

	slippage          fixed.Point
	rateProvider      RateProvider
	symbolsMap        map[string]exchange.SymbolInfo
	commissionHandler CommissionHandler
	swapHandler       SwapHandler

	firstPostDone bool
	equity        fixed.Point
	balance       fixed.Point

	simulationTime time.Time
	lastTickMap    map[string]common.Tick

	positionIdCounter common.PositionId
	openPositions     []*common.Position
	openOrders        []*common.Order
}

func NewSimulator(router *bus.Router, accountCurrency string, startBalance fixed.Point, options ...Option) *Simulator {
	s := &Simulator{
		router:          router,
		accountCurrency: accountCurrency,
		slippage:        fixed.Zero,
		symbolsMap:      make(map[string]exchange.SymbolInfo),
		equity:          startBalance,
		balance:         startBalance,
		lastTickMap:     make(map[string]common.Tick),
	}

	for _, option := range options {
		option(s)
	}

	return s
}

func (s *Simulator) OnOrder(_ context.Context, order common.Order) {
	_, ok := s.symbolsMap[strings.ToUpper(order.Symbol)]
	if !ok {
		slog.Warn("symbol info is not present, dropping order", "order", order)
		return
	}
	s.openOrders = append(s.openOrders, &order)
}

func (s *Simulator) OnTick(_ context.Context, tick common.Tick) {
	_, ok := s.symbolsMap[strings.ToUpper(tick.Symbol)]
	if !ok {
		slog.Warn("symbol info is not present, dropping tick", "tick", tick)
		return
	}

	s.simulationTime = tick.TimeStamp
	s.lastTickMap[strings.ToUpper(tick.Symbol)] = tick

	if !s.firstPostDone {
		s.firstPostDone = true
		if err := s.router.Post(bus.BalanceEvent, common.Balance{
			Source:      simulatorComponentName,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   s.simulationTime,
			Value:       s.balance,
		}); err != nil {
			slog.Error("unable to post balance event", "error", err)
		}
		if err := s.router.Post(bus.EquityEvent, common.Equity{
			Source:      simulatorComponentName,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   s.simulationTime,
			Value:       s.equity,
		}); err != nil {
			slog.Error("unable to post balance event", "error", err)
		}
	}

	lastBalance := s.balance
	lastEquity := s.equity

	s.checkPositions(tick)
	s.checkOrders(tick)
	s.processPendingChanges(tick)

	if lastBalance != s.balance {
		if err := s.router.Post(bus.BalanceEvent, common.Balance{
			Source:      simulatorComponentName,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   s.simulationTime,
			Value:       s.balance,
		}); err != nil {
			slog.Error("unable to post balance event", "error", err)
		}
	}
	if lastEquity != s.equity {
		if err := s.router.Post(bus.EquityEvent, common.Equity{
			Source:      simulatorComponentName,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   s.simulationTime,
			Value:       s.equity,
		}); err != nil {
			slog.Error("unable to post equity event", "error", err)
		}
	}
}

func (s *Simulator) CloseAllOpenPositions() {
	s.equity = s.balance

	for idx := range s.openPositions {
		position := s.openPositions[idx]

		closePrice := s.lastTickMap[strings.ToUpper(position.Symbol)].Bid
		if position.Side == common.PositionSideShort {
			closePrice = s.lastTickMap[strings.ToUpper(position.Symbol)].Ask
		}

		s.calcPositionProfits(position, closePrice)
		s.equity = s.equity.Add(position.NetProfit)

		position.Status = common.PositionStatusClosed
		position.ClosePrice = closePrice
		position.CloseTime = s.simulationTime

		if err := s.router.Post(bus.PositionCloseEvent, *position); err != nil {
			slog.Warn("unable to post position closed event", "error", err)
		}
	}

	s.balance = s.equity
}

func (s *Simulator) checkPositions(tick common.Tick) {
	for idx := range s.openPositions {
		position := s.openPositions[idx]

		if s.shouldClosePosition(*position, tick) {
			position.Status = positionStatusPendingClose
		}
	}
}

func (s *Simulator) checkOrders(tick common.Tick) {
	tmpOpenOrders := make([]*common.Order, 0, len(s.openOrders))
	orderAccepted := true

	for idx := range s.openOrders {
		order := s.openOrders[idx]
		if !strings.EqualFold(order.Symbol, tick.Symbol) {
			tmpOpenOrders = append(tmpOpenOrders, order)
			continue
		}

		switch order.Command {
		case common.OrderCommandPositionOpen:
			switch order.Type {
			case common.OrderTypeMarket:
				if err := s.executeOpenOrder(*order); err != nil {
					slog.Warn("unable to execute open order", "error", err)
					orderAccepted = false
				}
			case common.OrderTypeLimit:
				if !s.shouldOpenPosition(order.Price, order.Size, tick) {
					tmpOpenOrders = append(tmpOpenOrders, order)
					continue
				}
				if err := s.executeOpenOrder(*order); err != nil {
					slog.Warn("unable to execute open order", "error", err)
					orderAccepted = false
				}
			}
		case common.OrderCommandPositionClose:
			if err := s.executeCloseOrder(order.PositionId); err != nil {
				slog.Warn("unable to execute close order", "error", err)
				orderAccepted = false
			}
		case common.OrderCommandPositionModify:
			if err := s.modifyPosition(order.PositionId, order.StopLoss, order.TakeProfit); err != nil {
				slog.Warn("unable to modify open position", "error", err)
				orderAccepted = false
			}
		default:
			slog.Warn("unknown command", "cmd", order.Command)
		}

		if orderAccepted {
			if err := s.router.Post(bus.OrderAcceptanceEvent, common.OrderAccepted{
				Source:        simulatorComponentName,
				ExecutionId:   utility.GetExecutionID(),
				TraceID:       utility.CreateTraceID(),
				TimeStamp:     s.simulationTime,
				OriginalOrder: *order,
			}); err != nil {
				slog.Warn("unable to post order accepted event", "error", err)
			}
		} else {
			if err := s.router.Post(bus.OrderRejectionEvent, common.OrderRejected{
				Source:        simulatorComponentName,
				ExecutionId:   utility.GetExecutionID(),
				TraceID:       utility.CreateTraceID(),
				TimeStamp:     s.simulationTime,
				OriginalOrder: *order,
				Reason:        "unable to execute order",
			}); err != nil {
				slog.Warn("unable to post order rejected event", "error", err)
			}
			orderAccepted = true
		}
	}

	s.openOrders = tmpOpenOrders
}

func (s *Simulator) executeCloseOrder(id common.PositionId) error {
	for idx := range s.openPositions {
		position := s.openPositions[idx]

		if position.Id == id {
			position.Status = positionStatusPendingClose
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", id)
}

func (s *Simulator) executeOpenOrder(order common.Order) error {
	if order.Size.IsZero() {
		return fmt.Errorf("position size cannot be zero")
	}

	var positionSide common.PositionSide
	if order.Side == common.OrderSideBuy {
		if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Gte(order.TakeProfit) {
			return fmt.Errorf("long position: stop loss must be less than take profit")
		}
		positionSide = common.PositionSideLong
	} else {
		if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Lte(order.TakeProfit) {
			return fmt.Errorf("short position: stop loss must be greater than take profit")
		}
		positionSide = common.PositionSideShort
	}

	exchangeRate := fixed.One
	conversionFeeRate := fixed.Zero
	if s.rateProvider != nil {
		var err error
		symbolInfo := s.symbolsMap[strings.ToUpper(order.Symbol)]
		exchangeRate, conversionFeeRate, err = s.rateProvider.ExchangeRate(s.accountCurrency, symbolInfo.QuoteCurrency, s.simulationTime)
		if err != nil {
			return fmt.Errorf("unable to determine exchange rate and conversion fee rate for order %v :%w", order, err)
		}
	}

	s.positionIdCounter++
	s.openPositions = append(s.openPositions, &common.Position{
		Source:            simulatorComponentName,
		Symbol:            order.Symbol,
		ExecutionID:       utility.GetExecutionID(),
		TraceID:           utility.CreateTraceID(),
		OrderTraceIDs:     []utility.TraceID{order.TraceID},
		Id:                s.positionIdCounter,
		Status:            positionStatusPendingOpen,
		Side:              positionSide,
		Size:              order.Size,
		StopLoss:          order.StopLoss,
		TakeProfit:        order.TakeProfit,
		Currency:          s.accountCurrency,
		ExchangeRate:      exchangeRate,
		ConversionFeeRate: conversionFeeRate,
		TimeStamp:         s.simulationTime,
	})
	return nil
}

func (s *Simulator) modifyPosition(id common.PositionId, stopLoss, takeProfit fixed.Point) error {
	for idx := range s.openPositions {
		position := s.openPositions[idx]

		if position.Id == id {
			position.StopLoss = stopLoss
			position.TakeProfit = takeProfit
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", id)
}

func (s *Simulator) shouldOpenPosition(price, size fixed.Point, tick common.Tick) bool {
	if size.Gt(fixed.Zero) && tick.Ask.Lte(price) {
		return true
	}
	if size.Lt(fixed.Zero) && tick.Bid.Gte(price) {
		return true
	}
	return false
}

func (s *Simulator) shouldClosePosition(position common.Position, tick common.Tick) bool {
	if !strings.EqualFold(position.Symbol, tick.Symbol) {
		return false
	}

	if position.Side == common.PositionSideLong {
		if (!position.TakeProfit.IsZero() && tick.Bid.Gte(position.TakeProfit)) ||
			(!position.StopLoss.IsZero() && tick.Bid.Lte(position.StopLoss)) {
			return true
		}
	} else {
		if (!position.TakeProfit.IsZero() && tick.Ask.Lte(position.TakeProfit)) ||
			(!position.StopLoss.IsZero() && tick.Ask.Gte(position.StopLoss)) {
			return true
		}
	}
	return false
}

func (s *Simulator) processPendingChanges(tick common.Tick) {
	tmpOpenPositions := make([]*common.Position, 0, len(s.openPositions))
	s.equity = s.balance

	for idx := range s.openPositions {
		position := s.openPositions[idx]
		if !strings.EqualFold(position.Symbol, tick.Symbol) {
			tmpOpenPositions = append(tmpOpenPositions, position)
			continue
		}

		openPrice := tick.Bid
		closePrice := tick.Ask
		if position.Side == common.PositionSideLong {
			openPrice = tick.Ask
			closePrice = tick.Bid
		}

		switch position.Status {
		case positionStatusPendingOpen:
			position.Status = common.PositionStatusOpen
			position.OpenPrice = openPrice
			position.OpenTime = tick.TimeStamp
			if err := s.router.Post(bus.PositionOpenEvent, *position); err != nil {
				slog.Warn("unable to post position opened event", "error", err)
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		case positionStatusPendingClose:
			position.Status = common.PositionStatusClosed
			position.ClosePrice = closePrice
			position.CloseTime = tick.TimeStamp
			s.calcPositionProfits(position, closePrice)
			position.TimeStamp = s.simulationTime
			s.balance = s.balance.Add(position.NetProfit)
			if err := s.router.Post(bus.PositionCloseEvent, *position); err != nil {
				slog.Warn("unable to post position closed event", "error", err)
			}
		default:
			s.calcPositionProfits(position, closePrice)
			position.TimeStamp = s.simulationTime
			s.equity = s.equity.Add(position.NetProfit)
			if err := s.router.Post(bus.PositionUpdateEvent, *position); err != nil {
				slog.Warn("unable to post position pnl updated event", "error", err)
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		}
	}

	s.openPositions = tmpOpenPositions
}

func (s *Simulator) calcPositionProfits(position *common.Position, closePrice fixed.Point) {
	symbolInfo := s.symbolsMap[strings.ToUpper(position.Symbol)]

	priceDiff := position.OpenPrice.Sub(closePrice)
	if position.Side == common.PositionSideLong {
		priceDiff = closePrice.Sub(position.OpenPrice)
	}

	if position.Status == common.PositionStatusOpen {
		if s.commissionHandler != nil {
			position.Commissions = s.commissionHandler(symbolInfo, *position)
		}
		if !position.ConversionFeeRate.IsZero() {
			position.ConversionFee = position.Size.Mul(position.ExchangeRate).Mul(position.ConversionFeeRate)
		}
		position.Slippage = s.slippage
	}

	if position.Status == common.PositionStatusClosed {
		if s.commissionHandler != nil {
			position.Commissions = position.Commissions.Add(s.commissionHandler(symbolInfo, *position))
		}
		if !position.ConversionFeeRate.IsZero() {
			position.ConversionFee = position.ConversionFee.Add(position.Size.Mul(position.ExchangeRate).Mul(position.ConversionFeeRate))
		}
		position.Slippage = position.Slippage.Add(s.slippage)
	}

	daysPassed := int(s.simulationTime.Sub(position.TimeStamp).Hours()) / 24
	if daysPassed > 0 && s.swapHandler != nil {
		for range daysPassed {
			dailySwap := s.swapHandler(symbolInfo, *position)
			position.Swaps = position.Swaps.Add(dailySwap)
		}
	}

	priceDiff = priceDiff.Sub(s.slippage)
	position.GrossProfit = priceDiff.Mul(position.Size).Mul(symbolInfo.ContractSize).Mul(position.ExchangeRate)
	position.NetProfit = position.GrossProfit.Sub(position.Commissions).Sub(position.Swaps).Sub(position.ConversionFee)
}
