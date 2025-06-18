package ctrader

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/peter-kozarec/equinox/pkg/ctrader/openapi"
	"github.com/peter-kozarec/equinox/pkg/model"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"net"
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

	conn := newConnection(tlsConn, logger)
	conn.start()

	client := &Client{
		conn:   conn,
		logger: logger,
	}

	client.addErrRespHandler()
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

func (client *Client) GetSymbolInfo(ctx context.Context, accountId int64, symbol string) (model.Instrument, error) {

	req := &openapi.ProtoOASymbolsListReq{CtidTraderAccountId: &accountId}
	resp := &openapi.ProtoOASymbolsListRes{}

	if err := sendReceive(ctx, client.conn, req, resp); err != nil {
		return model.Instrument{}, fmt.Errorf("unable to retrieve symbol list: %w", err)
	}

	var instrument model.Instrument

	for _, s := range resp.GetSymbol() {
		if strings.ToUpper(s.GetSymbolName()) == strings.ToUpper(symbol) {
			instrument.Id = s.GetSymbolId()
			break
		}
	}

	if instrument.Id == 0 {
		return model.Instrument{}, fmt.Errorf("unable to retrieve symbol")
	}

	symbolReq := &openapi.ProtoOASymbolByIdReq{CtidTraderAccountId: &accountId, SymbolId: []int64{instrument.Id}}
	symbolResp := &openapi.ProtoOASymbolByIdRes{}

	if err := sendReceive(ctx, client.conn, symbolReq, symbolResp); err != nil {
		return model.Instrument{}, fmt.Errorf("unable to perform symbol by id request: %w", err)
	}

	for _, s := range symbolResp.GetSymbol() {
		if s.GetSymbolId() == instrument.Id {
			instrument.Digits = int(s.GetDigits())
			instrument.LotSize = fixed.New(s.GetLotSize(), 2) // Lot Size is in cents
			instrument.DenominationUnit = s.GetMeasurementUnits()
			return instrument, nil
		}
	}

	return model.Instrument{}, errors.New("symbol not found")
}

func (client *Client) AuthorizeAccount(ctx context.Context, accountId int64, accessToken string) error {

	req := &openapi.ProtoOAAccountAuthReq{CtidTraderAccountId: &accountId, AccessToken: &accessToken}
	resp := &openapi.ProtoOAAccountAuthRes{}

	if err := sendReceive(ctx, client.conn, req, resp); err != nil {
		return fmt.Errorf("unable to perform account authorization request: %w", err)
	}

	return nil
}

func (client *Client) GetBalance(ctx context.Context, accountId int64) (fixed.Point, error) {

	traderReq := &openapi.ProtoOATraderReq{CtidTraderAccountId: &accountId}
	traderResp := &openapi.ProtoOATraderRes{}

	if err := sendReceive(ctx, client.conn, traderReq, traderResp); err != nil {
		return fixed.Point{}, fmt.Errorf("unable to perform trader request: %w", err)
	}

	return fixed.New(*traderResp.Trader.Balance, int(*traderResp.Trader.MoneyDigits)), nil
}

func (client *Client) SubscribeSpots(ctx context.Context, accountId int64, instrument model.Instrument, period time.Duration, cb func(*openapi.ProtoMessage)) error {

	subTimeStamp := true
	spotsReq := &openapi.ProtoOASubscribeSpotsReq{CtidTraderAccountId: &accountId, SymbolId: []int64{instrument.Id}, SubscribeToSpotTimestamp: &subTimeStamp}
	spotsResp := &openapi.ProtoOASubscribeSpotsRes{}

	if err := sendReceive(ctx, client.conn, spotsReq, spotsResp); err != nil {
		return fmt.Errorf("unable to perform subscribe spots request: %w", err)
	}

	periodMinutes := openapi.ProtoOATrendbarPeriod(int32(period.Minutes()))
	barsReq := &openapi.ProtoOASubscribeLiveTrendbarReq{CtidTraderAccountId: &accountId, Period: &periodMinutes, SymbolId: &instrument.Id}
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

func (client *Client) GetOpenPositions(ctx context.Context, accountId int64) ([]*openapi.ProtoOAPosition, error) {

	req := &openapi.ProtoOAReconcileReq{CtidTraderAccountId: &accountId}
	resp := &openapi.ProtoOAReconcileRes{}

	if err := sendReceive(ctx, client.conn, req, resp); err != nil {
		return nil, fmt.Errorf("unable to perform reconcile request: %w", err)
	}

	return resp.GetPosition(), nil
}

func (client *Client) ClosePosition(ctx context.Context, accountId, positionId int64, size fixed.Point) error {

	vol := int64(size.Abs().MulInt(100).Float64())

	req := &openapi.ProtoOAClosePositionReq{
		CtidTraderAccountId: &accountId,
		PositionId:          &positionId,
		Volume:              &vol,
	}

	return send(ctx, client.conn, req)
}

func (client *Client) OpenPosition(
	ctx context.Context,
	accountId int64,
	instrument model.Instrument,
	openPrice, size, stopLoss, takeProfit fixed.Point,
	orderType model.OrderType) error {

	var limitPrice, sl, tp *float64 = nil, nil, nil

	var ot openapi.ProtoOAOrderType
	switch orderType {
	case model.Market:
		ot = openapi.ProtoOAOrderType_MARKET
	case model.Limit:
		ot = openapi.ProtoOAOrderType_LIMIT
		price := openPrice.Float64()
		limitPrice = &price
	}

	var ts openapi.ProtoOATradeSide
	if size.Gt(fixed.Zero) {
		ts = openapi.ProtoOATradeSide_BUY
	} else {
		ts = openapi.ProtoOATradeSide_SELL
	}

	vol := int64(size.Abs().MulInt(100).Float64())

	if !stopLoss.IsZero() {
		slF := stopLoss.Float64()
		sl = &slF
	}
	if !takeProfit.IsZero() {
		tpF := takeProfit.Float64()
		tp = &tpF
	}

	req := &openapi.ProtoOANewOrderReq{
		CtidTraderAccountId: &accountId,
		SymbolId:            &instrument.Id,
		StopLoss:            sl,
		TakeProfit:          tp,
		TradeSide:           &ts,
		OrderType:           &ot,
		Volume:              &vol,
		LimitPrice:          limitPrice,
	}

	return send(ctx, client.conn, req)
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

func (client *Client) addErrRespHandler() {

	_, _ = subscribe(client.conn, openapi.ProtoOAPayloadType_PROTO_OA_ERROR_RES, func(message *openapi.ProtoMessage) {
		var v openapi.ProtoOAErrorRes
		err := proto.Unmarshal(message.GetPayload(), &v)
		if err != nil {
			client.logger.Warn("unable to unmarshal error response", zap.Error(err))
		}
		client.logger.Warn("something went wrong", zap.Any("code", v.GetErrorCode()), zap.Any("description", v.GetDescription()))
	})

	_, _ = subscribe(client.conn, openapi.ProtoOAPayloadType_PROTO_OA_ORDER_ERROR_EVENT, func(message *openapi.ProtoMessage) {
		var v openapi.ProtoOAOrderErrorEvent
		err := proto.Unmarshal(message.GetPayload(), &v)
		if err != nil {
			client.logger.Warn("unable to unmarshal order error response", zap.Error(err))
		}
		client.logger.Warn("unable to process order event", zap.Any("code", v.GetErrorCode()), zap.Any("description", v.GetDescription()))
	})
}
