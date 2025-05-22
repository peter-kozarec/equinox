package ctrader

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"peter-kozarec/equinox/internal/ctrader/openapi"
	"time"
)

type Client struct {
	conn *connection
}

func DialLive() (*Client, error) {
	tcpConn, err := net.DialTimeout("tcp", "live.ctraderapi.com:5035", time.Second*5)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(tcpConn, &tls.Config{
		ServerName: "live.ctraderapi.com",
	})

	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}

	conn := newConnection(tlsConn)
	conn.start()

	return &Client{
		conn: conn,
	}, nil
}

func DialDemo() (*Client, error) {
	tcpConn, err := net.DialTimeout("tcp", "demo.ctraderapi.com:5035", time.Second*5)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(tcpConn, &tls.Config{
		ServerName: "demo.ctraderapi.com",
	})

	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}

	conn := newConnection(tlsConn)
	conn.start()

	return &Client{
		conn: conn,
	}, nil
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

func (client *Client) AuthorizeAccount(ctx context.Context, accountId int64, accessToken string) error {

	req := &openapi.ProtoOAAccountAuthReq{CtidTraderAccountId: &accountId, AccessToken: &accessToken}
	resp := &openapi.ProtoOAAccountAuthRes{}

	if err := sendReceive(ctx, client.conn, req, resp); err != nil {
		return fmt.Errorf("unable to perform account authorization request: %w", err)
	}

	return nil
}

func (client *Client) KeepAlive(interval time.Duration) {
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
