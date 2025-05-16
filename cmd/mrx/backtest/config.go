package main

import (
	"peter-kozarec/equinox/internal/middleware"
	"time"
)

var SimulationStart = time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
var SimulationEnd = time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)

const (
	RouterEventCapacity = 100
	TickDataSource      = "data/eurusd_ticks_2018-2025_v2.bin"
	MonitorFlags        = middleware.MonitorPositionsOpened | middleware.MonitorPositionsClosed | middleware.MonitorOrders
)
