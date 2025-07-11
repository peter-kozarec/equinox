package simulation

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	// Internal position statuses
	positionStatusPendingOpen  common.PositionStatus = "pending-open"
	positionStatusPendingClose common.PositionStatus = "pending-close"
)

type Simulator struct {
	instrument common.Instrument
	router     *bus.Router
	audit      *Audit

	equity  fixed.Point
	balance fixed.Point

	simulationTime time.Time
	lastTick       common.Tick

	positionIdCounter common.PositionId
	openPositions     []*common.Position
	openOrders        []*common.Order
}

func NewSimulator(router *bus.Router, audit *Audit, instrument common.Instrument, startBalance fixed.Point) *Simulator {
	return &Simulator{
		instrument: instrument,
		router:     router,
		audit:      audit,
		equity:     startBalance,
		balance:    startBalance,
	}
}

func (s *Simulator) OnOrder(_ context.Context, order common.Order) {
	s.openOrders = append(s.openOrders, &order)
}

func (s *Simulator) OnTick(_ context.Context, tick common.Tick) {

	// Set simulation time from a processed tick
	s.simulationTime = tick.TimeStamp
	s.lastTick = tick

	// Store balance and equity before processing the tick
	lastBalance := s.balance
	lastEquity := s.equity

	s.checkPositions(tick)
	s.checkOrders(tick)
	s.processPendingChanges(tick)

	// Post balance event if the current balance changed after the tick was processed
	if lastBalance != s.balance {
		if err := s.router.Post(bus.BalanceEvent, common.Balance{
			Source:      componentName,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   s.simulationTime,
			Value:       s.balance,
		}); err != nil {
			slog.Error("unable to post balance event", "error", err)
		}
	}
	// Post equity event if the current equity changed after the tick was processed
	if lastEquity != s.equity {
		if err := s.router.Post(bus.EquityEvent, common.Equity{
			Source:      componentName,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   s.simulationTime,
			Value:       s.equity,
		}); err != nil {
			slog.Error("unable to post equity event", "error", err)
		}
	}

	s.audit.AddAccountSnapshot(s.balance, s.equity, s.simulationTime)
}

