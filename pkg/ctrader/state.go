package ctrader

import (
	"context"
	"errors"
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/ctrader/openapi"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

type State struct {
	router *bus.Router
	logger *zap.Logger

	instrument common.Instrument
	barPeriod  time.Duration

	lastTick common.Tick
	lastBar  common.Bar

	openPositions []common.Position

	balanceMu   sync.Mutex
	postBalance bool
	balance     fixed.Point
	equity      fixed.Point
}

func NewState(router *bus.Router, logger *zap.Logger, instrument common.Instrument, barPeriod time.Duration) *State {
	return &State{
		router:      router,
		logger:      logger,
		instrument:  instrument,
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

	internalTick := common.Tick{}
	internalTick.Ask = fixed.FromInt64(int64(v.GetAsk()), state.instrument.Digits)
	internalTick.Bid = fixed.FromInt64(int64(v.GetBid()), state.instrument.Digits)
	internalTick.TimeStamp = v.GetTimestamp() * 1000

	if internalTick.Ask.Eq(fixed.Zero) {
		if state.lastTick.Ask.Eq(fixed.Zero) {
			return
		}
		internalTick.Ask = state.lastTick.Ask
	}
	if internalTick.Bid.Eq(fixed.Zero) {
		if state.lastTick.Bid.Eq(fixed.Zero) {
			return
		}
		internalTick.Bid = state.lastTick.Bid
	}

	if err := state.router.Post(bus.TickEvent, internalTick); err != nil {
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
		if err := state.router.Post(bus.BarEvent, state.lastBar); err != nil {
			state.logger.Warn("unable to post bar event", zap.Error(err))
			return
		}
	}

	var internalBar common.Bar
	internalBar.Period = state.barPeriod
	internalBar.TimeStamp = lastBarTimeStamp
	internalBar.Low = fixed.FromInt64(lastBar.GetLow(), state.instrument.Digits)
	internalBar.High = internalBar.Low.Add(fixed.FromInt64(int64(lastBar.GetDeltaHigh()), state.instrument.Digits))
	internalBar.Close = internalBar.Low.Add(fixed.FromInt64(int64(lastBar.GetDeltaClose()), state.instrument.Digits))
	internalBar.Open = internalBar.Low.Add(fixed.FromInt64(int64(lastBar.GetDeltaOpen()), state.instrument.Digits))
	internalBar.Volume = fixed.FromInt64(lastBar.GetVolume(), 0)
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

				internalPosition.State = common.Closed
				internalPosition.CloseTime = time.UnixMilli(int64(*position.TradeData.CloseTimestamp))

				// This is just approximation - not real closing price
				if internalPosition.IsLong() {
					internalPosition.ClosePrice = state.lastTick.Bid
				} else if internalPosition.IsShort() {
					internalPosition.ClosePrice = state.lastTick.Ask
				}

				// Remove the closed position
				state.openPositions = append(state.openPositions[:idx], state.openPositions[idx+1:]...)

				if err := state.router.Post(bus.PositionClosedEvent, *internalPosition); err != nil {
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
		var internalPosition common.Position

		internalPosition.Id = common.PositionId(position.GetPositionId())
		internalPosition.OpenTime = time.UnixMilli(*position.TradeData.OpenTimestamp)
		internalPosition.OpenPrice = fixed.FromFloat64(position.GetPrice())
		internalPosition.State = common.PendingOpen
		internalPosition.StopLoss = fixed.FromFloat64(position.GetStopLoss())
		internalPosition.TakeProfit = fixed.FromFloat64(position.GetTakeProfit())
		internalPosition.Size = fixed.FromInt64(position.TradeData.GetVolume(), 2)

		if position.TradeData.GetTradeSide() == openapi.ProtoOATradeSide_SELL {
			internalPosition.Size = internalPosition.Size.MulInt(-1)
		}

		state.openPositions = append(state.openPositions, internalPosition)

		if err := state.router.Post(bus.PositionOpenedEvent, internalPosition); err != nil {
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

		var internalPosition common.Position

		internalPosition.Id = common.PositionId(position.GetPositionId())
		internalPosition.OpenTime = time.UnixMilli(*position.TradeData.OpenTimestamp)
		internalPosition.OpenPrice = fixed.FromFloat64(position.GetPrice())
		internalPosition.State = common.Opened
		internalPosition.StopLoss = fixed.FromFloat64(position.GetStopLoss())
		internalPosition.TakeProfit = fixed.FromFloat64(position.GetTakeProfit())
		internalPosition.Size = fixed.FromInt64(position.TradeData.GetVolume(), 2)

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
	// Ensure minimum timeout of 5 seconds, but allow longer for slow poll intervals
	requestTimeout := 5 * time.Second
	if pollInterval > 10*time.Second {
		requestTimeout = pollInterval / 2
	}

	ticker := time.NewTicker(pollInterval)

	go func() {
		defer ticker.Stop()

	outer:
		for {
			select {
			case <-parentCtx.Done():
				break outer
			case <-ticker.C:
				// Use the calculated timeout, not the poll interval
				balanceCtx, cancel := context.WithTimeout(parentCtx, requestTimeout)

				balance, err := client.GetBalance(balanceCtx, accountId)
				if err != nil {
					// Distinguish between timeout and other errors for better debugging
					if errors.Is(balanceCtx.Err(), context.DeadlineExceeded) {
						state.logger.Warn("balance poll timed out",
							zap.Duration("timeout", requestTimeout),
							zap.Error(err))
					} else {
						state.logger.Warn("unable to poll balance", zap.Error(err))
					}
				} else {
					state.setBalance(balance)
				}

				cancel() // Always clean up the context
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
			position.NetProfit = state.lastTick.Bid.Sub(position.OpenPrice).Mul(state.instrument.LotSize).Mul(position.Size.Abs())
		} else if position.IsShort() {
			position.NetProfit = position.OpenPrice.Sub(state.lastTick.Ask).Mul(state.instrument.LotSize).Mul(position.Size.Abs())
		}

		// ToDo: Calc gross profit as well

		// Only post event if profit changed
		if !oldProfit.Eq(position.NetProfit) {
			if err := state.router.Post(bus.PositionPnLUpdatedEvent, *position); err != nil {
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
		if err := state.router.Post(bus.EquityEvent, state.equity); err != nil {
			state.logger.Warn("unable to post equity event", zap.Error(err))
		}
	}
}

func (state *State) setBalance(newBalance fixed.Point) {
	state.balanceMu.Lock()
	state.balance = newBalance
	if state.postBalance {
		state.postBalance = false
		if err := state.router.Post(bus.BalanceEvent, state.balance); err != nil {
			state.logger.Warn("unable to post balance event", zap.Error(err))
		}
	}
	state.balanceMu.Unlock()
}

func (state *State) getBalance(balance *fixed.Point) {
	state.balanceMu.Lock()
	*balance = state.balance
	state.balanceMu.Unlock()
}
