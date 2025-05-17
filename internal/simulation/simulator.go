package simulation

import (
	"fmt"
	"github.com/govalues/decimal"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
	"time"
)

type Simulator struct {
	logger     *zap.Logger
	router     *bus.Router
	aggregator *Aggregator
	audit      *Audit

	equity  model.Equity
	balance model.Balance

	SimulationTime time.Time

	positionIdCounter model.PositionId
	openPositions     []model.Position
	openOrders        []model.Order
}

func NewSimulator(logger *zap.Logger, router *bus.Router) *Simulator {
	simulator := &Simulator{
		logger:     logger,
		router:     router,
		aggregator: NewAggregator(BarPeriod, router),
		audit:      NewAudit(logger),
	}

	balance, err := decimal.NewFromFloat64(StartingBalance)
	if err != nil {
		panic(err)
	}

	simulator.balance = model.Balance(balance)
	simulator.equity = model.Equity(balance)

	return simulator
}

func (simulator *Simulator) OnOrder(order *model.Order) error {
	simulator.openOrders = append(simulator.openOrders, *order)
	return nil
}

func (simulator *Simulator) OnTick(tick *model.Tick) error {

	// Set simulation time from processed tick
	simulator.SimulationTime = time.Unix(0, tick.TimeStamp)

	// Store balance and equity before processing the tick
	lastBalance := simulator.balance
	lastEquity := simulator.equity

	// Check open positions
	if err := simulator.checkPositions(tick); err != nil {
		return fmt.Errorf("error checking positions: %w", err)
	}

	// Check open orders
	if err := simulator.checkOrders(tick); err != nil {
		return fmt.Errorf("error checking orders: %w", err)
	}

	// Process pending changes
	if err := simulator.processPendingChanges(tick); err != nil {
		return fmt.Errorf("error processing pending changes: %w", err)
	}

	// Post balance event if the current balance changed after the tick was processed
	if lastBalance != simulator.balance {
		if err := simulator.router.Post(bus.BalanceEvent, &simulator.balance); err != nil {
			return fmt.Errorf("post balance event: %w", err)
		}
	}
	// Post equity event if the current equity changed after the tick was processed
	if lastEquity != simulator.equity {
		if err := simulator.router.Post(bus.EquityEvent, &simulator.equity); err != nil {
			return fmt.Errorf("post equity event: %w", err)
		}
	}

	// Snapshot the balance and equity
	simulator.audit.SnapshotAccountState(simulator.balance, simulator.equity, simulator.SimulationTime)

	// Post the tick
	if err := simulator.router.Post(bus.TickEvent, tick); err != nil {
		return fmt.Errorf("post tick event: %w", err)
	}

	// Aggregate into bars
	if err := simulator.aggregator.OnTick(tick); err != nil {
		return fmt.Errorf("error aggregating ticks: %w", err)
	}

	return nil
}

func (simulator *Simulator) checkPositions(tick *model.Tick) error {

	for idx := range simulator.openPositions {
		position := &simulator.openPositions[idx]

		if simulator.shouldClosePosition(position, tick) {
			position.State = model.PendingClose
		}
	}

	return nil
}

func (simulator *Simulator) checkOrders(tick *model.Tick) error {

	tmpOpenOrders := make([]model.Order, 0, len(simulator.openOrders))
	defer func() {
		simulator.openOrders = tmpOpenOrders
	}()

	for idx := range simulator.openOrders {
		order := &simulator.openOrders[idx]

		switch order.Command {
		case model.CmdOpen:
			switch order.OrderType {
			case model.Market:
				if err := simulator.executeOpenOrder(order.Size, order.StopLoss, order.TakeProfit); err != nil {
					return fmt.Errorf("error executing open order: %w", err)
				}
			case model.Limit:
				if !simulator.shouldOpenPosition(order.Price, order.Size, tick) {
					tmpOpenOrders = append(tmpOpenOrders, *order)
					continue
				}
				if err := simulator.executeOpenOrder(order.Size, order.StopLoss, order.TakeProfit); err != nil {
					return fmt.Errorf("error executing open order: %w", err)
				}
			}
		case model.CmdClose:
			if err := simulator.executeCloseOrder(order.PositionId); err != nil {
				return fmt.Errorf("unable to close position: %w", err)
			}
		case model.CmdModify:
			if err := simulator.modifyPosition(order.PositionId, order.StopLoss, order.TakeProfit); err != nil {
				return fmt.Errorf("unable to modify position: %w", err)
			}
		case model.CmdRemove:
			continue
		default:
			return fmt.Errorf("unknown order command: %d", order.Command)
		}
	}

	return nil
}

