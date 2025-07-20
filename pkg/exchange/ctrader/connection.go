package ctrader

import (
	"context"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/peter-kozarec/equinox/pkg/exchange/ctrader/openapi"
	"github.com/peter-kozarec/equinox/pkg/utility"
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
	conn *websocket.Conn

	ctx       context.Context
	ctxCancel context.CancelFunc

	writeChan chan *openapi.ProtoMessage
	msgQueue  chan *openapi.ProtoMessage

	pending       sync.Map // map[uint64]chan openapi.ProtoMessage
	subscribersMu sync.RWMutex
	subscribers   map[openapi.ProtoOAPayloadType][]chan *openapi.ProtoMessage
}

func newConnection(conn *websocket.Conn) *connection {
	ctx, cancel := context.WithCancel(context.Background())

	c := &connection{
		conn:        conn,
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
				if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					slog.Debug("websocket closed", "error", err)
					return
				}
				slog.Warn("cannot read data", "error", err)
				time.Sleep(1 * time.Second) // prevent tight loop
				return
			}

			var msg openapi.ProtoMessage
			if err := proto.Unmarshal(message, &msg); err != nil {
				slog.Warn("unmarshal failed",
					"raw", hex.EncodeToString(message),
					"error", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			payloadType := openapi.ProtoOAPayloadType(utility.U32ToI32Unsafe(msg.GetPayloadType()))
			_, isStream := streamMessageTypes[payloadType]

			slog.Debug("read",
				"type", payloadType.String(),
				"payload", hex.EncodeToString(msg.GetPayload()))

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

			slog.Debug("write",
				"type", openapi.ProtoOAPayloadType(utility.U32ToI32Unsafe(msg.GetPayloadType())).String(),
				"payload", hex.EncodeToString(msg.GetPayload()))

			data, err := proto.Marshal(msg)
			if err != nil {
				slog.Warn("failed to marshal message", "error", err)
				continue
			}

			if err = c.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				slog.Warn("failed to write to connection", "error", err)
				time.Sleep(1 * time.Second) // prevent tight loop
				continue
			}
		}
	}
}
