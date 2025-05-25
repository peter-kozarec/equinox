package main

import (
	"os"
	"peter-kozarec/equinox/internal/middleware"
	"strconv"
)

var appId = os.Getenv("CtAppId")
var appSecret = os.Getenv("CtAppSecret")
var accountId, _ = strconv.Atoi(os.Getenv("CtAccountId"))
var accessToken = os.Getenv("CtAccessToken")

const (
	RouterEventCapacity = 1000
	MonitorFlags        = middleware.MonitorPositionsOpened | middleware.MonitorPositionsClosed | middleware.MonitorOrders | middleware.MonitorBalance
)
