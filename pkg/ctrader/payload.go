package ctrader

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"peter-kozarec/equinox/pkg/ctrader/openapi"
)

func mapPayload(message proto.Message) (openapi.ProtoOAPayloadType, error) {
	switch message.(type) {
	case *openapi.ProtoOAApplicationAuthReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_APPLICATION_AUTH_REQ, nil
	case *openapi.ProtoOAAccountAuthReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_ACCOUNT_AUTH_REQ, nil
	case *openapi.ProtoOAVersionReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_VERSION_REQ, nil
	case *openapi.ProtoOANewOrderReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_NEW_ORDER_REQ, nil
	case *openapi.ProtoOACancelOrderReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_CANCEL_ORDER_REQ, nil
	case *openapi.ProtoOAAmendOrderReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_AMEND_ORDER_REQ, nil
	case *openapi.ProtoOAAmendPositionSLTPReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_AMEND_POSITION_SLTP_REQ, nil
	case *openapi.ProtoOAClosePositionReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_CLOSE_POSITION_REQ, nil
	case *openapi.ProtoOAAssetListReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_ASSET_LIST_REQ, nil
	case *openapi.ProtoOASymbolsListReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_SYMBOLS_LIST_REQ, nil
	case *openapi.ProtoOASymbolByIdReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_SYMBOL_BY_ID_REQ, nil
	case *openapi.ProtoOASymbolsForConversionReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_SYMBOLS_FOR_CONVERSION_REQ, nil
	case *openapi.ProtoOATraderReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_TRADER_REQ, nil
	case *openapi.ProtoOAReconcileReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_RECONCILE_REQ, nil
	case *openapi.ProtoOASubscribeSpotsReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_SUBSCRIBE_SPOTS_REQ, nil
	case *openapi.ProtoOAUnsubscribeSpotsReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_UNSUBSCRIBE_SPOTS_REQ, nil
	case *openapi.ProtoOADealListReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_DEAL_LIST_REQ, nil
	case *openapi.ProtoOASubscribeLiveTrendbarReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_SUBSCRIBE_LIVE_TRENDBAR_REQ, nil
	case *openapi.ProtoOAUnsubscribeLiveTrendbarReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_UNSUBSCRIBE_LIVE_TRENDBAR_REQ, nil
	case *openapi.ProtoOAGetTrendbarsReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_GET_TRENDBARS_REQ, nil
	case *openapi.ProtoOAExpectedMarginReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_EXPECTED_MARGIN_REQ, nil
	case *openapi.ProtoOACashFlowHistoryListReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_CASH_FLOW_HISTORY_LIST_REQ, nil
	case *openapi.ProtoOAGetTickDataReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_GET_TICKDATA_REQ, nil
	case *openapi.ProtoOAGetAccountListByAccessTokenReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_GET_ACCOUNTS_BY_ACCESS_TOKEN_REQ, nil
	case *openapi.ProtoOAGetCtidProfileByTokenReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_GET_CTID_PROFILE_BY_TOKEN_REQ, nil
	case *openapi.ProtoOAAssetClassListReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_ASSET_CLASS_LIST_REQ, nil
	case *openapi.ProtoOASubscribeDepthQuotesReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_SUBSCRIBE_DEPTH_QUOTES_REQ, nil
	case *openapi.ProtoOAUnsubscribeDepthQuotesReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_UNSUBSCRIBE_DEPTH_QUOTES_REQ, nil
	case *openapi.ProtoOASymbolCategoryListReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_SYMBOL_CATEGORY_REQ, nil
	case *openapi.ProtoOAAccountLogoutReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_ACCOUNT_LOGOUT_REQ, nil
	case *openapi.ProtoOAMarginCallListReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_MARGIN_CALL_LIST_REQ, nil
	case *openapi.ProtoOAMarginCallUpdateReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_MARGIN_CALL_UPDATE_REQ, nil
	case *openapi.ProtoOARefreshTokenReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_REFRESH_TOKEN_REQ, nil
	case *openapi.ProtoOAOrderListReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_ORDER_LIST_REQ, nil
	case *openapi.ProtoOAGetDynamicLeverageByIDReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_GET_DYNAMIC_LEVERAGE_REQ, nil
	case *openapi.ProtoOADealListByPositionIdReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_DEAL_LIST_BY_POSITION_ID_REQ, nil
	case *openapi.ProtoOAOrderDetailsReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_ORDER_DETAILS_REQ, nil
	case *openapi.ProtoOAOrderListByPositionIdReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_ORDER_LIST_BY_POSITION_ID_REQ, nil
	case *openapi.ProtoOADealOffsetListReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_DEAL_OFFSET_LIST_REQ, nil
	case *openapi.ProtoOAGetPositionUnrealizedPnLReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_GET_POSITION_UNREALIZED_PNL_REQ, nil
	case *openapi.ProtoOAv1PnLChangeSubscribeReq:
		return openapi.ProtoOAPayloadType_PROTO_OA_V1_PNL_CHANGE_SUBSCRIBE_REQ, nil
	default:
		return 0, errors.New("unknown proto type")
	}
}