func (s *Simulator) CloseAllOpenPositions() {

	s.equity = s.balance

	for idx := range s.openPositions {
		position := s.openPositions[idx]

		closePrice := s.lastTick.Bid
		if position.Side == common.PositionSideShort {
			closePrice = s.lastTick.Ask
		}

		s.calcPositionProfits(position, closePrice)
		s.equity = s.equity.Add(position.NetProfit)

		position.Status = common.PositionStatusClosed
		position.ClosePrice = closePrice
		position.CloseTime = s.lastTick.TimeStamp
		s.audit.AddClosedPosition(*position)
	}

	s.balance = s.equity
	s.audit.addSnapshot(s.balance, s.equity, s.simulationTime)
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
			if err := s.router.Post(bus.OrderAcceptedEvent, common.OrderAccepted{
				Source:        componentName,
				ExecutionId:   utility.GetExecutionID(),
				TraceID:       utility.CreateTraceID(),
				TimeStamp:     s.simulationTime,
				OriginalOrder: *order,
			}); err != nil {
				slog.Warn("unable to post order accepted event", "error", err)
			}
		} else {
			if err := s.router.Post(bus.OrderRejectedEvent, common.OrderRejected{
				Source:        componentName,
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

	// Validate position size
	if order.Size.IsZero() {
		return fmt.Errorf("position size cannot be zero")
	}

	var positionSide common.PositionSide

	// Validate stop loss and take profit logic
	if order.Side == common.OrderSideBuy { // Long position
		if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Gte(order.TakeProfit) {
			return fmt.Errorf("long position: stop loss must be less than take profit")
		}
		positionSide = common.PositionSideLong
	} else if order.Side == common.OrderSideSell { // Short position
		if !order.StopLoss.IsZero() && !order.TakeProfit.IsZero() && order.StopLoss.Lte(order.TakeProfit) {
			return fmt.Errorf("short position: stop loss must be greater than take profit")
		}
		positionSide = common.PositionSideShort
	} else {
		panic("invalid order, unable to determine buy/sell side")
	}

	s.positionIdCounter++
	s.openPositions = append(s.openPositions, &common.Position{
		Source:        componentName,
		Symbol:        s.instrument.Symbol,
		ExecutionID:   utility.GetExecutionID(),
		TraceID:       utility.CreateTraceID(),
		OrderTraceIDs: []utility.TraceID{order.TraceID},
		Id:            s.positionIdCounter,
		Status:        positionStatusPendingOpen,
		Side:          positionSide,
		Size:          order.Size,
		StopLoss:      order.StopLoss,
		TakeProfit:    order.TakeProfit,
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

	// For long limit: trigger when Ask <= limit price
	if size.Gt(fixed.Zero) && tick.Ask.Lte(price) {
		return true
	}
	// For short limit: trigger when Bid >= limit price
	if size.Lt(fixed.Zero) && tick.Bid.Gte(price) {
		return true
	}
	return false
}

func (s *Simulator) shouldClosePosition(position common.Position, tick common.Tick) bool {

	if position.Side == common.PositionSideLong {
		// Long, check if take profit or stop loss has been reached
		if (!position.TakeProfit.IsZero() && tick.Bid.Gte(position.TakeProfit)) ||
			(!position.StopLoss.IsZero() && tick.Bid.Lte(position.StopLoss)) {
			return true
		}
		return false
	} else if position.Side == common.PositionSideShort {
		// Short, check if take profit or stop loss has been reached
		if (!position.TakeProfit.IsZero() && tick.Ask.Lte(position.TakeProfit)) ||
			(!position.StopLoss.IsZero() && tick.Ask.Gte(position.StopLoss)) {
			return true
		}
		return false
	}

	panic("invalid position, unable to determine long/short side")
}

func (s *Simulator) processPendingChanges(tick common.Tick) {
	tmpOpenPositions := make([]*common.Position, 0, len(s.openPositions))
	s.equity = s.balance

	for idx := range s.openPositions {
		position := s.openPositions[idx]
		position.TimeStamp = s.simulationTime

		var openPrice, closePrice fixed.Point
		if position.Side == common.PositionSideLong {
			openPrice = tick.Ask
			closePrice = tick.Bid
		} else if position.Side == common.PositionSideShort {
			openPrice = tick.Bid
			closePrice = tick.Ask
		} else {
			panic("invalid position, unable to determine long/short side")
		}

		switch position.Status {
		case positionStatusPendingOpen:
			position.Status = common.PositionStatusOpen
			position.OpenPrice = openPrice
			position.OpenTime = tick.TimeStamp
			if err := s.router.Post(bus.PositionOpenedEvent, *position); err != nil {
				slog.Warn("unable to post position opened event", "error", err)
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		case positionStatusPendingClose:
			position.Status = common.PositionStatusClosed
			position.ClosePrice = closePrice
			position.CloseTime = tick.TimeStamp
			s.calcPositionProfits(position, closePrice)
			s.balance = s.balance.Add(position.NetProfit)
			s.audit.AddClosedPosition(*position)
			if err := s.router.Post(bus.PositionClosedEvent, *position); err != nil {
				slog.Warn("unable to post position closed event", "error", err)
			}
		default:
			s.calcPositionProfits(position, closePrice)
			s.equity = s.equity.Add(position.NetProfit)
			if err := s.router.Post(bus.PositionPnLUpdatedEvent, *position); err != nil {
				slog.Warn("unable to post position pnl updated event", "error", err)
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		}
	}

	s.openPositions = tmpOpenPositions
}

func (s *Simulator) calcPositionProfits(position *common.Position, closePrice fixed.Point) {
	var pipPnL fixed.Point

	if position.Side == common.PositionSideLong {
		pipPnL = closePrice.Sub(position.OpenPrice)
	} else if position.Side == common.PositionSideShort {
		pipPnL = position.OpenPrice.Sub(closePrice)
	} else {
		panic("invalid position, unable to determine long/short side")
	}

	pipPnL = pipPnL.Sub(s.instrument.PipSlippage.MulInt64(2))
	pips := pipPnL.Div(s.instrument.PipSize)

	// Use average price for more accurate lot value
	avgPrice := position.OpenPrice.Add(closePrice).DivInt64(2)
	currentLotValue := s.instrument.PipSize.Mul(s.instrument.ContractSize).Mul(avgPrice)

	position.GrossProfit = pips.Mul(position.Size.Abs()).Mul(currentLotValue)

	commission := s.instrument.CommissionPerLot.MulInt64(2).Mul(position.Size.Abs())

	position.Commission = commission
	position.NetProfit = position.GrossProfit.Sub(commission)
}
