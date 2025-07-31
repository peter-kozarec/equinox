package ctrader

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/exchange/ctrader/openapi"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

const (
	openapiComponentName = "exchange.ctrader.openapi"
)

const (
	// Internal position statuses
	positionStatusPendingOpen common.PositionStatus = "pending-open"
)

type State struct {
	router     *bus.Router
	symbolInfo exchange.SymbolInfo

	lastTick common.Tick

	openPositions []common.Position

	balanceMu   sync.Mutex
	postBalance bool
	balance     fixed.Point
	equity      fixed.Point
}

func NewState(router *bus.Router, symbolInfo exchange.SymbolInfo) *State {
	return &State{
		router:      router,
		symbolInfo:  symbolInfo,
		postBalance: true, // Post balance on first poll, then only when position is closed
	}
}

func (state *State) OnSpotsEvent(msg *openapi.ProtoMessage) {
	var v openapi.ProtoOASpotEvent

	err := proto.Unmarshal(msg.GetPayload(), &v)
	if err != nil {
		slog.Warn("unable to unmarshal spots event", "error", err)
		return
	}

	internalTick := common.Tick{}
	internalTick.Symbol = state.symbolInfo.SymbolName
	internalTick.Source = openapiComponentName
	internalTick.ExecutionId = utility.GetExecutionID()
	internalTick.TraceID = utility.CreateTraceID()
	internalTick.Ask = fixed.FromUint64(v.GetAsk(), state.symbolInfo.Digits)
	internalTick.Bid = fixed.FromUint64(v.GetBid(), state.symbolInfo.Digits)
	internalTick.TimeStamp = time.UnixMilli(v.GetTimestamp())

	if internalTick.Ask.Eq(fixed.Zero) {
		if state.lastTick.Ask.Eq(fixed.Zero) {
			return
		}
		internalTick.Ask = state.lastTick.Ask
	} else {
		// Tick volume, basically when ask changes, this is one
		internalTick.AskVolume = fixed.One
	}

	if internalTick.Bid.Eq(fixed.Zero) {
		if state.lastTick.Bid.Eq(fixed.Zero) {
			return
		}
		internalTick.Bid = state.lastTick.Bid
	} else {
		// Tick volume, basically when bid changes, this is one
		internalTick.BidVolume = fixed.One
	}

	if err := state.router.Post(bus.TickEvent, internalTick); err != nil {
		slog.Warn("unable to post tick event", "error", err)
		return
	}

	state.lastTick = internalTick
	state.calcPnL()
	state.calcEquity()
}

