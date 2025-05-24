package ctrader

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/ctrader/openapi"
	"peter-kozarec/equinox/internal/model"
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
	router *bus.Router) (func(*model.Order) error, error) {

	symbolInfoContext, symbolInfoCancel := context.WithTimeout(ctx, time.Second)
	defer symbolInfoCancel()

	// Get info about symbol
	symbolInfo, err := client.GetSymbolInfo(symbolInfoContext, accountId, symbol)
	if err != nil {
		return nil, fmt.Errorf("unable to get %s symbol info: %w", symbol, err)
	}
	symbolInfo.Digits = 5
	client.logger.Info("symbol info retrieved", zap.String("symbol", symbol), zap.Int64("id", symbolInfo.Id), zap.Int("digits", symbolInfo.Digits))

	// Create internal state
	state := NewState(router, client.logger, symbolInfo, period)

	loadPosContext, loadPosCancel := context.WithTimeout(ctx, time.Second)
	defer loadPosCancel()

	if err := state.LoadOpenPositions(loadPosContext, client, accountId); err != nil {
		return nil, fmt.Errorf("unable to load open positions: %w", err)
	}
	client.logger.Info("open positions loaded")

	spotsContext, spotsCancel := context.WithTimeout(ctx, time.Second)
	defer spotsCancel()

	// Subscribe to spot events
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

	return func(order *model.Order) error {
		if order.Command == model.CmdClose {
			closeContext, closeCancel := context.WithTimeout(ctx, time.Second)
			defer closeCancel()

			if err := client.ClosePosition(closeContext, accountId, order.PositionId.Int64(), order.Size); err != nil {
				return fmt.Errorf("unable to close position: %w", err)
			}
		} else if order.Command == model.CmdOpen {
			openContext, openCancel := context.WithTimeout(ctx, time.Second)
			defer openCancel()

			if err := client.OpenPosition(openContext, accountId, symbolInfo, order.Price, order.Size, order.Size, order.TakeProfit, order.OrderType); err != nil {
				return fmt.Errorf("unable to open position: %w", err)
			}
		}

		return nil
	}, nil
}
