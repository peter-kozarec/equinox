package simulation

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type Simulator struct {
	router     *bus.Router
	aggregator *Aggregator
	audit      *Audit

	equity  fixed.Point
	balance fixed.Point

	simulationTime time.Time
	lastTick       common.Tick

	positionIdCounter common.PositionId
	openPositions     []*common.Position
	openOrders        []*common.Order

	cfg Configuration
}

func NewSimulator(router *bus.Router, audit *Audit, cfg Configuration) *Simulator {
	return &Simulator{
		router:     router,
		aggregator: NewAggregator(cfg.BarPeriod, router),
		audit:      audit,
		equity:     cfg.StartBalance,
		balance:    cfg.StartBalance,
		cfg:        cfg,
	}
}

func (s *Simulator) PrintDetails() {
	slog.Info("simulation details",
		"slippage", s.cfg.PipSlippage,
		"commissions", s.cfg.CommissionPerLot,
		"contract_size", s.cfg.ContractSize,
		"pip_size", s.cfg.PipSize,
		"aggregator_interval", s.aggregator.interval)
}

func (s *Simulator) OnOrder(order common.Order) {
	s.openOrders = append(s.openOrders, &order)
}

func (s *Simulator) OnTick(tick common.Tick) error {

	// Set simulation time from processed tick
	s.simulationTime = time.Unix(0, tick.TimeStamp)
	s.lastTick = tick

	// Store balance and equity before processing the tick
	lastBalance := s.balance
	lastEquity := s.equity

	s.checkPositions(tick)
	s.checkOrders(tick)
	s.processPendingChanges(tick)

	// Post balance event if the current balance changed after the tick was processed
	if lastBalance != s.balance {
		if err := s.router.Post(bus.BalanceEvent, s.balance); err != nil {
			slog.Error("unable to post balance event", "error", err)
		}
	}
	// Post equity event if the current equity changed after the tick was processed
	if lastEquity != s.equity {
		if err := s.router.Post(bus.EquityEvent, s.equity); err != nil {
			slog.Error("unable to post equity event", "error", err)
		}
	}

	s.audit.AddAccountSnapshot(s.balance, s.equity, s.simulationTime)

	if err := s.router.Post(bus.TickEvent, tick); err != nil {
		slog.Error("unable to post tick event", "error", err)
	}

	if err := s.aggregator.OnTick(tick); err != nil {
		slog.Warn("unable to aggregate ticks", "error", err)
	}

	return nil
}

func (s *Simulator) CloseAllOpenPositions() {

	s.equity = s.balance

	for idx := range s.openPositions {
		position := s.openPositions[idx]

		closePrice := s.lastTick.Bid
		if position.IsShort() {
			closePrice = s.lastTick.Ask
		}

		s.calcPositionProfits(position, closePrice)
		s.equity = s.equity.Add(position.NetProfit)

		position.State = common.Closed
		position.ClosePrice = closePrice
		position.CloseTime = time.Unix(0, s.lastTick.TimeStamp)
		s.audit.AddClosedPosition(*position)
	}

	s.balance = s.equity
	s.audit.addSnapshot(s.balance, s.equity, s.simulationTime)
}

func (s *Simulator) checkPositions(tick common.Tick) {

	for idx := range s.openPositions {
		position := s.openPositions[idx]

		if s.shouldClosePosition(*position, tick) {
			position.State = common.PendingClose
		}
	}
}

