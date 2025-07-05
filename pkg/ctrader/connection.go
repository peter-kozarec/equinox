package ctrader

import (
	"context"
	"encoding/hex"
	"github.com/gorilla/websocket"
	"github.com/peter-kozarec/equinox/pkg/ctrader/openapi"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"sync"
	"time"
)

var streamMessageTypes = map[openapi.ProtoOAPayloadType]struct{}{
	openapi.ProtoOAPayloadType_PROTO_OA_EXECUTION_EVENT:                  {},
	openapi.ProtoOAPayloadType_PROTO_OA_ORDER_ERROR_EVENT:                {},
	openapi.ProtoOAPayloadType_PROTO_OA_MARGIN_CHANGED_EVENT:             {},
	openapi.ProtoOAPayloadType_PROTO_OA_SYMBOL_CHANGED_EVENT:             {},
	openapi.ProtoOAPayloadType_PROTO_OA_SPOT_EVENT:                       {},
	openapi.ProtoOAPayloadType_PROTO_OA_CLIENT_DISCONNECT_EVENT:          {},
	openapi.ProtoOAPayloadType_PROTO_OA_ACCOUNTS_TOKEN_INVALIDATED_EVENT: {},
	openapi.ProtoOAPayloadType_PROTO_OA_ACCOUNT_DISCONNECT_EVENT:         {},
	openapi.ProtoOAPayloadType_PROTO_OA_TRADER_UPDATE_EVENT:              {},
	openapi.ProtoOAPayloadType_PROTO_OA_DEPTH_EVENT:                      {},
	openapi.ProtoOAPayloadType_PROTO_OA_MARGIN_CALL_UPDATE_EVENT:         {},
	openapi.ProtoOAPayloadType_PROTO_OA_MARGIN_CALL_TRIGGER_EVENT:        {},
	openapi.ProtoOAPayloadType_PROTO_OA_V1_PNL_CHANGE_EVENT:              {},

	// Not a stream types, but this lets the request time out
	openapi.ProtoOAPayloadType_PROTO_OA_ERROR_RES: {},
}

type connection struct {
	conn   *websocket.Conn
	logger *zap.Logger

	ctx       context.Context
	ctxCancel context.CancelFunc

	writeChan chan *openapi.ProtoMessage
	msgQueue  chan *openapi.ProtoMessage

	pending       sync.Map // map[uint64]chan openapi.ProtoMessage
	subscribersMu sync.RWMutex
	subscribers   map[openapi.ProtoOAPayloadType][]chan *openapi.ProtoMessage
}

func newConnection(conn *websocket.Conn, logger *zap.Logger) *connection {
	ctx, cancel := context.WithCancel(context.Background())

	c := &connection{
		conn:        conn,
		logger:      logger,
		ctx:         ctx,
		ctxCancel:   cancel,
		writeChan:   make(chan *openapi.ProtoMessage, 100),
		msgQueue:    make(chan *openapi.ProtoMessage, 1024),
		subscribers: make(map[openapi.ProtoOAPayloadType][]chan *openapi.ProtoMessage),
	}
	return c
}

func (c *connection) start() {
	go c.read()
	go c.write()
}

func (c *connection) stop() {
	c.ctxCancel()
	_ = c.conn.Close()
}

func (c *connection) read() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				c.logger.Warn("cannot read data", zap.Error(err))
				time.Sleep(1 * time.Second) // prevent tight loop
				return
			}

			var msg openapi.ProtoMessage
			if err := proto.Unmarshal(message, &msg); err != nil {
				c.logger.Warn("unmarshal failed",
					zap.String("raw", hex.EncodeToString(message)),
					zap.Error(err))
				time.Sleep(100 * time.Millisecond)
				continue
			}

			payloadType := openapi.ProtoOAPayloadType(*msg.PayloadType) // #nosec G115
			_, isStream := streamMessageTypes[payloadType]

			c.logger.Debug("read",
				zap.String("type", payloadType.String()),
				zap.String("payload", hex.EncodeToString(msg.GetPayload())))

			if isStream {
				c.subscribersMu.RLock()
				for _, ch := range c.subscribers[payloadType] {
					select {
					case ch <- &msg:
					default: // drop if blocked
					}
				}
				c.subscribersMu.RUnlock()
			} else if msg.ClientMsgId != nil {
				if ch, ok := c.pending.LoadAndDelete(*msg.ClientMsgId); ok {
					select {
					case ch.(chan *openapi.ProtoMessage) <- &msg:
					default: // drop if blocked
					}
				}
			}
		}
	}
}

func (c *connection) write() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.writeChan:
			if !ok {
				return // channel closed — stop writing
			}

			c.logger.Debug("write",
				zap.String("type", openapi.ProtoOAPayloadType(msg.GetPayloadType()).String()), // #nosec G115
				zap.String("payload", hex.EncodeToString(msg.GetPayload())))

			data, err := proto.Marshal(msg)
			if err != nil {
				c.logger.Warn("failed to marshal message", zap.Error(err))
				continue
			}

			if err = c.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				c.logger.Warn("failed to write to connection", zap.Error(err))
				time.Sleep(1 * time.Second) // prevent tight loop
				continue
			}
		}
	}
}
