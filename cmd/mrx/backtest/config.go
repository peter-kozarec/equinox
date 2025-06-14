package main

import (
	"peter-kozarec/equinox/internal/middleware"
	"time"
)

var SimulationStart = time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
var SimulationEnd = time.Date(2018, 12, 31, 0, 0, 0, 0, time.UTC)

const (
	RouterEventCapacity = 1000
	TickDataSource      = "data/audusd.bin"
	MonitorFlags        = middleware.MonitorPositionsClosed | middleware.MonitorPositionsOpened
)
