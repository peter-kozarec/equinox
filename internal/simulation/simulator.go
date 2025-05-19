package simulation

import (
	"fmt"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/utility"
	"time"
)

type Simulator struct {
	logger     *zap.Logger
	router     *bus.Router
	aggregator *Aggregator
	audit      *Audit

	equity  utility.Fixed
	balance utility.Fixed

	simulationTime time.Time

	positionIdCounter model.PositionId
	openPositions     []*model.Position
	openOrders        []*model.Order

	slippage    utility.Fixed
	commissions utility.Fixed
	lotValue    utility.Fixed
	pipSize     utility.Fixed
}

func NewSimulator(logger *zap.Logger, router *bus.Router, audit *Audit) *Simulator {
	return &Simulator{
		logger:      logger,
		router:      router,
		aggregator:  NewAggregator(BarPeriod, router),
		audit:       audit,
		equity:      utility.NewFixed(StartingBalance, StartingBalancePrecision),
		balance:     utility.NewFixed(StartingBalance, StartingBalancePrecision),
		slippage:    utility.NewFixed(Slippage, SlippagePrecision),
		commissions: utility.NewFixed(Commission, CommissionPrecision),
		lotValue:    utility.NewFixed(LotValue, LotValuePrecision),
		pipSize:     utility.NewFixed(PipSize, PipSizePrecision),
	}
}

func (simulator *Simulator) OnOrder(order *model.Order) error {
	simulator.openOrders = append(simulator.openOrders, order)
	return nil
}

func (simulator *Simulator) OnTick(tick *model.Tick) error {

	// Set simulation time from processed tick
	simulator.simulationTime = time.Unix(0, tick.TimeStamp)

	// Store balance and equity before processing the tick
	lastBalance := simulator.balance
	lastEquity := simulator.equity

	simulator.checkPositions(tick)
	simulator.checkOrders(tick)
	simulator.processPendingChanges(tick)

	// Post balance event if the current balance changed after the tick was processed
	if lastBalance != simulator.balance {
		if err := simulator.router.Post(bus.BalanceEvent, &simulator.balance); err != nil {
			simulator.logger.Error("unable to post balance event", zap.Error(err))
		}
	}
	// Post equity event if the current equity changed after the tick was processed
	if lastEquity != simulator.equity {
		if err := simulator.router.Post(bus.EquityEvent, &simulator.equity); err != nil {
			simulator.logger.Error("unable to post equity event", zap.Error(err))
		}
	}

	simulator.audit.SnapshotAccount(simulator.balance, simulator.equity, simulator.simulationTime)

	if err := simulator.router.Post(bus.TickEvent, tick); err != nil {
		simulator.logger.Error("unable to post tick event", zap.Error(err))
	}

	if err := simulator.aggregator.OnTick(tick); err != nil {
		simulator.logger.Warn("unable to aggregate ticks", zap.Error(err))
	}

	return nil
}

func (simulator *Simulator) checkPositions(tick *model.Tick) {

	for idx := range simulator.openPositions {
		position := simulator.openPositions[idx]

		if simulator.shouldClosePosition(position, tick) {
			position.State = model.PendingClose
		}
	}
}

func (simulator *Simulator) checkOrders(tick *model.Tick) {

	tmpOpenOrders := make([]*model.Order, 0, len(simulator.openOrders))

	for idx := range simulator.openOrders {
		order := simulator.openOrders[idx]

		switch order.Command {
		case model.CmdOpen:
			switch order.OrderType {
			case model.Market:
				if err := simulator.executeOpenOrder(order.Size, order.StopLoss, order.TakeProfit); err != nil {
					simulator.logger.Warn("unable to execute open order", zap.Error(err))
				}
			case model.Limit:
				if !simulator.shouldOpenPosition(order.Price, order.Size, tick) {
					tmpOpenOrders = append(tmpOpenOrders, order)
					continue
				}
				if err := simulator.executeOpenOrder(order.Size, order.StopLoss, order.TakeProfit); err != nil {
					simulator.logger.Warn("unable to execute open order", zap.Error(err))
				}
			}
		case model.CmdClose:
			if err := simulator.executeCloseOrder(order.PositionId); err != nil {
				simulator.logger.Warn("unable to execute close order", zap.Error(err))
			}
		case model.CmdModify:
			if err := simulator.modifyPosition(order.PositionId, order.StopLoss, order.TakeProfit); err != nil {
				simulator.logger.Warn("unable to modify open position", zap.Error(err))
			}
		case model.CmdRemove:
			continue
		default:
			simulator.logger.Warn("unknown command", zap.Any("cmd", order.Command))
		}
	}

	simulator.openOrders = tmpOpenOrders
}

