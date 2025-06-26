package simulation

import (
	"fmt"
	"time"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/model"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
)

type Simulator struct {
	logger     *zap.Logger
	router     *bus.Router
	aggregator *Aggregator
	audit      *Audit

	equity  fixed.Point
	balance fixed.Point

	simulationTime time.Time
	lastTick       model.Tick

	positionIdCounter model.PositionId
	openPositions     []*model.Position
	openOrders        []*model.Order

	cfg Configuration
}

func NewSimulator(logger *zap.Logger, router *bus.Router, audit *Audit, cfg Configuration) *Simulator {
	return &Simulator{
		logger:     logger,
		router:     router,
		aggregator: NewAggregator(cfg.BarPeriod, router),
		audit:      audit,
		equity:     cfg.StartBalance,
		balance:    cfg.StartBalance,
		cfg:        cfg,
	}
}

func (s *Simulator) PrintDetails() {
	s.logger.Info("simulation details",
		zap.String("slippage", s.cfg.PipSlippage.String()),
		zap.String("commissions", s.cfg.CommissionPerLot.String()),
		zap.String("contract_size", s.cfg.ContractSize.String()),
		zap.String("pip_size", s.cfg.PipSize.String()),
		zap.String("aggregator_interval", s.aggregator.interval.String()))
}

func (s *Simulator) OnOrder(order model.Order) {
	s.openOrders = append(s.openOrders, &order)
}

func (s *Simulator) OnTick(tick model.Tick) error {

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
			s.logger.Error("unable to post balance event", zap.Error(err))
		}
	}
	// Post equity event if the current equity changed after the tick was processed
	if lastEquity != s.equity {
		if err := s.router.Post(bus.EquityEvent, s.equity); err != nil {
			s.logger.Error("unable to post equity event", zap.Error(err))
		}
	}

	s.audit.AddAccountSnapshot(s.balance, s.equity, s.simulationTime)

	if err := s.router.Post(bus.TickEvent, tick); err != nil {
		s.logger.Error("unable to post tick event", zap.Error(err))
	}

	if err := s.aggregator.OnTick(tick); err != nil {
		s.logger.Warn("unable to aggregate ticks", zap.Error(err))
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

		position.State = model.Closed
		position.ClosePrice = closePrice
		position.CloseTime = time.Unix(0, s.lastTick.TimeStamp)
		s.audit.AddClosedPosition(*position)
	}

	s.balance = s.equity
	s.audit.addSnapshot(s.balance, s.equity, s.simulationTime)
}

func (s *Simulator) checkPositions(tick model.Tick) {

	for idx := range s.openPositions {
		position := s.openPositions[idx]

		if s.shouldClosePosition(*position, tick) {
			position.State = model.PendingClose
		}
	}
}

func (s *Simulator) checkOrders(tick model.Tick) {

	tmpOpenOrders := make([]*model.Order, 0, len(s.openOrders))

	for idx := range s.openOrders {
		order := s.openOrders[idx]

		switch order.Command {
		case model.CmdOpen:
			switch order.OrderType {
			case model.Market:
				if err := s.executeOpenOrder(order.Size, order.StopLoss, order.TakeProfit); err != nil {
					s.logger.Warn("unable to execute open order", zap.Error(err))
				}
			case model.Limit:
				if !s.shouldOpenPosition(order.Price, order.Size, tick) {
					tmpOpenOrders = append(tmpOpenOrders, order)
					continue
				}
				if err := s.executeOpenOrder(order.Size, order.StopLoss, order.TakeProfit); err != nil {
					s.logger.Warn("unable to execute open order", zap.Error(err))
				}
			}
		case model.CmdClose:
			if err := s.executeCloseOrder(order.PositionId); err != nil {
				s.logger.Warn("unable to execute close order", zap.Error(err))
			}
		case model.CmdModify:
			if err := s.modifyPosition(order.PositionId, order.StopLoss, order.TakeProfit); err != nil {
				s.logger.Warn("unable to modify open position", zap.Error(err))
			}
		case model.CmdRemove:
			continue
		default:
			s.logger.Warn("unknown command", zap.Any("cmd", order.Command))
		}
	}

	s.openOrders = tmpOpenOrders
}

func (s *Simulator) executeCloseOrder(id model.PositionId) error {

	for idx := range s.openPositions {
		position := s.openPositions[idx]

		if position.Id == id {
			position.State = model.PendingClose
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
	s.openPositions = append(s.openPositions, &model.Position{
		Id:         s.positionIdCounter,
		State:      model.PendingOpen,
		Size:       size,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
	})
	return nil
}

func (s *Simulator) modifyPosition(id model.PositionId, stopLoss, takeProfit fixed.Point) error {

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

func (s *Simulator) shouldOpenPosition(price, size fixed.Point, tick model.Tick) bool {

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

func (s *Simulator) shouldClosePosition(position model.Position, tick model.Tick) bool {

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

func (s *Simulator) processPendingChanges(tick model.Tick) {
	tmpOpenPositions := make([]*model.Position, 0, len(s.openPositions))
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
		case model.PendingOpen:
			position.State = model.Opened
			position.OpenPrice = openPrice
			position.OpenTime = time.Unix(0, tick.TimeStamp)
			if err := s.router.Post(bus.PositionOpenedEvent, *position); err != nil {
				s.logger.Warn("unable to post position opened event", zap.Error(err))
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		case model.PendingClose:
			position.State = model.Closed
			position.ClosePrice = closePrice
			position.CloseTime = time.Unix(0, tick.TimeStamp)
			s.calcPositionProfits(position, closePrice)
			s.balance = s.balance.Add(position.NetProfit)
			s.audit.AddClosedPosition(*position)
			if err := s.router.Post(bus.PositionClosedEvent, *position); err != nil {
				s.logger.Warn("unable to post position closed event", zap.Error(err))
			}
		default:
			s.calcPositionProfits(position, closePrice)
			s.equity = s.equity.Add(position.NetProfit)
			if err := s.router.Post(bus.PositionPnLUpdatedEvent, *position); err != nil {
				s.logger.Warn("unable to post position pnl updated event", zap.Error(err))
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		}
	}

	s.openPositions = tmpOpenPositions
}

func (s *Simulator) calcPositionProfits(position *model.Position, closePrice fixed.Point) {
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
	avgPrice := position.OpenPrice.Add(closePrice).DivInt(2)
	currentLotValue := s.cfg.PipSize.Mul(s.cfg.ContractSize).Mul(avgPrice)

	position.GrossProfit = pips.Mul(position.Size.Abs()).Mul(currentLotValue)

	commission := s.cfg.CommissionPerLot.MulInt64(2).Mul(position.Size.Abs())
	position.NetProfit = position.GrossProfit.Sub(commission)
}
