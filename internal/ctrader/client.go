package ctrader

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"net"
	"peter-kozarec/equinox/internal/ctrader/openapi"
	"peter-kozarec/equinox/internal/model"
	"strings"
	"time"
)

type Client struct {
	conn   *connection
	logger *zap.Logger
}

func dial(logger *zap.Logger, host, port string) (*Client, error) {
	tcpConn, err := net.DialTimeout("tcp", host+":"+port, time.Second*5)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(tcpConn, &tls.Config{
		ServerName: host,
	})

	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}

	conn := newConnection(tlsConn, logger, 50)
	conn.start()

	client := &Client{
		conn:   conn,
		logger: logger,
	}

	client.keepAlive(time.Second * 30)
	return client, nil
}

func DialLive(logger *zap.Logger) (*Client, error) {
	return dial(logger, "live.ctraderapi.com", "5035")
}

func DialDemo(logger *zap.Logger) (*Client, error) {
	return dial(logger, "demo.ctraderapi.com", "5035")
}

func (client *Client) Close() {
	client.conn.stop()
}

func (client *Client) AuthorizeApplication(ctx context.Context, id, secret string) error {

	req := &openapi.ProtoOAApplicationAuthReq{ClientId: &id, ClientSecret: &secret}
	resp := &openapi.ProtoOAApplicationAuthRes{}

	if err := sendReceive(ctx, client.conn, req, resp); err != nil {
		return fmt.Errorf("unable to perform application authorization request: %w", err)
	}

	return nil
}

func (client *Client) GetAccountList(ctx context.Context, accessToken string) ([]*openapi.ProtoOACtidTraderAccount, error) {

	req := &openapi.ProtoOAGetAccountListByAccessTokenReq{AccessToken: &accessToken}
	resp := &openapi.ProtoOAGetAccountListByAccessTokenRes{}

	if err := sendReceive(ctx, client.conn, req, resp); err != nil {
		return nil, fmt.Errorf("unable to retrieve account list: %w", err)
	}

	return resp.GetCtidTraderAccount(), nil
}

func (client *Client) GetSymbolInfo(ctx context.Context, accountId int64, symbol string) (model.Symbol, error) {

	req := &openapi.ProtoOASymbolsListReq{CtidTraderAccountId: &accountId}
	resp := &openapi.ProtoOASymbolsListRes{}

	if err := sendReceive(ctx, client.conn, req, resp); err != nil {
		return model.Symbol{}, fmt.Errorf("unable to retrieve symbol list: %w", err)
	}

	var symbolInfo model.Symbol

	for _, s := range resp.GetSymbol() {
		if strings.ToUpper(s.GetSymbolName()) == strings.ToUpper(symbol) {
			symbolInfo.Id = s.GetSymbolId()
			break
		}
	}

	if symbolInfo.Id == 0 {
		return model.Symbol{}, fmt.Errorf("unable to retrieve symbol")
	}

	symbolReq := &openapi.ProtoOASymbolByIdReq{CtidTraderAccountId: &accountId, SymbolId: []int64{symbolInfo.Id}}
	symbolResp := &openapi.ProtoOASymbolByIdRes{}

	if err := sendReceive(ctx, client.conn, symbolReq, symbolResp); err != nil {
		return model.Symbol{}, fmt.Errorf("unable to perform symbol by id request: %w", err)
	}

	for _, s := range symbolResp.GetSymbol() {
		if s.GetSymbolId() == symbolInfo.Id {
			symbolInfo.Digits = s.GetDigits()
			return symbolInfo, nil
		}
	}

	return model.Symbol{}, errors.New("symbol not found")
}

func (client *Client) AuthorizeAccount(ctx context.Context, accountId int64, accessToken string) error {

	req := &openapi.ProtoOAAccountAuthReq{CtidTraderAccountId: &accountId, AccessToken: &accessToken}
	resp := &openapi.ProtoOAAccountAuthRes{}

	if err := sendReceive(ctx, client.conn, req, resp); err != nil {
		return fmt.Errorf("unable to perform account authorization request: %w", err)
	}

	return nil
}

func (client *Client) SubscribeSpots(ctx context.Context, accountId int64, symbol model.Symbol, period time.Duration, cb func(*openapi.ProtoMessage)) error {

	spotsReq := &openapi.ProtoOASubscribeSpotsReq{CtidTraderAccountId: &accountId, SymbolId: []int64{symbol.Id}}
	spotsResp := &openapi.ProtoOASubscribeSpotsRes{}

	if err := sendReceive(ctx, client.conn, spotsReq, spotsResp); err != nil {
		return fmt.Errorf("unable to perform subscribe spots request: %w", err)
	}

	periodMinutes := openapi.ProtoOATrendbarPeriod(int32(period.Minutes()))
	barsReq := &openapi.ProtoOASubscribeLiveTrendbarReq{CtidTraderAccountId: &accountId, Period: &periodMinutes, SymbolId: &symbol.Id}
	barsResp := &openapi.ProtoOASubscribeLiveTrendbarRes{}

	if err := sendReceive(ctx, client.conn, barsReq, barsResp); err != nil {
		return fmt.Errorf("unable to perform subscribe live bars request: %w", err)
	}

	_, err := subscribe(client.conn, openapi.ProtoOAPayloadType_PROTO_OA_SPOT_EVENT, cb)
	if err != nil {
		client.logger.Warn("unable to subscribe", zap.Error(err))
	}

	return nil
}

func (client *Client) keepAlive(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-client.conn.ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				payloadType := uint32(openapi.ProtoPayloadType_HEARTBEAT_EVENT)
				msg := openapi.ProtoMessage{
					PayloadType: &payloadType,
				}
				client.conn.writeChan <- msg
			}
		}
	}()

}
