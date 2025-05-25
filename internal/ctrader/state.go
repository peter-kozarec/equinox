package ctrader

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/ctrader/openapi"
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/utility"
	"sync"
	"time"
)

type State struct {
	router *bus.Router
	logger *zap.Logger

	symbolInfo model.SymbolInfo
	barPeriod  time.Duration

	lastTick model.Tick
	lastBar  model.Bar

	openPositions []model.Position

	balanceMu   sync.Mutex
	postBalance bool
	balance     utility.Fixed
	equity      utility.Fixed
}

func NewState(router *bus.Router, logger *zap.Logger, symbolInfo model.SymbolInfo, barPeriod time.Duration) *State {
	return &State{
		router:      router,
		logger:      logger,
		symbolInfo:  symbolInfo,
		barPeriod:   barPeriod,
		postBalance: true, // Post balance on first poll, then only when position is closed
	}
}

func (state *State) OnSpotsEvent(msg *openapi.ProtoMessage) {
	var v openapi.ProtoOASpotEvent

	err := proto.Unmarshal(msg.GetPayload(), &v)
	if err != nil {
		state.logger.Warn("unable to unmarshal spots event", zap.Error(err))
		return
	}

	internalTick := model.Tick{}
	internalTick.Ask = utility.NewFixedFromUInt(v.GetAsk(), state.symbolInfo.Digits)
	internalTick.Bid = utility.NewFixedFromUInt(v.GetBid(), state.symbolInfo.Digits)
	internalTick.TimeStamp = v.GetTimestamp()

	if internalTick.Ask.Eq(utility.ZeroFixed) {
		if state.lastTick.Ask.Eq(utility.ZeroFixed) {
			return
		}
		internalTick.Ask = state.lastTick.Ask
	}
	if internalTick.Bid.Eq(utility.ZeroFixed) {
		if state.lastTick.Bid.Eq(utility.ZeroFixed) {
			return
		}
		internalTick.Bid = state.lastTick.Bid
	}

	if err := state.router.Post(bus.TickEvent, &internalTick); err != nil {
		state.logger.Warn("unable to post tick event", zap.Error(err))
		return
	} else {
		state.lastTick = internalTick
	}

	// Calculate PnL for open positions
	state.calcPnL()
	// Calculate equity from unrealized PnL
	state.calcEquity()

	if len(v.GetTrendbar()) == 0 {
		return
	}

	lastBar := v.GetTrendbar()[len(v.GetTrendbar())-1]
	lastInternalBarTimeStamp := state.lastBar.TimeStamp
	lastBarTimeStamp := int64(lastBar.GetUtcTimestampInMinutes()) * int64(time.Minute)

	if lastInternalBarTimeStamp != 0 && lastBarTimeStamp != lastInternalBarTimeStamp { // New bar has came, propagate old one
		if err := state.router.Post(bus.BarEvent, &state.lastBar); err != nil {
			state.logger.Warn("unable to post bar event", zap.Error(err))
			return
		}
	}

	var internalBar model.Bar
	internalBar.Period = state.barPeriod
	internalBar.TimeStamp = lastBarTimeStamp
	internalBar.Low = utility.NewFixedFromInt(lastBar.GetLow(), state.symbolInfo.Digits)
	internalBar.High = internalBar.Low.Add(utility.NewFixedFromUInt(lastBar.GetDeltaHigh(), state.symbolInfo.Digits))
	internalBar.Close = internalBar.Low.Add(utility.NewFixedFromUInt(lastBar.GetDeltaClose(), state.symbolInfo.Digits))
	internalBar.Open = internalBar.Low.Add(utility.NewFixedFromUInt(lastBar.GetDeltaOpen(), state.symbolInfo.Digits))
	internalBar.Volume = lastBar.GetVolume()
	state.lastBar = internalBar
}

func (state *State) OnExecutionEvent(msg *openapi.ProtoMessage) {

	var v openapi.ProtoOAExecutionEvent

	if err := proto.Unmarshal(msg.GetPayload(), &v); err != nil {
		state.logger.Warn("unable to unmarshal execution event", zap.Error(err))
		return
	}

	if v.GetExecutionType() != openapi.ProtoOAExecutionType_ORDER_FILLED || v.GetPosition() == nil {
		// Not interested in other execution types
		return
	}

	position := v.GetPosition()

	if position.GetPositionStatus() == openapi.ProtoOAPositionStatus_POSITION_STATUS_CLOSED {
		for idx := range state.openPositions {
			internalPosition := &state.openPositions[idx]

			if internalPosition.Id.Int64() == position.GetPositionId() {

				internalPosition.State = model.Closed
				internalPosition.CloseTime = time.UnixMilli(int64(*position.TradeData.CloseTimestamp))

				// This is just approximation - not real closing price
				if internalPosition.IsLong() {
					internalPosition.ClosePrice = state.lastTick.Bid
				} else if internalPosition.IsShort() {
					internalPosition.ClosePrice = state.lastTick.Ask
				}

				// Remove the closed position
				state.openPositions = append(state.openPositions[:idx], state.openPositions[idx+1:]...)

				if err := state.router.Post(bus.PositionClosedEvent, internalPosition); err != nil {
					state.logger.Warn("unable to post position closed event", zap.Error(err))
				}

				state.balanceMu.Lock()
				state.postBalance = true
				state.balanceMu.Unlock()
				return
			}
		}
		state.logger.Warn("position not found", zap.Int64("id", position.GetPositionId()))

	} else if position.GetPositionStatus() == openapi.ProtoOAPositionStatus_POSITION_STATUS_OPEN {
		// This can be only open
		var internalPosition model.Position

		internalPosition.Id = model.PositionId(position.GetPositionId())
		internalPosition.OpenTime = time.UnixMilli(*position.TradeData.OpenTimestamp)
		internalPosition.OpenPrice = utility.NewFixedFromFloat64(position.GetPrice())
		internalPosition.State = model.PendingOpen
		internalPosition.StopLoss = utility.NewFixedFromFloat64(position.GetStopLoss())
		internalPosition.TakeProfit = utility.NewFixedFromFloat64(position.GetTakeProfit())
		internalPosition.Size = utility.NewFixedFromInt(position.TradeData.GetVolume(), 2)

		if position.TradeData.GetTradeSide() == openapi.ProtoOATradeSide_SELL {
			internalPosition.Size = internalPosition.Size.MulInt(-1)
		}

		state.openPositions = append(state.openPositions, internalPosition)

		if err := state.router.Post(bus.PositionOpenedEvent, &internalPosition); err != nil {
			state.logger.Warn("unable to post position opened event", zap.Error(err))
			return
		}
	}
}