func (state *State) OnExecutionEvent(msg *openapi.ProtoMessage) {

	var v openapi.ProtoOAExecutionEvent

	if err := proto.Unmarshal(msg.GetPayload(), &v); err != nil {
		slog.Warn("unable to unmarshal execution event", "error", err)
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

			if internalPosition.Id == position.GetPositionId() {

				internalPosition.Status = common.PositionStatusClosed
				internalPosition.CloseTime = time.UnixMilli(utility.U64ToI64Unsafe(position.TradeData.GetCloseTimestamp()))

				// This is just approximation - not real closing price
				if internalPosition.Side == common.PositionSideLong {
					internalPosition.ClosePrice = state.lastTick.Bid
				} else if internalPosition.Side == common.PositionSideShort {
					internalPosition.ClosePrice = state.lastTick.Ask
				} else {
					panic("invalid internal position side")
				}

				// Remove the closed position
				state.openPositions = append(state.openPositions[:idx], state.openPositions[idx+1:]...)

				if err := state.router.Post(bus.PositionCloseEvent, *internalPosition); err != nil {
					slog.Warn("unable to post position closed event", "error", err)
				}

				state.balanceMu.Lock()
				state.postBalance = true
				state.balanceMu.Unlock()
				return
			}
		}
		slog.Warn("position not found", "id", position.GetPositionId())

	} else if position.GetPositionStatus() == openapi.ProtoOAPositionStatus_POSITION_STATUS_OPEN {
		// This can be only open
		var internalPosition common.Position

		internalPosition.Symbol = state.symbolInfo.SymbolName
		internalPosition.Source = openapiComponentName
		internalPosition.ExecutionID = utility.GetExecutionID()
		internalPosition.TraceID = utility.CreateTraceID()
		internalPosition.TimeStamp = time.Now()
		internalPosition.Side = common.PositionSideLong
		internalPosition.Id = position.GetPositionId()
		internalPosition.OpenTime = time.UnixMilli(*position.TradeData.OpenTimestamp)
		internalPosition.OpenPrice = fixed.FromFloat64(position.GetPrice())
		internalPosition.Status = positionStatusPendingOpen
		internalPosition.StopLoss = fixed.FromFloat64(position.GetStopLoss())
		internalPosition.TakeProfit = fixed.FromFloat64(position.GetTakeProfit())
		internalPosition.Size = fixed.FromInt64(position.TradeData.GetVolume(), 2).Div(state.symbolInfo.ContractSize)
		internalPosition.Commissions = fixed.FromInt64(position.GetCommission(), int(position.GetMoneyDigits())).Abs().MulInt(2)

		if position.TradeData.GetTradeSide() == openapi.ProtoOATradeSide_SELL {
			internalPosition.Size = internalPosition.Size.MulInt(-1)
			internalPosition.Side = common.PositionSideShort
		}

		state.openPositions = append(state.openPositions, internalPosition)

		if err := state.router.Post(bus.PositionOpenEvent, internalPosition); err != nil {
			slog.Warn("unable to post position opened event", "error", err)
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

		internalPosition.Symbol = state.symbolInfo.SymbolName
		internalPosition.Source = openapiComponentName
		internalPosition.ExecutionID = utility.GetExecutionID()
		internalPosition.TraceID = utility.CreateTraceID()
		internalPosition.TimeStamp = time.Now()
		internalPosition.Side = common.PositionSideLong
		internalPosition.Id = position.GetPositionId()
		internalPosition.OpenTime = time.UnixMilli(*position.TradeData.OpenTimestamp)
		internalPosition.OpenPrice = fixed.FromFloat64(position.GetPrice())
		internalPosition.Status = positionStatusPendingOpen
		internalPosition.StopLoss = fixed.FromFloat64(position.GetStopLoss())
		internalPosition.TakeProfit = fixed.FromFloat64(position.GetTakeProfit())
		internalPosition.Size = fixed.FromInt64(position.TradeData.GetVolume(), 2).Div(state.symbolInfo.ContractSize)
		internalPosition.Commissions = fixed.FromInt64(position.GetCommission(), int(position.GetMoneyDigits())).Abs().MulInt(2)

		if position.TradeData.GetTradeSide() == openapi.ProtoOATradeSide_SELL {
			internalPosition.Size = internalPosition.Size.MulInt(-1)
			internalPosition.Side = common.PositionSideShort
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
						slog.Warn("balance poll timed out",
							"timeout", requestTimeout,
							"error", err)
					} else {
						slog.Warn("unable to poll balance", "error", err)
					}
				} else {
					state.setBalance(balance)
				}

				cancel() // Always clean up the context
			}
		}

		slog.Debug("balance polling stopped", "error", parentCtx.Err())
	}()
}

func (state *State) calcPnL() {

	for idx := range state.openPositions {
		position := &state.openPositions[idx]

		oldProfit := position.NetProfit

		// This is without commissions
		if position.Side == common.PositionSideLong {
			position.NetProfit = state.lastTick.Bid.Sub(position.OpenPrice).Mul(state.symbolInfo.ContractSize).Mul(position.Size.Abs())
			position.GrossProfit = position.NetProfit.Add(position.Commissions)
		} else if position.Side == common.PositionSideShort {
			position.NetProfit = position.OpenPrice.Sub(state.lastTick.Ask).Mul(state.symbolInfo.ContractSize).Mul(position.Size.Abs())
			position.GrossProfit = position.NetProfit.Sub(position.Commissions)
		} else {
			panic("invalid position side")
		}

		if !oldProfit.Eq(position.NetProfit) {
			position.TimeStamp = time.Now()
			if err := state.router.Post(bus.PositionUpdateEvent, *position); err != nil {
				slog.Warn("unable to post position updated event", "error", err)
			}
		}
	}
}

func (state *State) calcEquity() {
	oldEquity := state.equity
	state.getBalance(&state.equity)

	for idx := range state.openPositions {
		position := &state.openPositions[idx]

		state.equity = state.equity.Add(position.NetProfit)
	}

	if !oldEquity.Eq(state.equity) {
		if err := state.router.Post(bus.EquityEvent, common.Equity{
			Source:      openapiComponentName,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   time.Now(),
			Value:       state.equity,
		}); err != nil {
			slog.Warn("unable to post equity event", "error", err)
		}
	}
}

func (state *State) setBalance(newBalance fixed.Point) {
	state.balanceMu.Lock()
	state.balance = newBalance
	if state.postBalance {
		state.postBalance = false
		if err := state.router.Post(bus.BalanceEvent, common.Balance{
			Source:      openapiComponentName,
			ExecutionId: utility.GetExecutionID(),
			TraceID:     utility.CreateTraceID(),
			TimeStamp:   time.Now(),
			Value:       state.balance,
		}); err != nil {
			slog.Warn("unable to post balance event", "error", err)
		}
	}
	state.balanceMu.Unlock()
}

func (state *State) getBalance(balance *fixed.Point) {
	state.balanceMu.Lock()
	*balance = state.balance
	state.balanceMu.Unlock()
}
