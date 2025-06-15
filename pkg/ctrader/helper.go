package ctrader

import (
	"context"
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/ctrader/openapi"
	"github.com/peter-kozarec/equinox/pkg/model"
	"go.uber.org/zap"
	"time"
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
	client.logger.Info("application authorized")

	authAccCtx, authAccCancel := context.WithTimeout(ctx, time.Second*5)
	defer authAccCancel()

	if err := client.AuthorizeAccount(authAccCtx, int64(accountId), accessToken); err != nil {
		return fmt.Errorf("unable to authorize account: %w", err)
	}
	client.logger.Info("account authorized")

	return nil
}

func InitTradeSession(
	ctx context.Context,
	client *Client,
	accountId int64,
	symbol string,
	period time.Duration,
	router *bus.Router) (func(*model.Order), error) {

	symbolInfoContext, symbolInfoCancel := context.WithTimeout(ctx, time.Second)
	defer symbolInfoCancel()

	// Get info about symbol
	symbolInfo, err := client.GetSymbolInfo(symbolInfoContext, accountId, symbol)
	if err != nil {
		return nil, fmt.Errorf("unable to get %s symbol info: %w", symbol, err)
	}
	symbolInfo.Digits = 5
	client.logger.Info("symbol info",
		zap.String("symbol", symbol),
		zap.Int64("id", symbolInfo.Id),
		zap.Int("digits", symbolInfo.Digits),
		zap.String("lot_size", symbolInfo.LotSize.String()),
		zap.String("denomination_unit", symbolInfo.DenominationUnit))

	// Create internal state
	state := NewState(router, client.logger, symbolInfo, period)

	// Load balance
	balanceContext, balanceCancel := context.WithTimeout(ctx, time.Second)
	defer balanceCancel()
	if err := state.LoadBalance(balanceContext, client, accountId); err != nil {
		return nil, fmt.Errorf("unable to load balance: %w", err)
	}
	client.logger.Info("account info", zap.String("balance", state.balance.String()))

	// Load open positions
	loadPosContext, loadPosCancel := context.WithTimeout(ctx, time.Second)
	defer loadPosCancel()
	if err := state.LoadOpenPositions(loadPosContext, client, accountId); err != nil {
		return nil, fmt.Errorf("unable to load open positions: %w", err)
	}
	if len(state.openPositions) > 0 {
		client.logger.Info("opened positions present", zap.Int("count", len(state.openPositions)))
	} else {
		client.logger.Info("no opened positions are present")
	}

	// Subscribe to spot events
	spotsContext, spotsCancel := context.WithTimeout(ctx, time.Second)
	defer spotsCancel()
	if err := client.SubscribeSpots(spotsContext, accountId, symbolInfo, period, state.OnSpotsEvent); err != nil {
		return nil, fmt.Errorf("unable to subscribe to spot changes for %s: %w", symbol, err)
	}
	client.logger.Info("subscribed to spot events")

	// Subscribe to execution events
	_, err = subscribe(client.conn, openapi.ProtoOAPayloadType_PROTO_OA_EXECUTION_EVENT, state.OnExecutionEvent)
	if err != nil {
		return nil, fmt.Errorf("unable to subscribe to execution events: %w", err)
	}
	client.logger.Info("subscribed to execution events")

	// Start balance polling
	state.StartBalancePolling(ctx, client, accountId, time.Millisecond*500)
	client.logger.Info("started balance polling", zap.Duration("poll_interval", time.Millisecond*500))

	// Return callback for making orders
	return func(order *model.Order) {
		if order.Command == model.CmdClose {
			closeContext, closeCancel := context.WithTimeout(ctx, time.Second)
			defer closeCancel()

			if err := client.ClosePosition(closeContext, accountId, order.PositionId.Int64(), order.Size); err != nil {
				client.logger.Warn("unable to close position", zap.Error(err))
			}
		} else if order.Command == model.CmdOpen {
			openContext, openCancel := context.WithTimeout(ctx, time.Second)
			defer openCancel()

			if err := client.OpenPosition(openContext, accountId, symbolInfo, order.Price, order.Size, order.StopLoss, order.TakeProfit, order.OrderType); err != nil {
				client.logger.Warn("unable to open position", zap.Error(err))
			}
		}
	}, nil
}
