package ctrader

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/peter-kozarec/equinox/pkg/common"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/ctrader/openapi"
)

func Authenticate(
	ctx context.Context,
	client *Client,
	accountId int64,
	accessToken, appId, appSecret string) error {

	authAppCtx, authAppCancel := context.WithTimeout(ctx, time.Second*5)
	defer authAppCancel()

	if err := client.AuthorizeApplication(authAppCtx, appId, appSecret); err != nil {
		return fmt.Errorf("unable to authorize application: %w", err)
	}
	slog.Info("application authorized")

	authAccCtx, authAccCancel := context.WithTimeout(ctx, time.Second*5)
	defer authAccCancel()

	if err := client.AuthorizeAccount(authAccCtx, accountId, accessToken); err != nil {
		return fmt.Errorf("unable to authorize account: %w", err)
	}
	slog.Info("account authorized")

	return nil
}

func InitTradeSession(
	ctx context.Context,
	client *Client,
	accountId int64,
	symbol string,
	router *bus.Router) (func(common.Order), error) {

	symbolInfoContext, symbolInfoCancel := context.WithTimeout(ctx, time.Second)
	defer symbolInfoCancel()

	// Get info about symbol
	symbolInfo, err := client.GetSymbolInfo(symbolInfoContext, accountId, symbol)
	if err != nil {
		return nil, fmt.Errorf("unable to get %s symbol info: %w", symbol, err)
	}
	symbolInfo.Digits = 5
	slog.Info("info",
		"symbol", symbol,
		"id", symbolInfo.Id,
		"digits", symbolInfo.Digits,
		"lot_size", symbolInfo.ContractSize.String(),
		"denomination_unit", symbolInfo.DenominationUnit)

	// Create internal state
	state := NewState(router, symbolInfo)

	// Load balance
	balanceContext, balanceCancel := context.WithTimeout(ctx, time.Second)
	defer balanceCancel()
	if err := state.LoadBalance(balanceContext, client, accountId); err != nil {
		return nil, fmt.Errorf("unable to load balance: %w", err)
	}
	slog.Info("account", "balance", state.balance)

	// Load open positions
	loadPosContext, loadPosCancel := context.WithTimeout(ctx, time.Second)
	defer loadPosCancel()
	if err := state.LoadOpenPositions(loadPosContext, client, accountId); err != nil {
		return nil, fmt.Errorf("unable to load open positions: %w", err)
	}
	if len(state.openPositions) > 0 {
		slog.Info("opened positions present", "count", len(state.openPositions))
	} else {
		slog.Info("no opened positions")
	}

	// Subscribe to spot events
	spotsContext, spotsCancel := context.WithTimeout(ctx, time.Second)
	defer spotsCancel()
	if err := client.SubscribeSpots(spotsContext, accountId, symbolInfo, state.OnSpotsEvent); err != nil {
		return nil, fmt.Errorf("unable to subscribe to spot changes for %s: %w", symbol, err)
	}
	slog.Info("subscribed to spot events")

	// Subscribe to execution events
	_, err = subscribe(client.conn, openapi.ProtoOAPayloadType_PROTO_OA_EXECUTION_EVENT, state.OnExecutionEvent)
	if err != nil {
		return nil, fmt.Errorf("unable to subscribe to execution events: %w", err)
	}
	slog.Info("subscribed to execution events")

	// Start balance polling
	state.StartBalancePolling(ctx, client, accountId, time.Second*10)
	slog.Info("started balance polling", "poll_interval", time.Second*10)

	// Return callback for making orders
	return func(order common.Order) {
		switch order.Command {
		case common.OrderCommandPositionClose:
			closeContext, closeCancel := context.WithTimeout(ctx, time.Second)
			defer closeCancel()

			if err := client.ClosePosition(closeContext, accountId, order.PositionId, order.Size); err != nil {
				slog.Warn("unable to close position", "error", err)
			}
		case common.OrderCommandPositionOpen:
			openContext, openCancel := context.WithTimeout(ctx, time.Second)
			defer openCancel()

			if err := client.OpenPosition(openContext, accountId, symbolInfo, order.Price, order.Size, order.StopLoss, order.TakeProfit, order.Type); err != nil {
				slog.Warn("unable to open position", "error", err)
			}
		default:
		}
	}, nil
}
