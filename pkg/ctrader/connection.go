package ctrader

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"github.com/peter-kozarec/equinox/pkg/ctrader/openapi"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"strings"
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
	conn   net.Conn
	logger *zap.Logger

	ctx       context.Context
	ctxCancel context.CancelFunc

	writeChan chan *openapi.ProtoMessage
	msgQueue  chan *openapi.ProtoMessage

	pending       sync.Map // map[uint64]chan openapi.ProtoMessage
	subscribersMu sync.RWMutex
	subscribers   map[openapi.ProtoOAPayloadType][]chan *openapi.ProtoMessage
}

func newConnection(conn net.Conn, logger *zap.Logger) *connection {
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
			header := make([]byte, 4)
			if _, err := io.ReadFull(c.conn, header); err != nil {
				if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
					c.logger.Debug("connection closed, exiting read loop")
					time.Sleep(500 * time.Millisecond)
					return
				}
				c.logger.Warn("read header failed", zap.Error(err))
				time.Sleep(500 * time.Millisecond)
				continue
			}
			length := binary.BigEndian.Uint32(header)
			if length == 0 || length > 10_000_000 { // sanity check
				c.logger.Warn("invalid message length", zap.Uint32("length", length))
				time.Sleep(100 * time.Millisecond)
				continue
			}

			data := make([]byte, length)
			if _, err := io.ReadFull(c.conn, data); err != nil {
				c.logger.Warn("cannot read data", zap.Error(err))
				time.Sleep(1 * time.Second) // prevent tight loop
				continue
			}

			var msg openapi.ProtoMessage
			if err := proto.Unmarshal(data, &msg); err != nil {
				c.logger.Warn("unmarshal failed",
					zap.String("raw", hex.EncodeToString(data)),
					zap.Error(err))
				time.Sleep(100 * time.Millisecond)
				continue
			}

			payloadType := openapi.ProtoOAPayloadType(*msg.PayloadType)
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
				zap.String("type", openapi.ProtoOAPayloadType(*msg.PayloadType).String()),
				zap.String("payload", hex.EncodeToString(msg.GetPayload())))

			data, err := proto.Marshal(msg)
			if err != nil {
				c.logger.Warn("failed to marshal message", zap.Error(err))
				continue
			}

			full := make([]byte, 4+len(data))
			binary.BigEndian.PutUint32(full[:4], uint32(len(data)))
			copy(full[4:], data)

			if tcpConn, ok := c.conn.(*net.TCPConn); ok {
				err := tcpConn.SetWriteDeadline(time.Now().Add(time.Second))
				if err != nil {
					c.logger.Warn("failed to set write deadline", zap.Error(err))
				}
			} else if c.conn != nil {
				err := c.conn.SetWriteDeadline(time.Now().Add(time.Second))
				if err != nil {
					c.logger.Warn("failed to set write deadline", zap.Error(err))
				}
			}

			if _, err = c.conn.Write(full); err != nil {
				c.logger.Warn("failed to write to connection", zap.Error(err))
				time.Sleep(1 * time.Second) // prevent tight loop
				continue
			}
		}
	}
}