func (state *State) LoadOpenPositions(ctx context.Context, client *Client, accountId int64) error {

	openPositions, err := client.GetOpenPositions(ctx, accountId)
	if err != nil {
		return fmt.Errorf("unable to retrieve open positions: %w", err)
	}

	for _, position := range openPositions {

		var internalPosition model.Position

		internalPosition.Id = model.PositionId(position.GetPositionId())
		internalPosition.OpenTime = time.UnixMilli(*position.TradeData.OpenTimestamp)
		internalPosition.OpenPrice = utility.NewFixedFromFloat64(position.GetPrice())
		internalPosition.State = model.Opened
		internalPosition.StopLoss = utility.NewFixedFromFloat64(position.GetStopLoss())
		internalPosition.TakeProfit = utility.NewFixedFromFloat64(position.GetTakeProfit())
		internalPosition.Size = utility.NewFixedFromInt(position.TradeData.GetVolume(), 2)

		if position.TradeData.GetTradeSide() == openapi.ProtoOATradeSide_SELL {
			internalPosition.Size = internalPosition.Size.MulInt(-1)
		}

		state.openPositions = append(state.openPositions, internalPosition)
	}

	return nil
}

func (state *State) LoadBalance(ctx context.Context, client *Client, accountId int64) error {

	balance, err := client.GetBalance(ctx, accountId)
	if err != nil {
		return fmt.Errorf("unable to retrieve balance: %w", err)
	}

	state.balance = balance
	state.equity = balance
	return nil
}

func (state *State) StartBalancePolling(parentCtx context.Context, client *Client, accountId int64, pollInterval time.Duration) {

	ticker := time.NewTicker(pollInterval)

	go func() {
		defer ticker.Stop()

	outer:
		for {
			select {
			case <-parentCtx.Done():
				break outer
			case <-ticker.C:
				balanceContext, balanceCancel := context.WithTimeout(parentCtx, pollInterval-(time.Millisecond*50))
				balance, err := client.GetBalance(balanceContext, accountId)
				if err != nil {
					state.logger.Warn("unable to poll balance", zap.Error(err))
				} else {
					state.setBalance(balance)
				}
				balanceCancel()
			}
		}

		state.logger.Debug("balance polling stopped", zap.Error(parentCtx.Err()))
	}()
}

func (state *State) calcPnL() {

	for idx := range state.openPositions {
		position := &state.openPositions[idx]

		oldProfit := position.NetProfit

		// This is without commissions
		if position.IsLong() {
			position.NetProfit = state.lastTick.Bid.Sub(position.OpenPrice).Mul(state.symbolInfo.LotSize).Mul(position.Size.Abs())
		} else if position.IsShort() {
			position.NetProfit = position.OpenPrice.Sub(state.lastTick.Ask).Mul(state.symbolInfo.LotSize).Mul(position.Size.Abs())
		}

		// ToDo: Calc gross profit as well

		// Only post event if profit changed
		if !oldProfit.Eq(position.NetProfit) {
			if err := state.router.Post(bus.PositionPnLUpdatedEvent, position); err != nil {
				state.logger.Warn("unable to post position updated event", zap.Error(err))
			}
		}
	}
}

func (state *State) calcEquity() {

	// Reset equity
	oldEquity := state.equity
	state.getBalance(&state.equity)

	// Add unrealized PnL to equity
	for idx := range state.openPositions {
		position := &state.openPositions[idx]

		state.equity = state.equity.Add(position.NetProfit)
	}

	if !oldEquity.Eq(state.equity) {
		if err := state.router.Post(bus.EquityEvent, &state.equity); err != nil {
			state.logger.Warn("unable to post equity event", zap.Error(err))
		}
	}
}

func (state *State) setBalance(newBalance utility.Fixed) {
	state.balanceMu.Lock()
	state.balance = newBalance
	if state.postBalance {
		state.postBalance = false
		if err := state.router.Post(bus.BalanceEvent, &state.balance); err != nil {
			state.logger.Warn("unable to post balance event", zap.Error(err))
		}
	}
	state.balanceMu.Unlock()
}

func (state *State) getBalance(balance *utility.Fixed) {
	state.balanceMu.Lock()
	*balance = state.balance
	state.balanceMu.Unlock()
}