func (simulator *Simulator) executeCloseOrder(id model.PositionId) error {

	for idx := range simulator.openPositions {
		position := &simulator.openPositions[idx]

		if position.Id == id {
			position.State = model.PendingClose
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", id)
}

func (simulator *Simulator) executeOpenOrder(size model.Size, stopLoss, takeProfit model.Price) error {

	simulator.positionIdCounter++
	simulator.openPositions = append(simulator.openPositions, model.Position{
		Id:         simulator.positionIdCounter,
		State:      model.PendingOpen,
		Size:       size,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
	})
	return nil
}

func (simulator *Simulator) modifyPosition(id model.PositionId, stopLoss, takeProfit model.Price) error {

	for idx := range simulator.openPositions {
		position := &simulator.openPositions[idx]

		if position.Id == id {
			position.StopLoss = stopLoss
			position.TakeProfit = takeProfit
			return nil
		}
	}
	return fmt.Errorf("position with id %d not found", id)
}

func (simulator *Simulator) shouldOpenPosition(price model.Price, size model.Size, tick *model.Tick) bool {

	if size > 0 {
		// Long, check if price reached Ask
		if price >= tick.Ask {
			return true
		}
	} else if size < 0 {
		// Short, check if price reached Bid
		if price <= tick.Bid {
			return true
		}
	}
	return false
}

func (simulator *Simulator) shouldClosePosition(position *model.Position, tick *model.Tick) bool {

	if position.Size > 0 {
		// Long, check if take profit or stop loss has been reached
		if (position.TakeProfit != 0 && position.TakeProfit >= tick.Bid) ||
			(position.StopLoss != 0 && position.StopLoss <= tick.Bid) {
			return true
		}
	} else if position.Size < 0 {
		// Short, check if take profit or stop loss has been reached
		if (position.TakeProfit != 0 && position.TakeProfit <= tick.Ask) ||
			(position.StopLoss != 0 && position.StopLoss >= tick.Ask) {
			return true
		}
	}
	return false
}

func (simulator *Simulator) processPendingChanges(tick *model.Tick) error {

	tmpOpenPositions := make([]model.Position, 0, len(simulator.openPositions))
	defer func() {
		simulator.openPositions = tmpOpenPositions
	}()

	for idx := range simulator.openPositions {
		position := &simulator.openPositions[idx]

		openPrice := tick.Ask
		closePrice := tick.Bid
		if position.Size < 0 {
			openPrice = tick.Bid
			closePrice = tick.Ask
		}

		// Calculate PnL
		if position.Size > 0 {
			position.PnL = closePrice - position.OpenPrice
		} else if position.Size < 0 {
			position.PnL = position.OpenPrice - closePrice
		}

		switch position.State {
		case model.PendingOpen:
			position.State = model.Opened
			position.OpenPrice = openPrice
			position.OpenTime = time.Unix(0, tick.TimeStamp)
			if err := simulator.router.Post(bus.PositionOpenedEvent, position); err != nil {
				return fmt.Errorf("error posting position opened event: %w", err)
			}
		case model.PendingClose:
			position.State = model.Closed
			position.ClosePrice = closePrice
			position.CloseTime = time.Unix(0, tick.TimeStamp)
			simulator.audit.ClosePosition(*position)
			if err := simulator.router.Post(bus.PositionClosedEvent, position); err != nil {
				return fmt.Errorf("error posting position closed event: %w", err)
			}
		default:
		}

		if err := simulator.router.Post(bus.PositionPnLUpdatedEvent, position); err != nil {
			return fmt.Errorf("error posting position pnl updated event: %w", err)
		}
	}

	return nil
}
