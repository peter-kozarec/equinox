package ctrader

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/ctrader/openapi"
	"time"
)

func createSpotsCallback(router *bus.Router, logger *zap.Logger, period time.Duration) func(*openapi.ProtoMessage) {

	return func(msg *openapi.ProtoMessage) {

		var v openapi.ProtoOASpotEvent

		err := proto.Unmarshal(msg.GetPayload(), &v)
		if err != nil {
			logger.Warn("unable to unmarshal spots event", zap.Error(err))
		}

	}
}

func createErrorResponseCallback(logger *zap.Logger) func(*openapi.ProtoMessage) {

	return func(msg *openapi.ProtoMessage) {

		var v openapi.ProtoOAErrorRes

		err := proto.Unmarshal(msg.GetPayload(), &v)
		if err != nil {
			logger.Warn("unable to unmarshal error response", zap.Error(err))
		}

		logger.Warn("error", zap.Any("code", v.GetErrorCode()), zap.Any("description", v.GetDescription()))
	}
}