func (simulator *Simulator) executeCloseOrder(id model.PositionId) error {

	for idx := range simulator.openPositions {
		position := simulator.openPositions[idx]

		if position.Id == id {
			position.State = model.PendingClose
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", id)
}

func (simulator *Simulator) executeOpenOrder(size, stopLoss, takeProfit utility.Fixed) error {

	simulator.positionIdCounter++
	simulator.openPositions = append(simulator.openPositions, &model.Position{
		Id:         simulator.positionIdCounter,
		State:      model.PendingOpen,
		Size:       size,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
	})
	return nil
}

func (simulator *Simulator) modifyPosition(id model.PositionId, stopLoss, takeProfit utility.Fixed) error {

	for idx := range simulator.openPositions {
		position := simulator.openPositions[idx]

		if position.Id == id {
			position.StopLoss = stopLoss
			position.TakeProfit = takeProfit
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", id)
}

func (simulator *Simulator) shouldOpenPosition(price, size utility.Fixed, tick *model.Tick) bool {

	if size.Value > 0 {
		// Long, check if price reached Ask
		if price.Gt(tick.Ask) {
			return true
		}
	} else if size.Value < 0 {
		// Short, check if price reached Bid
		if price.Lt(tick.Bid) {
			return true
		}
	}
	return false
}

func (simulator *Simulator) shouldClosePosition(position *model.Position, tick *model.Tick) bool {

	if position.IsLong() {
		// Long, check if take profit or stop loss has been reached
		if (position.TakeProfit.Value != 0 && position.TakeProfit.Gte(tick.Bid)) ||
			(position.StopLoss.Value != 0 && position.StopLoss.Lte(tick.Bid)) {
			return true
		}
	} else if position.IsShort() {
		// Short, check if take profit or stop loss has been reached
		if (position.TakeProfit.Value != 0 && position.TakeProfit.Lte(tick.Ask)) ||
			(position.StopLoss.Value != 0 && position.StopLoss.Gte(tick.Ask)) {
			return true
		}
	}
	return false
}

func (simulator *Simulator) processPendingChanges(tick *model.Tick) {

	tmpOpenPositions := make([]*model.Position, 0, len(simulator.openPositions))

	for idx := range simulator.openPositions {
		position := simulator.openPositions[idx]

		openPrice := tick.Ask
		closePrice := tick.Bid
		if position.Size.Value < 0 {
			openPrice = tick.Bid
			closePrice = tick.Ask
		}

		simulator.calcPositionProfits(position, closePrice)
		simulator.equity = simulator.equity.Add(position.NetProfit)

		switch position.State {
		case model.PendingOpen:
			tmpOpenPositions = append(tmpOpenPositions, position)
			position.State = model.Opened
			position.OpenPrice = openPrice
			position.OpenTime = time.Unix(0, tick.TimeStamp)
			if err := simulator.router.Post(bus.PositionOpenedEvent, position); err != nil {
				simulator.logger.Warn("unable to post position opened event", zap.Error(err))
			}
		case model.PendingClose:
			position.State = model.Closed
			position.ClosePrice = closePrice
			position.CloseTime = time.Unix(0, tick.TimeStamp)
			simulator.balance = simulator.balance.Add(position.NetProfit)
			simulator.audit.AddClosedPosition(*position)
			if err := simulator.router.Post(bus.PositionClosedEvent, position); err != nil {
				simulator.logger.Warn("unable to post position closed event", zap.Error(err))
			}
		default:
			tmpOpenPositions = append(tmpOpenPositions, position)
			if err := simulator.router.Post(bus.PositionPnLUpdatedEvent, position); err != nil {
				simulator.logger.Warn("unable to post position pnl updated event", zap.Error(err))
			}
		}
	}

	simulator.openPositions = tmpOpenPositions
}

func (simulator *Simulator) calcPositionProfits(position *model.Position, closePrice utility.Fixed) {

	if position.IsLong() {
		position.PipPnL = closePrice.Sub(position.OpenPrice)
	} else if position.IsShort() {
		position.PipPnL = position.OpenPrice.Sub(closePrice)
	}

	position.PipPnL = position.PipPnL.Sub(simulator.slippage.MulInt(2))
	position.GrossProfit = position.PipPnL.Div(simulator.pipSize).Mul(position.Size.Abs()).Mul(simulator.lotValue).Mul(simulator.pipSize)
	position.NetProfit = position.GrossProfit.Sub(simulator.commissions.MulInt(2))
}
