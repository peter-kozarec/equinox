package ctrader

import (
	"context"
	"encoding/binary"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"net"
	"peter-kozarec/equinox/pkg/ctrader/openapi"
	"sync"
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
	conn   net.Conn
	logger *zap.Logger

	ctx       context.Context
	ctxCancel context.CancelFunc

	writeChan chan openapi.ProtoMessage
	msgQueue  chan openapi.ProtoMessage

	pending       sync.Map // map[uint64]chan openapi.ProtoMessage
	subscribersMu sync.RWMutex
	subscribers   map[openapi.ProtoOAPayloadType][]chan openapi.ProtoMessage
}

func newConnection(conn net.Conn, logger *zap.Logger) *connection {
	ctx, cancel := context.WithCancel(context.Background())

	c := &connection{
		conn:        conn,
		logger:      logger,
		ctx:         ctx,
		ctxCancel:   cancel,
		writeChan:   make(chan openapi.ProtoMessage),
		msgQueue:    make(chan openapi.ProtoMessage, 1024),
		subscribers: make(map[openapi.ProtoOAPayloadType][]chan openapi.ProtoMessage),
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
			header := make([]byte, 4)
			if _, err := c.conn.Read(header); err != nil {
				continue
			}
			length := binary.BigEndian.Uint32(header)
			if length == 0 {
				continue
			}

			data := make([]byte, length)
			if _, err := c.conn.Read(data); err != nil {
				continue
			}

			var msg openapi.ProtoMessage
			if err := proto.Unmarshal(data, &msg); err != nil {
				continue
			}

			c.logger.Debug("read", zap.String("msg", msg.String()))

			payloadType := openapi.ProtoOAPayloadType(*msg.PayloadType)
			_, isStream := streamMessageTypes[payloadType]

			if isStream {
				c.subscribersMu.RLock()
				for _, ch := range c.subscribers[payloadType] {
					select {
					case ch <- msg:
					default:
					}
				}
				c.subscribersMu.RUnlock()
			} else if msg.ClientMsgId != nil {
				if ch, ok := c.pending.LoadAndDelete(*msg.ClientMsgId); ok {
					ch.(chan openapi.ProtoMessage) <- msg
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
				continue
			}
			c.logger.Debug("write", zap.String("msg", msg.String()))

			data, err := proto.Marshal(&msg)
			if err != nil {
				continue
			}

			full := append(make([]byte, 4), data...)
			binary.BigEndian.PutUint32(full[:4], uint32(len(data)))

			if _, err = c.conn.Write(full); err != nil {
				continue
			}
		}
	}
}
