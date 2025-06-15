package simulation

import (
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/model"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
	"time"
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

func (simulator *Simulator) PrintDetails() {
	simulator.logger.Info("simulation details",
		zap.String("slippage", simulator.cfg.PipSlippage.String()),
		zap.String("commissions", simulator.cfg.CommissionPerLot.String()),
		zap.String("contract_size", simulator.cfg.ContractSize.String()),
		zap.String("pip_size", simulator.cfg.PipSize.String()),
		zap.String("aggregator_interval", simulator.aggregator.interval.String()))
}

func (simulator *Simulator) OnOrder(order model.Order) {
	simulator.openOrders = append(simulator.openOrders, &order)
}

func (simulator *Simulator) OnTick(tick model.Tick) error {

	// Set simulation time from processed tick
	simulator.simulationTime = time.Unix(0, tick.TimeStamp)
	simulator.lastTick = tick

	// Store balance and equity before processing the tick
	lastBalance := simulator.balance
	lastEquity := simulator.equity

	simulator.checkPositions(tick)
	simulator.checkOrders(tick)
	simulator.processPendingChanges(tick)

	// Post balance event if the current balance changed after the tick was processed
	if lastBalance != simulator.balance {
		if err := simulator.router.Post(bus.BalanceEvent, simulator.balance); err != nil {
			simulator.logger.Error("unable to post balance event", zap.Error(err))
		}
	}
	// Post equity event if the current equity changed after the tick was processed
	if lastEquity != simulator.equity {
		if err := simulator.router.Post(bus.EquityEvent, simulator.equity); err != nil {
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

func (simulator *Simulator) CloseAllOpenPositions() {

	simulator.equity = simulator.balance

	for idx := range simulator.openPositions {
		position := simulator.openPositions[idx]

		closePrice := simulator.lastTick.Bid
		if position.IsShort() {
			closePrice = simulator.lastTick.Ask
		}

		simulator.calcPositionProfits(position, closePrice)
		simulator.equity = simulator.equity.Add(position.NetProfit)

		position.State = model.Closed
		position.ClosePrice = closePrice
		position.CloseTime = time.Unix(0, simulator.lastTick.TimeStamp)
		simulator.audit.AddClosedPosition(*position)
	}

	simulator.balance = simulator.equity
	simulator.audit.addSnapshot(simulator.balance, simulator.equity, simulator.simulationTime)
}

func (simulator *Simulator) checkPositions(tick model.Tick) {

	for idx := range simulator.openPositions {
		position := simulator.openPositions[idx]

		if simulator.shouldClosePosition(*position, tick) {
			position.State = model.PendingClose
		}
	}
}

func (simulator *Simulator) checkOrders(tick model.Tick) {

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

func (simulator *Simulator) executeOpenOrder(size, stopLoss, takeProfit fixed.Point) error {

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

func (simulator *Simulator) modifyPosition(id model.PositionId, stopLoss, takeProfit fixed.Point) error {

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

func (simulator *Simulator) shouldOpenPosition(price, size fixed.Point, tick model.Tick) bool {

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

func (simulator *Simulator) shouldClosePosition(position model.Position, tick model.Tick) bool {

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

func (simulator *Simulator) processPendingChanges(tick model.Tick) {
	tmpOpenPositions := make([]*model.Position, 0, len(simulator.openPositions))
	simulator.equity = simulator.balance

	for idx := range simulator.openPositions {
		position := simulator.openPositions[idx]

		openPrice := tick.Ask
		closePrice := tick.Bid
		if position.IsShort() {
			openPrice = tick.Bid
			closePrice = tick.Ask
		}
		switch position.State {
		case model.PendingOpen:
			position.State = model.Opened
			position.OpenPrice = openPrice
			position.OpenTime = time.Unix(0, tick.TimeStamp)
			if err := simulator.router.Post(bus.PositionOpenedEvent, *position); err != nil {
				simulator.logger.Warn("unable to post position opened event", zap.Error(err))
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		case model.PendingClose:
			position.State = model.Closed
			position.ClosePrice = closePrice
			position.CloseTime = time.Unix(0, tick.TimeStamp)
			simulator.calcPositionProfits(position, closePrice)
			simulator.balance = simulator.balance.Add(position.NetProfit)
			simulator.audit.AddClosedPosition(*position)
			if err := simulator.router.Post(bus.PositionClosedEvent, *position); err != nil {
				simulator.logger.Warn("unable to post position closed event", zap.Error(err))
			}
		default:
			simulator.calcPositionProfits(position, closePrice)
			simulator.equity = simulator.equity.Add(position.NetProfit)
			if err := simulator.router.Post(bus.PositionPnLUpdatedEvent, *position); err != nil {
				simulator.logger.Warn("unable to post position pnl updated event", zap.Error(err))
			}
			tmpOpenPositions = append(tmpOpenPositions, position)
		}
	}

	simulator.openPositions = tmpOpenPositions
}

func (simulator *Simulator) calcPositionProfits(position *model.Position, closePrice fixed.Point) {
	var pipPnL fixed.Point

	// Calculate price difference in pips
	if position.IsLong() {
		pipPnL = closePrice.Sub(position.OpenPrice)
	} else {
		pipPnL = position.OpenPrice.Sub(closePrice)
	}

	// Apply slippage
	pipPnL = pipPnL.Sub(simulator.cfg.PipSlippage.MulInt64(2))

	// Convert price difference to pips
	pips := pipPnL.Div(simulator.cfg.PipSize)

	// Calculate dynamic lot value based on current close price
	// For EUR/USD: LotValue = PipSize × ContractSize × CurrentRate
	currentLotValue := simulator.cfg.PipSize.Mul(simulator.cfg.ContractSize).Mul(closePrice)

	// Calculate gross profit
	position.GrossProfit = pips.Mul(position.Size.Abs()).Mul(currentLotValue)

	// Commission calculation
	commission := simulator.cfg.CommissionPerLot.MulInt64(2).Mul(position.Size.Abs())
	position.NetProfit = position.GrossProfit.Sub(commission)
}
