package ctrader

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gorilla/websocket"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/ctrader/openapi"

	"strings"
	"time"

	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	conn *connection
}

func dial(host, port string) (*Client, error) {

	wsConn, _, err := websocket.DefaultDialer.Dial("wss://"+host+":"+port, nil)
	if err != nil {
		return nil, err
	}

	conn := newConnection(wsConn)
	conn.start()

	client := &Client{
		conn: conn,
	}

	client.addErrRespHandler()
	client.keepAlive(time.Second * 30)
	return client, nil
}

//goland:noinspection GoUnusedExportedFunction
func DialLive() (*Client, error) {
	return dial("live.ctraderapi.com", "5035")
}

func DialDemo() (*Client, error) {
	return dial("demo.ctraderapi.com", "5035")
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

func (client *Client) GetSymbolInfo(ctx context.Context, accountId int64, symbol string) (common.Instrument, error) {

	req := &openapi.ProtoOASymbolsListReq{CtidTraderAccountId: &accountId}
	resp := &openapi.ProtoOASymbolsListRes{}

	if err := sendReceive(ctx, client.conn, req, resp); err != nil {
		return common.Instrument{}, fmt.Errorf("unable to retrieve symbol list: %w", err)
	}

	var instrument common.Instrument

	for _, s := range resp.GetSymbol() {
		if strings.EqualFold(s.GetSymbolName(), symbol) {
			instrument.Id = s.GetSymbolId()
			break
		}
	}

	if instrument.Id == 0 {
		return common.Instrument{}, fmt.Errorf("unable to retrieve symbol")
	}

	symbolReq := &openapi.ProtoOASymbolByIdReq{CtidTraderAccountId: &accountId, SymbolId: []int64{instrument.Id}}
	symbolResp := &openapi.ProtoOASymbolByIdRes{}

	if err := sendReceive(ctx, client.conn, symbolReq, symbolResp); err != nil {
		return common.Instrument{}, fmt.Errorf("unable to perform symbol by id request: %w", err)
	}

	for _, s := range symbolResp.GetSymbol() {
		if s.GetSymbolId() == instrument.Id {
			instrument.Digits = int(s.GetDigits())
			instrument.LotSize = fixed.FromInt64(s.GetLotSize(), 2) // Lot Size is in cents
			instrument.DenominationUnit = s.GetMeasurementUnits()
			return instrument, nil
		}
	}

	return common.Instrument{}, errors.New("symbol not found")
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

	return fixed.FromInt64(*traderResp.Trader.Balance, int(*traderResp.Trader.MoneyDigits)), nil
}

func (client *Client) SubscribeSpots(ctx context.Context, accountId int64, instrument common.Instrument, period time.Duration, cb func(*openapi.ProtoMessage)) error {

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
		slog.Warn("unable to subscribe", "error", err)
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

	vol, _ := size.Abs().MulInt(100).Float64()
	volume := int64(vol)

	req := &openapi.ProtoOAClosePositionReq{
		CtidTraderAccountId: &accountId,
		PositionId:          &positionId,
		Volume:              &volume,
	}

	return send(ctx, client.conn, req)
}

func (client *Client) OpenPosition(
	ctx context.Context,
	accountId int64,
	instrument common.Instrument,
	openPrice, size, stopLoss, takeProfit fixed.Point,
	orderType common.OrderType) error {

	var limitPrice, sl, tp *float64 = nil, nil, nil

	var ot openapi.ProtoOAOrderType
	switch orderType {
	case common.Market:
		ot = openapi.ProtoOAOrderType_MARKET
	case common.Limit:
		ot = openapi.ProtoOAOrderType_LIMIT
		price, _ := openPrice.Float64()
		limitPrice = &price
	}

	var ts openapi.ProtoOATradeSide
	if size.Gt(fixed.Zero) {
		ts = openapi.ProtoOATradeSide_BUY
	} else {
		ts = openapi.ProtoOATradeSide_SELL
	}

	vol, _ := size.Abs().MulInt(100).Float64()
	volume := int64(vol)

	if !stopLoss.IsZero() {
		slF, _ := stopLoss.Float64()
		sl = &slF
	}
	if !takeProfit.IsZero() {
		tpF, _ := takeProfit.Float64()
		tp = &tpF
	}

	req := &openapi.ProtoOANewOrderReq{
		CtidTraderAccountId: &accountId,
		SymbolId:            &instrument.Id,
		StopLoss:            sl,
		TakeProfit:          tp,
		TradeSide:           &ts,
		OrderType:           &ot,
		Volume:              &volume,
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
				msg := &openapi.ProtoMessage{PayloadType: &payloadType}

				select {
				case client.conn.writeChan <- msg:
				default:
					slog.Warn("heartbeat dropped: writeChan full")
				}
			}
		}
	}()
}

func (client *Client) addErrRespHandler() {

	_, _ = subscribe(client.conn, openapi.ProtoOAPayloadType_PROTO_OA_ERROR_RES, func(message *openapi.ProtoMessage) {
		var v openapi.ProtoOAErrorRes
		err := proto.Unmarshal(message.GetPayload(), &v)
		if err != nil {
			slog.Warn("unable to unmarshal error response", "error", err)
		}
		slog.Warn("something went wrong", "code", v.GetErrorCode(), "description", v.GetDescription())
	})

	_, _ = subscribe(client.conn, openapi.ProtoOAPayloadType_PROTO_OA_ORDER_ERROR_EVENT, func(message *openapi.ProtoMessage) {
		var v openapi.ProtoOAOrderErrorEvent
		err := proto.Unmarshal(message.GetPayload(), &v)
		if err != nil {
			slog.Warn("unable to unmarshal order error response", "error", err)
		}
		slog.Warn("unable to process order event", "code", v.GetErrorCode(), "description", v.GetDescription())
	})
}
