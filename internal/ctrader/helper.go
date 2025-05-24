package ctrader

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/ctrader/openapi"
	"time"
)

func Authenticate(ctx context.Context, client *Client, accountId int64, accessToken, appId, appSecret string) error {

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

func Subscribe(ctx context.Context, client *Client, accountId int64, symbol string, period time.Duration, router *bus.Router) error {

	_, err := subscribe(client.conn, openapi.ProtoOAPayloadType_PROTO_OA_ERROR_RES, createErrorResponseCallback(client.logger))
	if err != nil {
		return fmt.Errorf("unable to subscribe to error responses: %w", err)
	}

	symbolInfoContext, symbolInfoCancel := context.WithTimeout(ctx, time.Second)
	defer symbolInfoCancel()

	symbolInfo, err := client.GetSymbolInfo(symbolInfoContext, accountId, symbol)
	if err != nil {
		return fmt.Errorf("unable to get %s symbol info: %w", symbol, err)
	}
	client.logger.Info("symbol info retrieved", zap.String("symbol", symbol), zap.Int64("id", symbolInfo.Id), zap.Int32("digits", symbolInfo.Digits))

	spotsContext, spotsCancel := context.WithTimeout(ctx, time.Second)
	defer spotsCancel()

	if err := client.SubscribeSpots(spotsContext, accountId, symbolInfo, period, createSpotsCallback(router, client.logger, period)); err != nil {
		return fmt.Errorf("unable to subscribe to spot changes for %s: %w", symbol, err)
	}
	client.logger.Info("subscribed to spot events")

	return nil
}
