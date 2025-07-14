package risk

import (
	"context"
	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
	"sync/atomic"
	"testing"
	"time"
)

func createTestInstrument() common.Instrument {
	return common.Instrument{
		Symbol:       "EURUSD",
		Digits:       5,
		PipSize:      fixed.FromFloat64(0.0001),
		ContractSize: fixed.FromInt(100000, 0),
	}
}

func createTestConfiguration() Configuration {
	return Configuration{
		MaxRiskPercentage:            fixed.FromFloat64(2.0),
		MinRiskPercentage:            fixed.FromFloat64(0.5),
		BaseRiskPercentage:           fixed.FromFloat64(1.0),
		MaxOpenRiskPercentage:        fixed.FromFloat64(6.0),
		AtrPeriod:                    14,
		SlAtrMultiplier:              fixed.FromFloat64(2.0),
		BreakEvenMovePercentage:      fixed.FromFloat64(10.0),
		BreakEvenThresholdPercentage: fixed.FromFloat64(50.0),
	}
}

func createTestManager(routerCapacity int) *Manager {
	r := bus.NewRouter(routerCapacity)
	return NewManager(r, createTestInstrument(), createTestConfiguration())
}

func TestNewManager(t *testing.T) {
	r := bus.NewRouter(1000)
	instrument := createTestInstrument()
	cfg := createTestConfiguration()

	m := NewManager(r, instrument, cfg)

	if m.r != r {
		t.Errorf("expected router to be set")
	}
	if m.instrument.Symbol != instrument.Symbol {
		t.Errorf("expected instrument to be set")
	}
	if !m.configuration.MaxRiskPercentage.Eq(cfg.MaxRiskPercentage) {
		t.Errorf("expected configuration to be set")
	}
	if m.atr == nil {
		t.Errorf("expected ATR to be initialized")
	}
}

func TestValidateConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Configuration
		wantErr bool
	}{
		{
			name:    "valid configuration",
			cfg:     createTestConfiguration(),
			wantErr: false,
		},
		{
			name: "invalid max risk percentage - zero",
			cfg: Configuration{
				MaxRiskPercentage:     fixed.Zero,
				MinRiskPercentage:     fixed.FromFloat64(0.5),
				BaseRiskPercentage:    fixed.FromFloat64(1.0),
				MaxOpenRiskPercentage: fixed.FromFloat64(6.0),
				AtrPeriod:             14,
				SlAtrMultiplier:       fixed.FromFloat64(2.0),
			},
			wantErr: true,
		},
		{
			name: "invalid max risk percentage - over 100",
			cfg: Configuration{
				MaxRiskPercentage:     fixed.FromInt(101, 0),
				MinRiskPercentage:     fixed.FromFloat64(0.5),
				BaseRiskPercentage:    fixed.FromFloat64(1.0),
				MaxOpenRiskPercentage: fixed.FromFloat64(6.0),
				AtrPeriod:             14,
				SlAtrMultiplier:       fixed.FromFloat64(2.0),
			},
			wantErr: true,
		},
		{
			name: "invalid min risk percentage - greater than max",
			cfg: Configuration{
				MaxRiskPercentage:     fixed.FromFloat64(2.0),
				MinRiskPercentage:     fixed.FromFloat64(3.0),
				BaseRiskPercentage:    fixed.FromFloat64(1.0),
				MaxOpenRiskPercentage: fixed.FromFloat64(6.0),
				AtrPeriod:             14,
				SlAtrMultiplier:       fixed.FromFloat64(2.0),
			},
			wantErr: true,
		},
		{
			name: "invalid base risk percentage - less than min",
			cfg: Configuration{
				MaxRiskPercentage:     fixed.FromFloat64(2.0),
				MinRiskPercentage:     fixed.FromFloat64(0.5),
				BaseRiskPercentage:    fixed.FromFloat64(0.3),
				MaxOpenRiskPercentage: fixed.FromFloat64(6.0),
				AtrPeriod:             14,
				SlAtrMultiplier:       fixed.FromFloat64(2.0),
			},
			wantErr: true,
		},
		{
			name: "invalid ATR period",
			cfg: Configuration{
				MaxRiskPercentage:     fixed.FromFloat64(2.0),
				MinRiskPercentage:     fixed.FromFloat64(0.5),
				BaseRiskPercentage:    fixed.FromFloat64(1.0),
				MaxOpenRiskPercentage: fixed.FromFloat64(6.0),
				AtrPeriod:             0,
				SlAtrMultiplier:       fixed.FromFloat64(2.0),
			},
			wantErr: true,
		},
		{
			name: "invalid SL ATR multiplier",
			cfg: Configuration{
				MaxRiskPercentage:     fixed.FromFloat64(2.0),
				MinRiskPercentage:     fixed.FromFloat64(0.5),
				BaseRiskPercentage:    fixed.FromFloat64(1.0),
				MaxOpenRiskPercentage: fixed.FromFloat64(6.0),
				AtrPeriod:             14,
				SlAtrMultiplier:       fixed.Zero,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfiguration(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfiguration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRiskManager_CalculatePositionSize(t *testing.T) {
	m := createTestManager(1000)
	m.currentEquity = common.Equity{
		Value:     fixed.FromInt(10000, 0),
		TimeStamp: time.Now(),
	}

	tests := []struct {
		name           string
		entry          fixed.Point
		sl             fixed.Point
		riskPercentage fixed.Point
		want           fixed.Point
	}{
		{
			name:           "long position 1% risk",
			entry:          fixed.FromFloat64(1.2000),
			sl:             fixed.FromFloat64(1.1950),
			riskPercentage: fixed.FromFloat64(1.0),
			want:           fixed.FromFloat64(0.20),
		},
		{
			name:           "short position 1% risk",
			entry:          fixed.FromFloat64(1.2000),
			sl:             fixed.FromFloat64(1.2050),
			riskPercentage: fixed.FromFloat64(1.0),
			want:           fixed.FromFloat64(0.20),
		},
		{
			name:           "long position 2% risk",
			entry:          fixed.FromFloat64(1.2000),
			sl:             fixed.FromFloat64(1.1950),
			riskPercentage: fixed.FromFloat64(2.0),
			want:           fixed.FromFloat64(0.40),
		},
		{
			name:           "smaller stop loss",
			entry:          fixed.FromFloat64(1.2000),
			sl:             fixed.FromFloat64(1.1980),
			riskPercentage: fixed.FromFloat64(1.0),
			want:           fixed.FromFloat64(0.50),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.calculatePositionSize(tt.entry, tt.sl, tt.riskPercentage)
			if !got.Eq(tt.want) {
				t.Errorf("calculatePositionSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRiskManager_CalculateRiskPercentage(t *testing.T) {
	m := createTestManager(1000)
	m.currentEquity = common.Equity{
		Value:     fixed.FromInt(10000, 0),
		TimeStamp: time.Now(),
	}

	tests := []struct {
		name  string
		entry fixed.Point
		sl    fixed.Point
		size  fixed.Point
		want  fixed.Point
	}{
		{
			name:  "long position standard risk",
			entry: fixed.FromFloat64(1.2000),
			sl:    fixed.FromFloat64(1.1950),
			size:  fixed.FromFloat64(0.20),
			want:  fixed.FromFloat64(1.0),
		},
		{
			name:  "short position standard risk",
			entry: fixed.FromFloat64(1.2000),
			sl:    fixed.FromFloat64(1.2050),
			size:  fixed.FromFloat64(0.20),
			want:  fixed.FromFloat64(1.0),
		},
		{
			name:  "double position size",
			entry: fixed.FromFloat64(1.2000),
			sl:    fixed.FromFloat64(1.1950),
			size:  fixed.FromFloat64(0.40),
			want:  fixed.FromFloat64(2.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.calculateRiskPercentage(tt.entry, tt.sl, tt.size)
			if !got.Eq(tt.want) {
				t.Errorf("calculateRiskPercentage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRiskManager_CalculateStopLoss(t *testing.T) {
	m := createTestManager(1000)

	tests := []struct {
		name   string
		entry  fixed.Point
		target fixed.Point
		atr    fixed.Point
		want   fixed.Point
	}{
		{
			name:   "buy signal",
			entry:  fixed.FromFloat64(1.2000),
			target: fixed.FromFloat64(1.2100),
			atr:    fixed.FromFloat64(0.0020),
			want:   fixed.FromFloat64(1.1960),
		},
		{
			name:   "sell signal",
			entry:  fixed.FromFloat64(1.2000),
			target: fixed.FromFloat64(1.1900),
			atr:    fixed.FromFloat64(0.0020),
			want:   fixed.FromFloat64(1.2040),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.calculateStopLoss(tt.entry, tt.target, tt.atr)
			if !got.Eq(tt.want) {
				t.Errorf("calculateStopLoss() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRiskManager_CalculateCurrentOpenRisk(t *testing.T) {
	m := createTestManager(1000)
	m.currentEquity = common.Equity{
		Value:     fixed.FromInt(10000, 0),
		TimeStamp: time.Now(),
	}

	m.openPositions = []common.Position{
		{
			TraceID:   utility.CreateTraceID(),
			OpenPrice: fixed.FromFloat64(1.2000),
			StopLoss:  fixed.FromFloat64(1.1950),
			Size:      fixed.FromFloat64(0.20),
		},
		{
			TraceID:   utility.CreateTraceID(),
			OpenPrice: fixed.FromFloat64(1.3000),
			StopLoss:  fixed.FromFloat64(1.3050),
			Size:      fixed.FromFloat64(0.20),
		},
	}

	got := m.calculateCurrentOpenRisk()
	want := fixed.FromFloat64(2.0)

	if !got.Eq(want) {
		t.Errorf("calculateCurrentOpenRisk() = %v, want %v", got, want)
	}
}

func TestRiskManager_IsTimeToTrade(t *testing.T) {
	m := createTestManager(1000)

	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{
			name: "monday before 10:00",
			time: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "monday at 10:00",
			time: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "tuesday at 12:00",
			time: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "friday after 16:00",
			time: time.Date(2024, 1, 5, 17, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "saturday",
			time: time.Date(2024, 1, 6, 12, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "sunday",
			time: time.Date(2024, 1, 7, 12, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "before 8:00",
			time: time.Date(2024, 1, 2, 7, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "after 18:00",
			time: time.Date(2024, 1, 2, 19, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.serverTime = tt.time
			if got := m.isTimeToTrade(); got != tt.want {
				t.Errorf("isTimeToTrade() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRiskManager_CreateMarketOrder(t *testing.T) {
	m := createTestManager(1000)

	tests := []struct {
		name    string
		entry   fixed.Point
		tp      fixed.Point
		sl      fixed.Point
		size    fixed.Point
		wantErr bool
	}{
		{
			name:    "valid buy order",
			entry:   fixed.FromFloat64(1.2000),
			tp:      fixed.FromFloat64(1.2100),
			sl:      fixed.FromFloat64(1.1950),
			size:    fixed.FromFloat64(0.20),
			wantErr: false,
		},
		{
			name:    "valid sell order",
			entry:   fixed.FromFloat64(1.2000),
			tp:      fixed.FromFloat64(1.1900),
			sl:      fixed.FromFloat64(1.2050),
			size:    fixed.FromFloat64(0.20),
			wantErr: false,
		},
		{
			name:    "invalid buy order - tp less than entry",
			entry:   fixed.FromFloat64(1.2000),
			tp:      fixed.FromFloat64(1.1900),
			sl:      fixed.FromFloat64(1.1950),
			size:    fixed.FromFloat64(0.20),
			wantErr: true,
		},
		{
			name:    "invalid sell order - tp greater than entry",
			entry:   fixed.FromFloat64(1.2000),
			tp:      fixed.FromFloat64(1.2100),
			sl:      fixed.FromFloat64(1.2050),
			size:    fixed.FromFloat64(0.20),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.createMarketOrder(tt.entry, tt.tp, tt.sl, tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("createMarketOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Command != common.OrderCommandPositionOpen {
					t.Errorf("expected OrderCommandPositionOpen")
				}
				if got.Type != common.OrderTypeMarket {
					t.Errorf("expected OrderTypeMarket")
				}
			}
		})
	}
}

func TestRiskManager_OnTick(t *testing.T) {
	m := createTestManager(1000)

	tick := common.Tick{
		Symbol:    "EURUSD",
		TimeStamp: time.Now(),
		Bid:       fixed.FromFloat64(1.2000),
		Ask:       fixed.FromFloat64(1.2001),
	}

	m.OnTick(context.Background(), tick)

	if m.serverTime != tick.TimeStamp {
		t.Errorf("expected server time to be updated")
	}
	if !m.lastTick.Bid.Eq(tick.Bid) {
		t.Errorf("expected last tick to be updated")
	}
}

func TestRiskManager_OnBalance(t *testing.T) {
	m := createTestManager(1000)

	balance1 := common.Balance{
		Value:     fixed.FromInt(10000, 0),
		TimeStamp: time.Now(),
	}
	m.OnBalance(context.Background(), balance1)

	if !m.currentBalance.Value.Eq(balance1.Value) {
		t.Errorf("expected current balance to be updated")
	}
	if !m.maxBalance.Value.Eq(balance1.Value) {
		t.Errorf("expected max balance to be set")
	}

	balance2 := common.Balance{
		Value:     fixed.FromInt(9000, 0),
		TimeStamp: time.Now(),
	}
	m.OnBalance(context.Background(), balance2)

	if !m.currentBalance.Value.Eq(balance2.Value) {
		t.Errorf("expected current balance to be updated")
	}
	if !m.maxBalance.Value.Eq(balance1.Value) {
		t.Errorf("expected max balance to remain unchanged")
	}

	balance3 := common.Balance{
		Value:     fixed.FromInt(11000, 0),
		TimeStamp: time.Now(),
	}
	m.OnBalance(context.Background(), balance3)

	if !m.maxBalance.Value.Eq(balance3.Value) {
		t.Errorf("expected max balance to be updated")
	}
}

func TestRiskManager_OnEquity(t *testing.T) {
	m := createTestManager(1000)

	equity1 := common.Equity{
		Value:     fixed.FromInt(10000, 0),
		TimeStamp: time.Now(),
	}
	m.OnEquity(context.Background(), equity1)

	if !m.currentEquity.Value.Eq(equity1.Value) {
		t.Errorf("expected current equity to be updated")
	}
	if !m.maxEquity.Value.Eq(equity1.Value) {
		t.Errorf("expected max equity to be set")
	}

	equity2 := common.Equity{
		Value:     fixed.FromInt(9000, 0),
		TimeStamp: time.Now(),
	}
	m.OnEquity(context.Background(), equity2)

	if !m.currentEquity.Value.Eq(equity2.Value) {
		t.Errorf("expected current equity to be updated")
	}
	if !m.maxEquity.Value.Eq(equity1.Value) {
		t.Errorf("expected max equity to remain unchanged")
	}
}

func TestRiskManager_OnPositionOpened(t *testing.T) {
	m := createTestManager(1000)

	position := common.Position{
		TraceID:   utility.CreateTraceID(),
		Id:        123,
		OpenPrice: fixed.FromFloat64(1.2000),
		StopLoss:  fixed.FromFloat64(1.1950),
		Size:      fixed.FromFloat64(0.20),
	}

	m.OnPositionOpened(context.Background(), position)

	if len(m.openPositions) != 1 {
		t.Errorf("expected 1 open position, got %d", len(m.openPositions))
	}
	if m.openPositions[0].TraceID != position.TraceID {
		t.Errorf("expected position to be added")
	}
}

func TestRiskManager_OnPositionClosed(t *testing.T) {
	m := createTestManager(1000)

	position := common.Position{
		TraceID:   utility.CreateTraceID(),
		Id:        123,
		OpenPrice: fixed.FromFloat64(1.2000),
		StopLoss:  fixed.FromFloat64(1.1950),
		Size:      fixed.FromFloat64(0.20),
	}

	m.openPositions = append(m.openPositions, position)
	m.OnPositionClosed(context.Background(), position)

	if len(m.openPositions) != 0 {
		t.Errorf("expected 0 open positions, got %d", len(m.openPositions))
	}
	if len(m.closedPositions) != 1 {
		t.Errorf("expected 1 closed position, got %d", len(m.closedPositions))
	}
}

func TestRiskManager_OnPositionUpdated(t *testing.T) {
	m := createTestManager(1000)

	position := common.Position{
		TraceID:   utility.CreateTraceID(),
		Id:        123,
		OpenPrice: fixed.FromFloat64(1.2000),
		StopLoss:  fixed.FromFloat64(1.1950),
		Size:      fixed.FromFloat64(0.20),
	}

	m.openPositions = append(m.openPositions, position)

	updatedPosition := position
	updatedPosition.StopLoss = fixed.FromFloat64(1.1980)

	m.OnPositionUpdated(context.Background(), updatedPosition)

	if !m.openPositions[0].StopLoss.Eq(updatedPosition.StopLoss) {
		t.Errorf("expected position to be updated")
	}
}

func TestRiskManager_CheckForBreakEven(t *testing.T) {
	tests := []struct {
		name        string
		position    common.Position
		expectOrder bool
	}{
		{
			name: "long position reached threshold",
			position: common.Position{
				TraceID:    utility.CreateTraceID(),
				Id:         123,
				Side:       common.PositionSideLong,
				OpenPrice:  fixed.FromFloat64(1.2000),
				StopLoss:   fixed.FromFloat64(1.1950),
				TakeProfit: fixed.FromFloat64(1.2100),
			},
			expectOrder: true,
		},
		{
			name: "long position not reached threshold",
			position: common.Position{
				TraceID:    utility.CreateTraceID(),
				Id:         124,
				Side:       common.PositionSideLong,
				OpenPrice:  fixed.FromFloat64(1.2000),
				StopLoss:   fixed.FromFloat64(1.1950),
				TakeProfit: fixed.FromFloat64(1.2200),
			},
			expectOrder: false,
		},
		{
			name: "long position stop loss already at break even",
			position: common.Position{
				TraceID:    utility.CreateTraceID(),
				Id:         125,
				Side:       common.PositionSideLong,
				OpenPrice:  fixed.FromFloat64(1.2000),
				StopLoss:   fixed.FromFloat64(1.2000),
				TakeProfit: fixed.FromFloat64(1.2100),
			},
			expectOrder: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bus.NewRouter(1000)
			m := NewManager(r, createTestInstrument(), createTestConfiguration())

			m.serverTime = time.Now()
			m.lastTick = common.Tick{
				Bid: fixed.FromFloat64(1.2050),
				Ask: fixed.FromFloat64(1.2051),
			}

			var orderPosted atomic.Bool
			r.OrderHandler = func(ctx context.Context, order common.Order) {
				orderPosted.Store(true)
			}

			err := m.checkForBreakEven(tt.position)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			r.Exec(ctx)

			time.Sleep(100 * time.Millisecond) // Should be enough for the event to be dispatched
			cancel()

			if orderPosted.Load() != tt.expectOrder {
				t.Errorf("expected order posted = %v, got %v", tt.expectOrder, orderPosted.Load())
			}
		})
	}
}

func BenchmarkRiskManager_CalculatePositionSize(b *testing.B) {
	m := createTestManager(b.N)
	m.currentEquity = common.Equity{
		Value:     fixed.FromInt(10000, 0),
		TimeStamp: time.Now(),
	}

	entry := fixed.FromFloat64(1.2000)
	sl := fixed.FromFloat64(1.1950)
	risk := fixed.FromFloat64(1.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.calculatePositionSize(entry, sl, risk)
	}
}

func BenchmarkRiskManager_OnSignal(b *testing.B) {
	m := createTestManager(b.N)
	m.currentEquity = common.Equity{
		Value:     fixed.FromInt(10000, 0),
		TimeStamp: time.Now(),
	}
	m.maxEquity = m.currentEquity
	m.currentBalance = common.Balance{
		Value:     fixed.FromInt(10000, 0),
		TimeStamp: time.Now(),
	}
	m.maxBalance = m.currentBalance
	m.serverTime = time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	for i := 0; i < 14; i++ {
		m.OnBar(context.Background(), common.Bar{
			Open:  fixed.FromFloat64(1.2000),
			High:  fixed.FromFloat64(1.2050),
			Low:   fixed.FromFloat64(1.1950),
			Close: fixed.FromFloat64(1.2020),
		})
	}

	signal := common.Signal{
		Entry:    fixed.FromFloat64(1.2000),
		Target:   fixed.FromFloat64(1.2100),
		Strength: 80,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.OnSignal(context.Background(), signal)
		m.pendingOrders = nil
	}
}

func BenchmarkRiskManager_CalculateCurrentOpenRisk(b *testing.B) {
	m := createTestManager(b.N)
	m.currentEquity = common.Equity{
		Value:     fixed.FromInt(10000, 0),
		TimeStamp: time.Now(),
	}

	for i := 0; i < 5; i++ {
		m.openPositions = append(m.openPositions, common.Position{
			TraceID:   utility.CreateTraceID(),
			OpenPrice: fixed.FromFloat64(1.2000),
			StopLoss:  fixed.FromFloat64(1.1950),
			Size:      fixed.FromFloat64(0.10),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.calculateCurrentOpenRisk()
	}
}
