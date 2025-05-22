package ctrader

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"peter-kozarec/equinox/internal/ctrader/openapi"
)

func sendReceive[InType proto.Message, OutType proto.Message](
	ctx context.Context,
	conn *connection,
	in InType,
	out OutType,
) error {
	payloadType, err := mapPayload(in)
	if err != nil {
		return fmt.Errorf("cannot map payload: %w", err)
	}

	payloadBase, err := proto.Marshal(in)
	if err != nil {
		return fmt.Errorf("cannot marshal payload: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("cannot create uuid: %w", err)
	}

	msgID := id.String()
	reqType := uint32(payloadType)

	msg := openapi.ProtoMessage{
		ClientMsgId: &msgID,
		PayloadType: &reqType,
		Payload:     payloadBase,
	}

	respChan := make(chan openapi.ProtoMessage, 1)
	conn.pending.Store(msgID, respChan)
	defer conn.pending.Delete(msgID)

	// Send message or abort on context cancel
	select {
	case conn.writeChan <- msg:
	case <-ctx.Done():
		return ctx.Err()
	case <-conn.ctx.Done():
		return conn.ctx.Err()
	}

	// Wait for response or cancel
	select {
	case response, ok := <-respChan:
		if !ok {
			return fmt.Errorf("response channel closed")
		}
		if err := proto.Unmarshal(response.Payload, out); err != nil {
			return fmt.Errorf("cannot decode response payload: %w", err)
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-conn.ctx.Done():
		return conn.ctx.Err()
	}

	return nil
}

func subscribe(
	conn *connection,
	msgType openapi.ProtoOAPayloadType,
	cb func(*openapi.ProtoMessage),
) (func(), error) {

	internalChan := make(chan openapi.ProtoMessage, 256)

	conn.subscribersMu.Lock()
	defer conn.subscribersMu.Unlock()
	conn.subscribers[msgType] = append(conn.subscribers[msgType], internalChan)

	// Decode loop
	ctx, cancel := context.WithCancel(conn.ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-internalChan:
				if !ok {
					return
				}

				cb(&msg)
			}
		}
	}()

	unsub := func() {
		cancel()
		conn.subscribersMu.Lock()
		defer conn.subscribersMu.Unlock()
		channels := conn.subscribers[msgType]
		for i := range channels {
			if channels[i] == internalChan {
				conn.subscribers[msgType] = append(channels[:i], channels[i+1:]...)
				close(internalChan)
				break
			}
		}
	}

	return unsub, nil
}