func (s *Simulator) checkOrders(tick common.Tick) {

	tmpOpenOrders := make([]*common.Order, 0, len(s.openOrders))

	for idx := range s.openOrders {
		order := s.openOrders[idx]

		switch order.Command {
		case common.CmdOpen:
			switch order.OrderType {
			case common.Market:
				if err := s.executeOpenOrder(order.Size, order.StopLoss, order.TakeProfit); err != nil {
					slog.Warn("unable to execute open order", "error", err)
				}
			case common.Limit:
				if !s.shouldOpenPosition(order.Price, order.Size, tick) {
					tmpOpenOrders = append(tmpOpenOrders, order)
					continue
				}
				if err := s.executeOpenOrder(order.Size, order.StopLoss, order.TakeProfit); err != nil {
					slog.Warn("unable to execute open order", "error", err)
				}
			}
		case common.CmdClose:
			if err := s.executeCloseOrder(order.PositionId); err != nil {
				slog.Warn("unable to execute close order", "error", err)
			}
		case common.CmdModify:
			if err := s.modifyPosition(order.PositionId, order.StopLoss, order.TakeProfit); err != nil {
				slog.Warn("unable to modify open position", "error", err)
			}
		case common.CmdRemove:
			continue
		default:
			slog.Warn("unknown command", "cmd", order.Command)
		}
	}

	s.openOrders = tmpOpenOrders
}

func (s *Simulator) executeCloseOrder(id common.PositionId) error {

	for idx := range s.openPositions {
		position := s.openPositions[idx]

		if position.Id == id {
			position.State = common.PendingClose
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", id)
}

func (s *Simulator) executeOpenOrder(size, stopLoss, takeProfit fixed.Point) error {

	// Validate position size
	if size.IsZero() {
		return fmt.Errorf("position size cannot be zero")
	}

	// Validate stop loss and take profit logic
	if size.Gt(fixed.Zero) { // Long position
		if !stopLoss.IsZero() && !takeProfit.IsZero() && stopLoss.Gte(takeProfit) {
			return fmt.Errorf("long position: stop loss must be less than take profit")
		}
	} else { // Short position
		if !stopLoss.IsZero() && !takeProfit.IsZero() && stopLoss.Lte(takeProfit) {
			return fmt.Errorf("short position: stop loss must be greater than take profit")
		}
	}

	s.positionIdCounter++
	s.openPositions = append(s.openPositions, &common.Position{
		Id:         s.positionIdCounter,
		State:      common.PendingOpen,
		Size:       size,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
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

	if position.IsLong() {
		// Long, check if take profit or stop loss has been reached
		if (!position.TakeProfit.IsZero() && tick.Bid.Gte(position.TakeProfit)) ||
			(!position.StopLoss.IsZero() && tick.Bid.Lte(position.StopLoss)) {
			return true
		}
	} else if position.IsShort() {
		// Short, check if take profit or stop loss has been reached
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

		var openPrice, closePrice fixed.Point
		if position.IsLong() {
			openPrice = tick.Ask
			closePrice = tick.Bid
		} else if position.IsShort() {
			openPrice = tick.Bid
			closePrice = tick.Ask
		} else {
			panic("invalid position, unable to determine long/short side")
		}

		switch position.State {
		case common.PendingOpen:
			position.State = common.Opened
			position.OpenPrice = openPrice
			position.OpenTime = time.Unix(0, tick.TimeStamp)
			if err := s.router.Post(bus.PositionOpenedEvent, *position); err != nil {
				slog.Warn("unable to post position opened event", "error", err)
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		case common.PendingClose:
			position.State = common.Closed
			position.ClosePrice = closePrice
			position.CloseTime = time.Unix(0, tick.TimeStamp)
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

	if position.IsLong() {
		pipPnL = closePrice.Sub(position.OpenPrice)
	} else if position.IsShort() {
		pipPnL = position.OpenPrice.Sub(closePrice)
	} else {
		panic("invalid position, unable to determine long/short side")
	}

	pipPnL = pipPnL.Sub(s.cfg.PipSlippage.MulInt64(2))
	pips := pipPnL.Div(s.cfg.PipSize)

	// Use average price for more accurate lot value
	avgPrice := position.OpenPrice.Add(closePrice).DivInt64(2)
	currentLotValue := s.cfg.PipSize.Mul(s.cfg.ContractSize).Mul(avgPrice)

	position.GrossProfit = pips.Mul(position.Size.Abs()).Mul(currentLotValue)

	commission := s.cfg.CommissionPerLot.MulInt64(2).Mul(position.Size.Abs())
	position.NetProfit = position.GrossProfit.Sub(commission)
}
