package ctrader

import (
	"context"
	"fmt"
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

	authAccCtx, authAccCancel := context.WithTimeout(ctx, time.Second*5)
	defer authAccCancel()

	if err := client.AuthorizeAccount(authAccCtx, int64(accountId), accessToken); err != nil {
		return fmt.Errorf("unable to authorize account: %w", err)
	}

	return nil
}

func Subscribe(ctx context.Context, client *Client, accountId int64, symbol string, period time.Duration, router *bus.Router) error {

	symbolInfo, err := client.GetSymbolInfo(ctx, accountId, symbol)
	if err != nil {
		return fmt.Errorf("unable to get %s symbol info: %w", symbol, err)
	}

	if err := client.SubscribeSpots(ctx, accountId, symbolInfo, period, func(*openapi.ProtoMessage) {}); err != nil {
		return fmt.Errorf("unable to subscribe to %s: %w", symbol, err)
	}

	return nil
}
