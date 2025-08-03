package sandbox

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/exchange"
	"github.com/peter-kozarec/equinox/pkg/utility"
	"github.com/peter-kozarec/equinox/pkg/utility/fixed"
)

type mockRateProvider struct {
	rate          fixed.Point
	conversionFee fixed.Point
}

func (m *mockRateProvider) ExchangeRate(_, _ string, _ time.Time) (fixed.Point, fixed.Point, error) {
	return m.rate, m.conversionFee, nil
}

func createTestSimulator(t *testing.T) (*Simulator, *bus.Router) {
	router := bus.NewRouter(1000)

	eurUsd := exchange.SymbolInfo{
		SymbolName:    "EURUSD",
		QuoteCurrency: "USD",
		ContractSize:  fixed.FromInt(100000, 0),
		Leverage:      fixed.FromInt(100, 0),
	}
	gbpUsd := exchange.SymbolInfo{
		SymbolName:    "GBPUSD",
		QuoteCurrency: "USD",
		ContractSize:  fixed.FromInt(100000, 0),
		Leverage:      fixed.FromInt(100, 0),
	}

	sim, err := NewSimulator(router, "USD", fixed.FromInt(10000, 0), WithSymbols(eurUsd, gbpUsd))
	if err != nil {
		t.Fatal(err)
	}

	sim.simulationTime = time.Now()
	return sim, router
}

func TestSandboxSimulator_executeOpenOrder(t *testing.T) {
	tests := []struct {
		name          string
		order         common.Order
		tick          common.Tick
		setup         func(*Simulator)
		expectedError string
		validate      func(*testing.T, *Simulator, *common.Position)
	}{
		{
			name: "successful buy market order",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
				TraceID:     1,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			validate: func(t *testing.T, sim *Simulator, pos *common.Position) {
				assert.Equal(t, common.PositionSideLong, pos.Side)
				assert.Equal(t, fixed.FromFloat64(0.1).String(), pos.Size.String())
				assert.Equal(t, positionStatusPendingOpen, pos.Status)
				assert.Equal(t, common.PositionId(1), pos.Id)
			},
		},
		{
			name: "successful sell market order",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.2),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
				TraceID:     2,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			validate: func(t *testing.T, sim *Simulator, pos *common.Position) {
				assert.Equal(t, common.PositionSideShort, pos.Side)
				assert.Equal(t, fixed.FromFloat64(0.2).String(), pos.Size.String())
			},
		},
		{
			name: "buy order with stop loss and take profit",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				StopLoss:    fixed.FromFloat64(1.0950),
				TakeProfit:  fixed.FromFloat64(1.1050),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			validate: func(t *testing.T, sim *Simulator, pos *common.Position) {
				assert.Equal(t, fixed.FromFloat64(1.0950).String(), pos.StopLoss.String())
				assert.Equal(t, fixed.FromFloat64(1.1050).String(), pos.TakeProfit.String())
			},
		},
		{
			name: "insufficient liquidity partial fill",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(5.0),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromFloat64(2.5),
			},
			validate: func(t *testing.T, sim *Simulator, pos *common.Position) {
				f1, _ := fixed.FromFloat64(2.5).Float64()
				f2, _ := pos.Size.Float64()
				assert.Equal(t, f1, f2)
			},
		},
		{
			name: "zero liquidity error",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.Zero,
			},
			expectedError: "available liquidity is zero",
		},
		{
			name: "invalid buy stop loss above bid",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				StopLoss:    fixed.FromFloat64(1.1005),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			expectedError: "stop loss must be less than bid",
		},
		{
			name: "invalid buy take profit below bid",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				TakeProfit:  fixed.FromFloat64(1.0995),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			expectedError: "take profit must be greater than bid",
		},
		{
			name: "invalid sell stop loss below ask",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				StopLoss:    fixed.FromFloat64(1.0995),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			expectedError: "stop loss must be greater than ask",
		},
		{
			name: "partially filled order",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(1.0),
				FilledSize:  fixed.FromFloat64(0.3),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			validate: func(t *testing.T, sim *Simulator, pos *common.Position) {
				assert.Equal(t, fixed.FromFloat64(0.7).String(), pos.Size.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			pos, err := sim.executeOpenOrder(tt.order, tt.tick)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pos)
				if tt.validate != nil {
					tt.validate(t, sim, pos)
				}
			}
		})
	}
}

func TestSandboxSimulator_executeCloseOrder(t *testing.T) {
	tests := []struct {
		name          string
		order         common.Order
		tick          common.Tick
		setup         func(*Simulator)
		expectedError string
		validate      func(*testing.T, *Simulator, *common.Position, fixed.Point)
	}{
		{
			name: "successful close full position",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionClose,
				PositionId:  1,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			setup: func(sim *Simulator) {
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:     1,
					Symbol: "EURUSD",
					Side:   common.PositionSideLong,
					Size:   fixed.FromFloat64(0.1),
					Status: common.PositionStatusOpen,
				})
			},
			validate: func(t *testing.T, sim *Simulator, pos *common.Position, size fixed.Point) {
				assert.Equal(t, positionStatusPendingClose, pos.Status)
				assert.Equal(t, fixed.FromFloat64(0.1).String(), size.String())
			},
		},
		{
			name: "partial close with insufficient liquidity",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(5.0),
				Command:     common.OrderCommandPositionClose,
				PositionId:  1,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromFloat64(2.0),
				AskVolume: fixed.FromInt(10, 0),
			},
			setup: func(sim *Simulator) {
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:     1,
					Symbol: "EURUSD",
					Side:   common.PositionSideLong,
					Size:   fixed.FromFloat64(5.0),
					Status: common.PositionStatusOpen,
				})
			},
			validate: func(t *testing.T, sim *Simulator, pos *common.Position, size fixed.Point) {
				f1, _ := fixed.Two.Float64()
				f2, _ := size.Float64()
				assert.Equal(t, f1, f2)
			},
		},
		{
			name: "position not found",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionClose,
				PositionId:  999,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			expectedError: "position with id 999 not found",
		},
		{
			name: "zero liquidity error",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionClose,
				PositionId:  1,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.Zero,
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.Zero,
				AskVolume: fixed.FromInt(10, 0),
			},
			setup: func(sim *Simulator) {
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:     1,
					Symbol: "EURUSD",
					Side:   common.PositionSideLong,
					Size:   fixed.FromFloat64(0.1),
					Status: common.PositionStatusOpen,
				})
			},
			expectedError: "available liquidity is zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			pos, size, err := sim.executeCloseOrder(tt.order, tt.tick)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pos)
				if tt.validate != nil {
					tt.validate(t, sim, pos, size)
				}
			}
		})
	}
}

func TestSandboxSimulator_modifyPosition(t *testing.T) {
	tests := []struct {
		name          string
		order         common.Order
		tick          common.Tick
		setup         func(*Simulator)
		expectedError string
		validate      func(*testing.T, *Simulator)
	}{
		{
			name: "successful modify both SL and TP",
			order: common.Order{
				Symbol:     "EURUSD",
				Command:    common.OrderCommandPositionModify,
				PositionId: 1,
				StopLoss:   fixed.FromFloat64(1.0950),
				TakeProfit: fixed.FromFloat64(1.1050),
				TraceID:    100,
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1000),
				Ask:    fixed.FromFloat64(1.1002),
			},
			setup: func(sim *Simulator) {
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:            1,
					Symbol:        "EURUSD",
					Side:          common.PositionSideLong,
					OrderTraceIDs: []utility.TraceID{1},
				})
			},
			validate: func(t *testing.T, sim *Simulator) {
				pos := sim.openPositions[0]
				assert.Equal(t, fixed.FromFloat64(1.0950), pos.StopLoss)
				assert.Equal(t, fixed.FromFloat64(1.1050), pos.TakeProfit)
				assert.Contains(t, pos.OrderTraceIDs, utility.TraceID(100))
			},
		},
		{
			name: "modify only stop loss",
			order: common.Order{
				Symbol:     "EURUSD",
				Command:    common.OrderCommandPositionModify,
				PositionId: 1,
				StopLoss:   fixed.FromFloat64(1.0980),
				TakeProfit: fixed.Zero,
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1000),
				Ask:    fixed.FromFloat64(1.1002),
			},
			setup: func(sim *Simulator) {
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:         1,
					Symbol:     "EURUSD",
					Side:       common.PositionSideLong,
					TakeProfit: fixed.FromFloat64(1.1050),
				})
			},
			validate: func(t *testing.T, sim *Simulator) {
				pos := sim.openPositions[0]
				assert.Equal(t, fixed.FromFloat64(1.0980), pos.StopLoss)
				assert.Equal(t, fixed.FromFloat64(1.1050), pos.TakeProfit)
			},
		},
		{
			name: "position not found",
			order: common.Order{
				Symbol:     "EURUSD",
				Command:    common.OrderCommandPositionModify,
				PositionId: 999,
				StopLoss:   fixed.FromFloat64(1.0950),
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1000),
				Ask:    fixed.FromFloat64(1.1002),
			},
			expectedError: "position with id 999 not found",
		},
		{
			name: "invalid long position SL above bid",
			order: common.Order{
				Symbol:     "EURUSD",
				Command:    common.OrderCommandPositionModify,
				PositionId: 1,
				StopLoss:   fixed.FromFloat64(1.1005),
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1000),
				Ask:    fixed.FromFloat64(1.1002),
			},
			setup: func(sim *Simulator) {
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:     1,
					Symbol: "EURUSD",
					Side:   common.PositionSideLong,
				})
			},
			expectedError: "stop loss must be less than bid",
		},
		{
			name: "invalid short position SL below ask",
			order: common.Order{
				Symbol:     "EURUSD",
				Command:    common.OrderCommandPositionModify,
				PositionId: 1,
				StopLoss:   fixed.FromFloat64(1.0995),
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1000),
				Ask:    fixed.FromFloat64(1.1002),
			},
			setup: func(sim *Simulator) {
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:     1,
					Symbol: "EURUSD",
					Side:   common.PositionSideShort,
				})
			},
			expectedError: "stop loss must be greater than ask",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			err := sim.modifyPosition(tt.order, tt.tick)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, sim)
				}
			}
		})
	}
}

func TestSandboxSimulator_shouldExecuteLimitOrder(t *testing.T) {
	tests := []struct {
		name     string
		order    common.Order
		tick     common.Tick
		expected bool
	}{
		{
			name: "buy limit order executable",
			order: common.Order{
				Command: common.OrderCommandPositionOpen,
				Side:    common.OrderSideBuy,
				Price:   fixed.FromFloat64(1.1005),
			},
			tick: common.Tick{
				Ask: fixed.FromFloat64(1.1002),
			},
			expected: true,
		},
		{
			name: "buy limit order not executable",
			order: common.Order{
				Command: common.OrderCommandPositionOpen,
				Side:    common.OrderSideBuy,
				Price:   fixed.FromFloat64(1.0999),
			},
			tick: common.Tick{
				Ask: fixed.FromFloat64(1.1002),
			},
			expected: false,
		},
		{
			name: "sell limit order executable",
			order: common.Order{
				Command: common.OrderCommandPositionOpen,
				Side:    common.OrderSideSell,
				Price:   fixed.FromFloat64(1.0995),
			},
			tick: common.Tick{
				Bid: fixed.FromFloat64(1.1000),
			},
			expected: true,
		},
		{
			name: "close buy order executable",
			order: common.Order{
				Command: common.OrderCommandPositionClose,
				Side:    common.OrderSideBuy,
				Price:   fixed.FromFloat64(1.0995),
			},
			tick: common.Tick{
				Bid: fixed.FromFloat64(1.1000),
			},
			expected: true,
		},
		{
			name: "close sell order executable",
			order: common.Order{
				Command: common.OrderCommandPositionClose,
				Side:    common.OrderSideSell,
				Price:   fixed.FromFloat64(1.1005),
			},
			tick: common.Tick{
				Ask: fixed.FromFloat64(1.1002),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			result := sim.shouldExecuteLimitOrder(tt.order, tt.tick)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSandboxSimulator_shouldClosePosition(t *testing.T) {
	tests := []struct {
		name     string
		position common.Position
		tick     common.Tick
		expected bool
	}{
		{
			name: "long position hit take profit",
			position: common.Position{
				Symbol:     "EURUSD",
				Side:       common.PositionSideLong,
				TakeProfit: fixed.FromFloat64(1.1050),
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1051),
			},
			expected: true,
		},
		{
			name: "long position hit stop loss",
			position: common.Position{
				Symbol:   "EURUSD",
				Side:     common.PositionSideLong,
				StopLoss: fixed.FromFloat64(1.0950),
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.0949),
			},
			expected: true,
		},
		{
			name: "short position hit take profit",
			position: common.Position{
				Symbol:     "EURUSD",
				Side:       common.PositionSideShort,
				TakeProfit: fixed.FromFloat64(1.0950),
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Ask:    fixed.FromFloat64(1.0949),
			},
			expected: true,
		},
		{
			name: "short position hit stop loss",
			position: common.Position{
				Symbol:   "EURUSD",
				Side:     common.PositionSideShort,
				StopLoss: fixed.FromFloat64(1.1050),
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Ask:    fixed.FromFloat64(1.1051),
			},
			expected: true,
		},
		{
			name: "position not ready to close",
			position: common.Position{
				Symbol:     "EURUSD",
				Side:       common.PositionSideLong,
				StopLoss:   fixed.FromFloat64(1.0950),
				TakeProfit: fixed.FromFloat64(1.1050),
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1000),
			},
			expected: false,
		},
		{
			name: "different symbol",
			position: common.Position{
				Symbol:     "GBPUSD",
				Side:       common.PositionSideLong,
				TakeProfit: fixed.FromFloat64(1.1050),
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1051),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			result := sim.shouldClosePosition(tt.position, tt.tick)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSandboxSimulator_calcPositionProfits(t *testing.T) {
	tests := []struct {
		name     string
		position *common.Position
		setup    func(*Simulator)
		validate func(*testing.T, *common.Position)
	}{
		{
			name: "long position profit calculation",
			position: &common.Position{
				Symbol:    "EURUSD",
				Side:      common.PositionSideLong,
				Size:      fixed.FromFloat64(0.1),
				OpenPrice: fixed.FromFloat64(1.1000),
				Status:    common.PositionStatusOpen,
				TimeStamp: time.Now(),
			},
			validate: func(t *testing.T, pos *common.Position) {
				expectedGross := fixed.FromFloat64(0.1).Mul(fixed.FromInt(100000, 0)).Mul(fixed.FromFloat64(0.005))
				assert.Equal(t, expectedGross.String(), pos.GrossProfit.String())
			},
		},
		{
			name: "short position loss calculation",
			position: &common.Position{
				Symbol:    "EURUSD",
				Side:      common.PositionSideShort,
				Size:      fixed.FromFloat64(0.2),
				OpenPrice: fixed.FromFloat64(1.1000),
				Status:    common.PositionStatusOpen,
				TimeStamp: time.Now(),
			},
			validate: func(t *testing.T, pos *common.Position) {
				expectedGross := fixed.FromFloat64(0.2).Mul(fixed.FromInt(100000, 0)).Mul(fixed.FromFloat64(-0.005))
				assert.Equal(t, expectedGross.String(), pos.GrossProfit.String())
			},
		},
		{
			name: "position with commission",
			position: &common.Position{
				Symbol:      "EURUSD",
				Side:        common.PositionSideLong,
				Size:        fixed.FromFloat64(0.1),
				OpenPrice:   fixed.FromFloat64(1.1000),
				Status:      common.PositionStatusClosed,
				Commissions: fixed.FromFloat64(5),
				TimeStamp:   time.Now(),
			},
			setup: func(sim *Simulator) {
				sim.commissionHandler = func(info exchange.SymbolInfo, pos common.Position) fixed.Point {
					return fixed.FromFloat64(2.5)
				}
			},
			validate: func(t *testing.T, pos *common.Position) {
				assert.Equal(t, fixed.FromFloat64(7.5).String(), pos.Commissions.String())
			},
		},
		{
			name: "position with slippage",
			position: &common.Position{
				Symbol:    "EURUSD",
				Side:      common.PositionSideLong,
				Size:      fixed.FromFloat64(0.1),
				OpenPrice: fixed.FromFloat64(1.1000),
				Status:    common.PositionStatusOpen,
				Slippage:  fixed.FromFloat64(0.0002),
				TimeStamp: time.Now(),
			},
			validate: func(t *testing.T, pos *common.Position) {
				closePrice := fixed.FromFloat64(1.1050)
				priceDiff := closePrice.Sub(pos.OpenPrice).Sub(pos.Slippage)
				expectedGross := priceDiff.Mul(pos.Size).Mul(fixed.FromInt(100000, 0))
				assert.Equal(t, expectedGross.String(), pos.GrossProfit.String())
			},
		},
		{
			name: "position with swap after multiple days",
			position: &common.Position{
				Symbol:    "EURUSD",
				Side:      common.PositionSideLong,
				Size:      fixed.FromFloat64(0.1),
				OpenPrice: fixed.FromFloat64(1.1000),
				Status:    common.PositionStatusOpen,
				TimeStamp: time.Now().Add(-72 * time.Hour),
			},
			setup: func(sim *Simulator) {
				sim.swapHandler = func(info exchange.SymbolInfo, pos common.Position) fixed.Point {
					return fixed.FromFloat64(1.5)
				}
			},
			validate: func(t *testing.T, pos *common.Position) {
				assert.Equal(t, fixed.FromFloat64(4.5).String(), pos.Swaps.String())
			},
		},
		{
			name: "position with exchange rate",
			position: &common.Position{
				Symbol:    "EURUSD",
				Side:      common.PositionSideLong,
				Size:      fixed.FromFloat64(0.1),
				OpenPrice: fixed.FromFloat64(1.1000),
				Status:    common.PositionStatusOpen,
				TimeStamp: time.Now(),
			},
			setup: func(sim *Simulator) {
				sim.rateProvider = &mockRateProvider{
					rate:          fixed.FromFloat64(1.2),
					conversionFee: fixed.FromFloat64(0.001),
				}
				sim.accountCurrency = "GBP"
			},
			validate: func(t *testing.T, pos *common.Position) {
				assert.Equal(t, fixed.FromFloat64(1.2).String(), pos.OpenExchangeRate.String())
				assert.Equal(t, fixed.FromFloat64(0.001).String(), pos.OpenConversionFeeRate.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			closePrice := fixed.FromFloat64(1.1050)
			sim.calcPositionProfits(tt.position, closePrice)

			if tt.validate != nil {
				tt.validate(t, tt.position)
			}
		})
	}
}

func TestSandboxSimulator_validateOrder(t *testing.T) {
	tests := []struct {
		name          string
		order         common.Order
		setup         func(*Simulator)
		expectedError string
	}{
		{
			name: "valid market order",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			setup: func(sim *Simulator) {
				sim.lastTickMap["EURUSD"] = common.Tick{
					Symbol: "EURUSD",
					Bid:    fixed.FromFloat64(1.1000),
					Ask:    fixed.FromFloat64(1.1002),
				}
			},
		},
		{
			name: "zero size order",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.Zero,
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			expectedError: "order size is zero or negative",
		},
		{
			name: "negative size order",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(-0.1),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			expectedError: "order size is zero or negative",
		},
		{
			name: "limit order without price",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceGoodTillCancel,
			},
			expectedError: "limit order must have a price",
		},
		{
			name: "market order with GTC",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceGoodTillCancel,
			},
			expectedError: "market order must not be GTC",
		},
		{
			name: "unsupported symbol",
			order: common.Order{
				Symbol:      "UNKNOWN",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			expectedError: "symbol info for symbol UNKNOWN is not supported",
		},
		{
			name: "close order without position ID",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideSell,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionClose,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			expectedError: "position ID required for close order",
		},
		{
			name: "insufficient margin",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(100),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			setup: func(sim *Simulator) {
				sim.lastTickMap["EURUSD"] = common.Tick{
					Symbol: "EURUSD",
					Bid:    fixed.FromFloat64(1.1000),
					Ask:    fixed.FromFloat64(1.1002),
				}
			},
			expectedError: "required margin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			err := sim.validateOrder(tt.order)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSandboxSimulator_OnOrder(t *testing.T) {
	tests := []struct {
		name     string
		order    common.Order
		setup    func(*Simulator)
		validate func(*testing.T, *Simulator, int, int)
	}{
		{
			name: "valid order accepted",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.1),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
				TraceID:     123,
			},
			setup: func(sim *Simulator) {
				sim.lastTickMap["EURUSD"] = common.Tick{
					Symbol: "EURUSD",
					Bid:    fixed.FromFloat64(1.1000),
					Ask:    fixed.FromFloat64(1.1002),
				}
			},
			validate: func(t *testing.T, sim *Simulator, acceptanceCount, rejectionCount int) {
				assert.Len(t, sim.openOrders, 1)
				assert.Equal(t, acceptanceCount, 1)
				assert.Equal(t, rejectionCount, 0)
			},
		},
		{
			name: "invalid order rejected",
			order: common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.Zero,
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
				TraceID:     124,
			},
			validate: func(t *testing.T, sim *Simulator, acceptanceCount, rejectionCount int) {
				assert.Empty(t, sim.openOrders)
				assert.Equal(t, acceptanceCount, 0)
				assert.Equal(t, rejectionCount, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, router := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			acceptanceCount := 0
			router.OnOrderAcceptance = func(_ context.Context, _ common.OrderAccepted) { acceptanceCount++ }

			rejectionCount := 0
			router.OnOrderRejection = func(_ context.Context, _ common.OrderRejected) { rejectionCount++ }

			sim.OnOrder(context.Background(), tt.order)
			_ = router.DrainEvents(context.Background())

			if tt.validate != nil {
				tt.validate(t, sim, acceptanceCount, rejectionCount)
			}
		})
	}
}

func TestSandboxSimulator_OnTick(t *testing.T) {
	tests := []struct {
		name     string
		tick     common.Tick
		setup    func(*Simulator)
		validate func(*testing.T, *Simulator, int, int, int)
	}{
		{
			name: "first tick posts balance and equity",
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				TimeStamp: time.Now(),
			},
			validate: func(t *testing.T, sim *Simulator, balanceCount, equityCount, filledCount int) {
				assert.True(t, sim.firstPostDone)
				assert.Equal(t, balanceCount, 1)
				assert.Equal(t, equityCount, 1)
			},
		},
		{
			name: "tick triggers position close on stop loss",
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.0949),
				Ask:       fixed.FromFloat64(1.0951),
				TimeStamp: time.Now(),
			},
			setup: func(sim *Simulator) {
				sim.firstPostDone = true
				sim.openPositions = append(sim.openPositions, &common.Position{
					Symbol:   "EURUSD",
					Side:     common.PositionSideLong,
					Size:     fixed.FromFloat64(0.1),
					StopLoss: fixed.FromFloat64(1.0950),
					Status:   common.PositionStatusOpen,
				})
			},
			validate: func(t *testing.T, sim *Simulator, _, _, _ int) {
				assert.Empty(t, sim.openPositions)
			},
		},
		{
			name: "tick executes pending market order",
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
				TimeStamp: time.Now(),
			},
			setup: func(sim *Simulator) {
				sim.firstPostDone = true
				sim.openOrders = append(sim.openOrders, &common.Order{
					Symbol:      "EURUSD",
					Side:        common.OrderSideBuy,
					Type:        common.OrderTypeMarket,
					Size:        fixed.FromFloat64(0.1),
					Command:     common.OrderCommandPositionOpen,
					TimeInForce: common.TimeInForceImmediateOrCancel,
				})
			},
			validate: func(t *testing.T, sim *Simulator, _, _, filledCount int) {
				assert.Empty(t, sim.openOrders)
				assert.Len(t, sim.openPositions, 1)
				assert.Equal(t, filledCount, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, router := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			balanceCount := 0
			router.OnBalance = func(_ context.Context, _ common.Balance) { balanceCount++ }

			equityCount := 0
			router.OnEquity = func(_ context.Context, _ common.Equity) { equityCount++ }

			filledCount := 0
			router.OnOrderFilled = func(_ context.Context, _ common.OrderFilled) { filledCount++ }

			sim.OnTick(context.Background(), tt.tick)
			_ = router.DrainEvents(context.Background())

			if tt.validate != nil {
				tt.validate(t, sim, balanceCount, equityCount, filledCount)
			}
		})
	}
}

func TestSandboxSimulator_checkMargin(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Simulator)
		validate func(*testing.T, *Simulator, int)
	}{
		{
			name: "margin call closes position",
			setup: func(sim *Simulator) {
				sim.equity = fixed.FromFloat64(100)
				sim.balance = fixed.FromFloat64(100)
				sim.freeMargin = fixed.FromFloat64(4)
				sim.maintenanceMarginRate = fixed.FromFloat64(5)
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:        1,
					Symbol:    "EURUSD",
					Side:      common.PositionSideLong,
					Size:      fixed.FromFloat64(0.1),
					OpenPrice: fixed.FromFloat64(1.1000),
					Status:    common.PositionStatusOpen,
					Margin:    fixed.FromFloat64(110),
				})
			},
			validate: func(t *testing.T, sim *Simulator, closeCount int) {
				assert.Empty(t, sim.openPositions)
				assert.Equal(t, closeCount, 1)
			},
		},
		{
			name: "sufficient margin no action",
			setup: func(sim *Simulator) {
				sim.equity = fixed.FromFloat64(1000)
				sim.balance = fixed.FromFloat64(1000)
				sim.freeMargin = fixed.FromFloat64(500)
				sim.maintenanceMarginRate = fixed.FromFloat64(5)
			},
			validate: func(t *testing.T, sim *Simulator, closeCount int) {
				assert.Equal(t, closeCount, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, router := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			tick := common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1000),
				Ask:    fixed.FromFloat64(1.1002),
			}

			closeCount := 0
			router.OnPositionClose = func(_ context.Context, _ common.Position) { closeCount++ }

			sim.checkMargin(tick)
			_ = router.DrainEvents(context.Background())

			if tt.validate != nil {
				tt.validate(t, sim, closeCount)
			}
		})
	}
}

func TestSandboxSimulator_CloseAllOpenPositions(t *testing.T) {
	sim, router := createTestSimulator(t)

	sim.lastTickMap["EURUSD"] = common.Tick{
		Symbol: "EURUSD",
		Bid:    fixed.FromFloat64(1.1000),
		Ask:    fixed.FromFloat64(1.1002),
	}
	sim.lastTickMap["GBPUSD"] = common.Tick{
		Symbol: "GBPUSD",
		Bid:    fixed.FromFloat64(1.2500),
		Ask:    fixed.FromFloat64(1.2502),
	}

	sim.openPositions = []*common.Position{
		{
			Id:        1,
			Symbol:    "EURUSD",
			Side:      common.PositionSideLong,
			Size:      fixed.FromFloat64(0.1),
			OpenPrice: fixed.FromFloat64(1.0950),
			NetProfit: fixed.FromFloat64(50),
			Status:    common.PositionStatusOpen,
		},
		{
			Id:        2,
			Symbol:    "GBPUSD",
			Side:      common.PositionSideShort,
			Size:      fixed.FromFloat64(0.2),
			OpenPrice: fixed.FromFloat64(1.2600),
			NetProfit: fixed.FromFloat64(100),
			Status:    common.PositionStatusOpen,
		},
	}

	sim.balance = fixed.FromFloat64(10000)
	sim.equity = fixed.FromFloat64(10150)

	positions := make([]common.Position, 0)
	router.OnPositionClose = func(_ context.Context, p common.Position) { positions = append(positions, p) }

	sim.CloseAllOpenPositions()
	_ = router.DrainEvents(context.Background())

	assert.Empty(t, sim.openPositions)
	assert.Equal(t, sim.balance, sim.equity)
	assert.Len(t, positions, 2)

	closedPos1 := positions[0]
	assert.Equal(t, common.PositionStatusClosed, closedPos1.Status)
	assert.Equal(t, fixed.FromFloat64(1.1000).String(), closedPos1.ClosePrice.String())

	closedPos2 := positions[1]
	assert.Equal(t, common.PositionStatusClosed, closedPos2.Status)
	assert.Equal(t, fixed.FromFloat64(1.2502).String(), closedPos2.ClosePrice.String())
}

func TestSandboxSimulator_checkOrders_TimeInForce(t *testing.T) {
	tests := []struct {
		name     string
		order    *common.Order
		tick     common.Tick
		validate func(*testing.T, *Simulator, int, int)
	}{
		{
			name: "IOC partial fill and cancel",
			order: &common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(5.0),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromFloat64(2.0),
			},
			validate: func(t *testing.T, sim *Simulator, filledCount, canceledCount int) {
				assert.Empty(t, sim.openOrders)
				assert.Len(t, sim.openPositions, 1)
				f1, _ := fixed.Two.Float64()
				f2, _ := sim.openPositions[0].Size.Float64()
				assert.Equal(t, f1, f2)
				assert.Equal(t, filledCount, 1)
				assert.Equal(t, canceledCount, 1)
			},
		},
		{
			name: "FOK all or nothing - filled",
			order: &common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(1.0),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceFillOrKill,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromFloat64(2.0),
			},
			validate: func(t *testing.T, sim *Simulator, filledCount, canceledCount int) {
				assert.Empty(t, sim.openOrders)
				assert.Len(t, sim.openPositions, 1)
				assert.Equal(t, filledCount, 1)
				assert.Equal(t, canceledCount, 0)
			},
		},
		{
			name: "FOK all or nothing - cancelled",
			order: &common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(5.0),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceFillOrKill,
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromFloat64(2.0),
			},
			validate: func(t *testing.T, sim *Simulator, filledCount, canceledCount int) {
				assert.Empty(t, sim.openOrders)
				assert.Empty(t, sim.openPositions)
				assert.Equal(t, filledCount, 0)
				assert.Equal(t, canceledCount, 1)
			},
		},
		{
			name: "GTD expired order cancelled",
			order: &common.Order{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeLimit,
				Price:       fixed.FromFloat64(1.0995),
				Size:        fixed.FromFloat64(1.0),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceGoodTillDate,
				ExpireTime:  time.Now().Add(-1 * time.Hour),
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			validate: func(t *testing.T, sim *Simulator, _, canceledCount int) {
				assert.Empty(t, sim.openOrders)
				assert.Empty(t, sim.openPositions)
				assert.Equal(t, canceledCount, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, router := createTestSimulator(t)
			sim.openOrders = append(sim.openOrders, tt.order)

			filledCount := 0
			router.OnOrderFilled = func(_ context.Context, _ common.OrderFilled) { filledCount++ }

			canceledCount := 0
			router.OnOrderCancel = func(_ context.Context, _ common.OrderCancelled) { canceledCount++ }

			sim.checkOrders(tt.tick)
			_ = router.DrainEvents(context.Background())

			if tt.validate != nil {
				tt.validate(t, sim, filledCount, canceledCount)
			}
		})
	}
}

func TestSandboxSimulator_validateTick(t *testing.T) {
	tests := []struct {
		name          string
		tick          common.Tick
		setup         func(*Simulator)
		expectedError string
	}{
		{
			name: "valid tick for existing symbol",
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				TimeStamp: time.Now(),
			},
			expectedError: "",
		},
		{
			name: "valid tick with lowercase symbol",
			tick: common.Tick{
				Symbol:    "eurusd",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				TimeStamp: time.Now(),
			},
			expectedError: "",
		},
		{
			name: "invalid tick for non-existent symbol",
			tick: common.Tick{
				Symbol:    "USDJPY",
				Bid:       fixed.FromFloat64(110.00),
				Ask:       fixed.FromFloat64(110.02),
				TimeStamp: time.Now(),
			},
			expectedError: "tick with symbol USDJPY is not present in the symbols map",
		},
		{
			name: "tick with mixed case symbol",
			tick: common.Tick{
				Symbol:    "EuRuSd",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				TimeStamp: time.Now(),
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			err := sim.validateTick(tt.tick)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSandboxSimulator_postBalance(t *testing.T) {
	sim, router := createTestSimulator(t)
	sim.balance = fixed.FromFloat64(12345.67)
	sim.simulationTime = time.Now()

	var receivedBalance common.Balance
	balanceReceived := false
	router.OnBalance = func(_ context.Context, b common.Balance) {
		receivedBalance = b
		balanceReceived = true
	}

	sim.postBalance()
	err := router.DrainEvents(context.Background())
	require.NoError(t, err)

	assert.True(t, balanceReceived)
	assert.Equal(t, simulatorComponentName, receivedBalance.Source)
	assert.Equal(t, sim.balance, receivedBalance.Value)
	assert.Equal(t, sim.simulationTime, receivedBalance.TimeStamp)
	assert.NotZero(t, receivedBalance.ExecutionId)
	assert.NotZero(t, receivedBalance.TraceID)
}

func TestSandboxSimulator_postEquity(t *testing.T) {
	sim, router := createTestSimulator(t)
	sim.equity = fixed.FromFloat64(15678.90)
	sim.simulationTime = time.Now()

	var receivedEquity common.Equity
	equityReceived := false
	router.OnEquity = func(_ context.Context, e common.Equity) {
		receivedEquity = e
		equityReceived = true
	}

	sim.postEquity()
	err := router.DrainEvents(context.Background())
	require.NoError(t, err)

	assert.True(t, equityReceived)
	assert.Equal(t, simulatorComponentName, receivedEquity.Source)
	assert.Equal(t, sim.equity, receivedEquity.Value)
	assert.Equal(t, sim.simulationTime, receivedEquity.TimeStamp)
	assert.NotZero(t, receivedEquity.ExecutionId)
	assert.NotZero(t, receivedEquity.TraceID)
}

func TestSandboxSimulator_postOrderRejected(t *testing.T) {
	sim, router := createTestSimulator(t)
	sim.simulationTime = time.Now()

	order := common.Order{
		Symbol:      "EURUSD",
		Side:        common.OrderSideBuy,
		Type:        common.OrderTypeMarket,
		Size:        fixed.FromFloat64(0.1),
		Command:     common.OrderCommandPositionOpen,
		TimeInForce: common.TimeInForceImmediateOrCancel,
		TraceID:     999,
	}

	var receivedRejection common.OrderRejected
	rejectionReceived := false
	router.OnOrderRejection = func(_ context.Context, r common.OrderRejected) {
		receivedRejection = r
		rejectionReceived = true
	}

	reason := "Insufficient margin for order execution"
	sim.postOrderRejected(order, reason)
	err := router.DrainEvents(context.Background())
	require.NoError(t, err)

	assert.True(t, rejectionReceived)
	assert.Equal(t, simulatorComponentName, receivedRejection.Source)
	assert.Equal(t, order, receivedRejection.OriginalOrder)
	assert.Equal(t, reason, receivedRejection.Reason)
	assert.Equal(t, sim.simulationTime, receivedRejection.TimeStamp)
	assert.NotZero(t, receivedRejection.ExecutionId)
	assert.NotZero(t, receivedRejection.TraceID)
}

func TestSandboxSimulator_postOrderFilled(t *testing.T) {
	sim, router := createTestSimulator(t)
	sim.simulationTime = time.Now()

	order := common.Order{
		Symbol:      "EURUSD",
		Side:        common.OrderSideBuy,
		Type:        common.OrderTypeMarket,
		Size:        fixed.FromFloat64(0.1),
		Command:     common.OrderCommandPositionOpen,
		TimeInForce: common.TimeInForceImmediateOrCancel,
		TraceID:     888,
	}

	var receivedFilled common.OrderFilled
	filledReceived := false
	router.OnOrderFilled = func(_ context.Context, f common.OrderFilled) {
		receivedFilled = f
		filledReceived = true
	}

	positionId := common.PositionId(42)
	sim.postOrderFilled(order, positionId)
	err := router.DrainEvents(context.Background())
	require.NoError(t, err)

	assert.True(t, filledReceived)
	assert.Equal(t, simulatorComponentName, receivedFilled.Source)
	assert.Equal(t, order, receivedFilled.OriginalOrder)
	assert.Equal(t, positionId, receivedFilled.PositionId)
	assert.Equal(t, sim.simulationTime, receivedFilled.TimeStamp)
	assert.NotZero(t, receivedFilled.ExecutionId)
	assert.NotZero(t, receivedFilled.TraceID)
}

func TestSandboxSimulator_postOrderCancel(t *testing.T) {
	sim, router := createTestSimulator(t)
	sim.simulationTime = time.Now()

	order := common.Order{
		Symbol:      "EURUSD",
		Side:        common.OrderSideBuy,
		Type:        common.OrderTypeLimit,
		Price:       fixed.FromFloat64(1.1000),
		Size:        fixed.FromFloat64(1.0),
		Command:     common.OrderCommandPositionOpen,
		TimeInForce: common.TimeInForceGoodTillCancel,
		TraceID:     777,
	}

	var receivedCancel common.OrderCancelled
	cancelReceived := false
	router.OnOrderCancel = func(_ context.Context, c common.OrderCancelled) {
		receivedCancel = c
		cancelReceived = true
	}

	cancelSize := fixed.FromFloat64(0.6)
	sim.postOrderCancel(order, cancelSize)
	err := router.DrainEvents(context.Background())
	require.NoError(t, err)

	assert.True(t, cancelReceived)
	assert.Equal(t, simulatorComponentName, receivedCancel.Source)
	assert.Equal(t, order, receivedCancel.OriginalOrder)
	assert.Equal(t, cancelSize, receivedCancel.CancelledSize)
	assert.Equal(t, sim.simulationTime, receivedCancel.TimeStamp)
	assert.NotZero(t, receivedCancel.ExecutionId)
	assert.NotZero(t, receivedCancel.TraceID)
}

func TestSandboxSimulator_checkPositions(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Simulator)
		tick     common.Tick
		validate func(*testing.T, *Simulator)
	}{
		{
			name: "long position hits take profit",
			setup: func(sim *Simulator) {
				sim.openPositions = []*common.Position{
					{
						Id:         1,
						Symbol:     "EURUSD",
						Side:       common.PositionSideLong,
						Size:       fixed.FromFloat64(0.1),
						Status:     common.PositionStatusOpen,
						TakeProfit: fixed.FromFloat64(1.1050),
					},
				}
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1051),
				Ask:    fixed.FromFloat64(1.1053),
			},
			validate: func(t *testing.T, sim *Simulator) {
				assert.Equal(t, positionStatusPendingClose, sim.openPositions[0].Status)
			},
		},
		{
			name: "long position hits stop loss",
			setup: func(sim *Simulator) {
				sim.openPositions = []*common.Position{
					{
						Id:       1,
						Symbol:   "EURUSD",
						Side:     common.PositionSideLong,
						Size:     fixed.FromFloat64(0.1),
						Status:   common.PositionStatusOpen,
						StopLoss: fixed.FromFloat64(1.0950),
					},
				}
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.0949),
				Ask:    fixed.FromFloat64(1.0951),
			},
			validate: func(t *testing.T, sim *Simulator) {
				assert.Equal(t, positionStatusPendingClose, sim.openPositions[0].Status)
			},
		},
		{
			name: "short position hits take profit",
			setup: func(sim *Simulator) {
				sim.openPositions = []*common.Position{
					{
						Id:         1,
						Symbol:     "EURUSD",
						Side:       common.PositionSideShort,
						Size:       fixed.FromFloat64(0.1),
						Status:     common.PositionStatusOpen,
						TakeProfit: fixed.FromFloat64(1.0950),
					},
				}
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.0947),
				Ask:    fixed.FromFloat64(1.0949),
			},
			validate: func(t *testing.T, sim *Simulator) {
				assert.Equal(t, positionStatusPendingClose, sim.openPositions[0].Status)
			},
		},
		{
			name: "position not affected by different symbol tick",
			setup: func(sim *Simulator) {
				sim.openPositions = []*common.Position{
					{
						Id:         1,
						Symbol:     "EURUSD",
						Side:       common.PositionSideLong,
						Size:       fixed.FromFloat64(0.1),
						Status:     common.PositionStatusOpen,
						TakeProfit: fixed.FromFloat64(1.1050),
					},
				}
			},
			tick: common.Tick{
				Symbol: "GBPUSD",
				Bid:    fixed.FromFloat64(1.2500),
				Ask:    fixed.FromFloat64(1.2502),
			},
			validate: func(t *testing.T, sim *Simulator) {
				assert.Equal(t, common.PositionStatusOpen, sim.openPositions[0].Status)
			},
		},
		{
			name: "multiple positions with different conditions",
			setup: func(sim *Simulator) {
				sim.openPositions = []*common.Position{
					{
						Id:         1,
						Symbol:     "EURUSD",
						Side:       common.PositionSideLong,
						Status:     common.PositionStatusOpen,
						TakeProfit: fixed.FromFloat64(1.1050),
					},
					{
						Id:       2,
						Symbol:   "EURUSD",
						Side:     common.PositionSideLong,
						Status:   common.PositionStatusOpen,
						StopLoss: fixed.FromFloat64(1.0950),
					},
					{
						Id:         3,
						Symbol:     "EURUSD",
						Side:       common.PositionSideLong,
						Status:     common.PositionStatusOpen,
						StopLoss:   fixed.FromFloat64(1.0900),
						TakeProfit: fixed.FromFloat64(1.1100),
					},
				}
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1051),
				Ask:    fixed.FromFloat64(1.1053),
			},
			validate: func(t *testing.T, sim *Simulator) {
				assert.Equal(t, positionStatusPendingClose, sim.openPositions[0].Status)
				assert.Equal(t, common.PositionStatusOpen, sim.openPositions[1].Status)
				assert.Equal(t, common.PositionStatusOpen, sim.openPositions[2].Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			sim.checkPositions(tt.tick)

			if tt.validate != nil {
				tt.validate(t, sim)
			}
		})
	}
}

func TestSandboxSimulator_processPendingChanges(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Simulator)
		tick     common.Tick
		validate func(*testing.T, *Simulator, int, int, int)
	}{
		{
			name: "process pending open position",
			setup: func(sim *Simulator) {
				sim.balance = fixed.FromFloat64(10000)
				sim.equity = fixed.FromFloat64(10000)
				sim.openPositions = []*common.Position{
					{
						Id:        1,
						Symbol:    "EURUSD",
						Side:      common.PositionSideLong,
						Size:      fixed.FromFloat64(0.1),
						Status:    positionStatusPendingOpen,
						TimeStamp: time.Now(),
					},
				}
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				TimeStamp: time.Now(),
			},
			validate: func(t *testing.T, sim *Simulator, openCount, closeCount, updateCount int) {
				assert.Len(t, sim.openPositions, 1)
				assert.Equal(t, common.PositionStatusOpen, sim.openPositions[0].Status)
				assert.Equal(t, fixed.FromFloat64(1.1002), sim.openPositions[0].OpenPrice)
				assert.Equal(t, openCount, 1)
				assert.Equal(t, closeCount, 0)
				assert.Equal(t, updateCount, 0)
			},
		},
		{
			name: "process pending close position",
			setup: func(sim *Simulator) {
				sim.balance = fixed.FromFloat64(10000)
				sim.equity = fixed.FromFloat64(10050)
				sim.openPositions = []*common.Position{
					{
						Id:        1,
						Symbol:    "EURUSD",
						Side:      common.PositionSideLong,
						Size:      fixed.FromFloat64(0.1),
						Status:    positionStatusPendingClose,
						OpenPrice: fixed.FromFloat64(1.0950),
						NetProfit: fixed.FromFloat64(50),
						TimeStamp: time.Now(),
					},
				}
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				TimeStamp: time.Now(),
			},
			validate: func(t *testing.T, sim *Simulator, openCount, closeCount, updateCount int) {
				assert.Empty(t, sim.openPositions)
				f1, _ := fixed.FromFloat64(10050).Float64()
				f2, _ := sim.balance.Float64()
				assert.Equal(t, f1, f2)
				assert.Equal(t, openCount, 0)
				assert.Equal(t, closeCount, 1)
				assert.Equal(t, updateCount, 0)
			},
		},
		{
			name: "update open position PnL",
			setup: func(sim *Simulator) {
				sim.balance = fixed.FromFloat64(10000)
				sim.equity = fixed.FromFloat64(10000)
				sim.openPositions = []*common.Position{
					{
						Id:        1,
						Symbol:    "EURUSD",
						Side:      common.PositionSideLong,
						Size:      fixed.FromFloat64(0.1),
						Status:    common.PositionStatusOpen,
						OpenPrice: fixed.FromFloat64(1.0950),
						TimeStamp: time.Now(),
					},
				}
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				TimeStamp: time.Now(),
			},
			validate: func(t *testing.T, sim *Simulator, openCount, closeCount, updateCount int) {
				assert.Len(t, sim.openPositions, 1)
				assert.True(t, sim.equity.Gt(sim.balance))
				assert.Equal(t, openCount, 0)
				assert.Equal(t, closeCount, 0)
				assert.Equal(t, updateCount, 1)
			},
		},
		{
			name: "process with slippage handler",
			setup: func(sim *Simulator) {
				sim.balance = fixed.FromFloat64(10000)
				sim.equity = fixed.FromFloat64(10000)
				sim.slippageHandler = func(p common.Position) fixed.Point {
					return fixed.FromFloat64(0.0002)
				}
				sim.openPositions = []*common.Position{
					{
						Id:        1,
						Symbol:    "EURUSD",
						Side:      common.PositionSideLong,
						Size:      fixed.FromFloat64(0.1),
						Status:    positionStatusPendingOpen,
						TimeStamp: time.Now(),
					},
				}
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				TimeStamp: time.Now(),
			},
			validate: func(t *testing.T, sim *Simulator, openCount, closeCount, updateCount int) {
				assert.Equal(t, fixed.FromFloat64(0.0002), sim.openPositions[0].Slippage)
				assert.Equal(t, openCount, 1)
			},
		},
		{
			name: "mixed positions different symbols",
			setup: func(sim *Simulator) {
				sim.balance = fixed.FromFloat64(10000)
				sim.equity = fixed.FromFloat64(10000)
				sim.openPositions = []*common.Position{
					{
						Id:        1,
						Symbol:    "EURUSD",
						Side:      common.PositionSideLong,
						Size:      fixed.FromFloat64(0.1),
						Status:    positionStatusPendingOpen,
						TimeStamp: time.Now(),
					},
					{
						Id:        2,
						Symbol:    "GBPUSD",
						Side:      common.PositionSideLong,
						Size:      fixed.FromFloat64(0.1),
						Status:    common.PositionStatusOpen,
						OpenPrice: fixed.FromFloat64(1.2500),
						TimeStamp: time.Now(),
					},
				}
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				TimeStamp: time.Now(),
			},
			validate: func(t *testing.T, sim *Simulator, openCount, closeCount, updateCount int) {
				assert.Len(t, sim.openPositions, 2)
				assert.Equal(t, common.PositionStatusOpen, sim.openPositions[0].Status)
				assert.Equal(t, common.PositionStatusOpen, sim.openPositions[1].Status)
				assert.Equal(t, openCount, 1)
				assert.Equal(t, updateCount, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, router := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			openCount := 0
			router.OnPositionOpen = func(_ context.Context, _ common.Position) { openCount++ }

			closeCount := 0
			router.OnPositionClose = func(_ context.Context, _ common.Position) { closeCount++ }

			updateCount := 0
			router.OnPositionUpdate = func(_ context.Context, _ common.Position) { updateCount++ }

			sim.processPendingChanges(tt.tick)
			_ = router.DrainEvents(context.Background())

			if tt.validate != nil {
				tt.validate(t, sim, openCount, closeCount, updateCount)
			}
		})
	}
}

func TestSandboxSimulator_calcFreeMargin(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Simulator)
		validate func(*testing.T, *Simulator)
	}{
		{
			name: "no open positions",
			setup: func(sim *Simulator) {
				sim.equity = fixed.FromFloat64(10000)
			},
			validate: func(t *testing.T, sim *Simulator) {
				sim.calcFreeMargin()
				assert.Equal(t, sim.equity, sim.freeMargin)
			},
		},
		{
			name: "single position",
			setup: func(sim *Simulator) {
				sim.equity = fixed.FromFloat64(10000)
				sim.openPositions = []*common.Position{
					{
						Margin: fixed.FromFloat64(1000),
					},
				}
			},
			validate: func(t *testing.T, sim *Simulator) {
				sim.calcFreeMargin()
				assert.Equal(t, fixed.FromFloat64(9000), sim.freeMargin)
			},
		},
		{
			name: "multiple positions",
			setup: func(sim *Simulator) {
				sim.equity = fixed.FromFloat64(10000)
				sim.openPositions = []*common.Position{
					{Margin: fixed.FromFloat64(1000)},
					{Margin: fixed.FromFloat64(500)},
					{Margin: fixed.FromFloat64(750)},
				}
			},
			validate: func(t *testing.T, sim *Simulator) {
				sim.calcFreeMargin()
				assert.Equal(t, fixed.FromFloat64(7750), sim.freeMargin)
			},
		},
		{
			name: "margin exceeds equity",
			setup: func(sim *Simulator) {
				sim.equity = fixed.FromFloat64(1000)
				sim.openPositions = []*common.Position{
					{Margin: fixed.FromFloat64(1500)},
				}
			},
			validate: func(t *testing.T, sim *Simulator) {
				sim.calcFreeMargin()
				assert.Equal(t, fixed.FromFloat64(-500), sim.freeMargin)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			if tt.validate != nil {
				tt.validate(t, sim)
			}
		})
	}
}

func TestSandboxSimulator_validateStopLossAndTakeProfit(t *testing.T) {
	tests := []struct {
		name          string
		order         common.Order
		setup         func(*Simulator)
		expectedError string
	}{
		{
			name: "valid buy order SL and TP",
			order: common.Order{
				Symbol:     "EURUSD",
				Side:       common.OrderSideBuy,
				StopLoss:   fixed.FromFloat64(1.1002),
				TakeProfit: fixed.FromFloat64(1.1050),
			},
			setup: func(sim *Simulator) {
				sim.lastTickMap["EURUSD"] = common.Tick{
					Symbol: "EURUSD",
					Bid:    fixed.FromFloat64(1.1000),
					Ask:    fixed.FromFloat64(1.1002),
				}
			},
			expectedError: "stop loss must be less than bid",
		},
		{
			name: "sell order TP above ask",
			order: common.Order{
				Symbol:     "EURUSD",
				Side:       common.OrderSideSell,
				TakeProfit: fixed.FromFloat64(1.1003),
			},
			setup: func(sim *Simulator) {
				sim.lastTickMap["EURUSD"] = common.Tick{
					Symbol: "EURUSD",
					Bid:    fixed.FromFloat64(1.1000),
					Ask:    fixed.FromFloat64(1.1002),
				}
			},
			expectedError: "take profit must be less than ask",
		},
		{
			name: "no tick for symbol",
			order: common.Order{
				Symbol:   "GBPUSD",
				Side:     common.OrderSideBuy,
				StopLoss: fixed.FromFloat64(1.2450),
			},
			expectedError: "no tick found for symbol GBPUSD",
		},
		{
			name: "valid order with only SL",
			order: common.Order{
				Symbol:   "EURUSD",
				Side:     common.OrderSideBuy,
				StopLoss: fixed.FromFloat64(1.0950),
			},
			setup: func(sim *Simulator) {
				sim.lastTickMap["EURUSD"] = common.Tick{
					Symbol: "EURUSD",
					Bid:    fixed.FromFloat64(1.1000),
					Ask:    fixed.FromFloat64(1.1002),
				}
			},
			expectedError: "",
		},
		{
			name: "valid order with only TP",
			order: common.Order{
				Symbol:     "EURUSD",
				Side:       common.OrderSideSell,
				TakeProfit: fixed.FromFloat64(1.0950),
			},
			setup: func(sim *Simulator) {
				sim.lastTickMap["EURUSD"] = common.Tick{
					Symbol: "EURUSD",
					Bid:    fixed.FromFloat64(1.1000),
					Ask:    fixed.FromFloat64(1.1002),
				}
			},
			expectedError: "",
		},
		{
			name: "order with zero SL and TP",
			order: common.Order{
				Symbol:     "EURUSD",
				Side:       common.OrderSideBuy,
				StopLoss:   fixed.Zero,
				TakeProfit: fixed.Zero,
			},
			setup: func(sim *Simulator) {
				sim.lastTickMap["EURUSD"] = common.Tick{
					Symbol: "EURUSD",
					Bid:    fixed.FromFloat64(1.1000),
					Ask:    fixed.FromFloat64(1.1002),
				}
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, _ := createTestSimulator(t)
			if tt.setup != nil {
				tt.setup(sim)
			}

			err := sim.validateStopLossAndTakeProfit(tt.order)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSandboxSimulator_NewSimulator(t *testing.T) {
	tests := []struct {
		name          string
		router        *bus.Router
		currency      string
		startBalance  fixed.Point
		options       []Option
		expectedError string
	}{
		{
			name:         "valid simulator creation",
			router:       bus.NewRouter(1000),
			currency:     "USD",
			startBalance: fixed.FromFloat64(10000),
			options: []Option{
				WithSymbols(exchange.SymbolInfo{
					SymbolName:    "EURUSD",
					QuoteCurrency: "USD",
					ContractSize:  fixed.FromInt(100000, 0),
					Leverage:      fixed.FromInt(100, 0),
				}),
			},
			expectedError: "",
		},
		{
			name:          "nil router",
			router:        nil,
			currency:      "USD",
			startBalance:  fixed.FromFloat64(10000),
			expectedError: "router is nil",
		},
		{
			name:          "empty account currency",
			router:        bus.NewRouter(1000),
			currency:      "",
			startBalance:  fixed.FromFloat64(10000),
			expectedError: "account currency not set",
		},
		{
			name:          "zero start balance",
			router:        bus.NewRouter(1000),
			currency:      "USD",
			startBalance:  fixed.Zero,
			expectedError: "start balance is invalid",
		},
		{
			name:          "negative start balance",
			router:        bus.NewRouter(1000),
			currency:      "USD",
			startBalance:  fixed.FromFloat64(-1000),
			expectedError: "start balance is invalid",
		},
		{
			name:          "empty symbols map",
			router:        bus.NewRouter(1000),
			currency:      "USD",
			startBalance:  fixed.FromFloat64(10000),
			options:       []Option{},
			expectedError: "symbol map is empty",
		},
		{
			name:         "symbol with zero leverage",
			router:       bus.NewRouter(1000),
			currency:     "USD",
			startBalance: fixed.FromFloat64(10000),
			options: []Option{
				WithSymbols(exchange.SymbolInfo{
					SymbolName:    "EURUSD",
					QuoteCurrency: "USD",
					ContractSize:  fixed.FromInt(100000, 0),
					Leverage:      fixed.Zero,
				}),
			},
			expectedError: "invalid leverage",
		},
		{
			name:         "with all options",
			router:       bus.NewRouter(1000),
			currency:     "USD",
			startBalance: fixed.FromFloat64(10000),
			options: []Option{
				WithSymbols(exchange.SymbolInfo{
					SymbolName:    "EURUSD",
					QuoteCurrency: "USD",
					ContractSize:  fixed.FromInt(100000, 0),
					Leverage:      fixed.FromInt(100, 0),
				}),
				WithRateProvider(&mockRateProvider{
					rate:          fixed.One,
					conversionFee: fixed.Zero,
				}),
				WithCommissionHandler(func(info exchange.SymbolInfo, pos common.Position) fixed.Point {
					return fixed.FromFloat64(5)
				}),
				WithSwapHandler(func(info exchange.SymbolInfo, pos common.Position) fixed.Point {
					return fixed.FromFloat64(1)
				}),
				WithSlippageHandler(func(pos common.Position) fixed.Point {
					return fixed.FromFloat64(0.0001)
				}),
				WithMaintenanceMargin(fixed.FromFloat64(10)),
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, err := NewSimulator(tt.router, tt.currency, tt.startBalance, tt.options...)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, sim)
			} else {
				require.NoError(t, err)
				require.NotNil(t, sim)
				assert.Equal(t, tt.currency, sim.accountCurrency)
				assert.Equal(t, tt.startBalance, sim.balance)
				assert.Equal(t, tt.startBalance, sim.equity)
				assert.Equal(t, tt.startBalance, sim.freeMargin)
			}
		})
	}
}

func TestSandboxSimulator_ComplexOrderScenarios(t *testing.T) {
	t.Run("partial fill with IOC then market order completion", func(t *testing.T) {
		sim, router := createTestSimulator(t)
		sim.lastTickMap["EURUSD"] = common.Tick{
			Symbol:    "EURUSD",
			Bid:       fixed.FromFloat64(1.1000),
			Ask:       fixed.FromFloat64(1.1002),
			BidVolume: fixed.FromInt(10, 0),
			AskVolume: fixed.FromFloat64(0.5),
		}

		order1 := &common.Order{
			Symbol:      "EURUSD",
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeMarket,
			Size:        fixed.FromFloat64(1.0),
			Command:     common.OrderCommandPositionOpen,
			TimeInForce: common.TimeInForceImmediateOrCancel,
		}
		sim.openOrders = append(sim.openOrders, order1)

		filledCount := 0
		cancelCount := 0
		router.OnOrderFilled = func(_ context.Context, _ common.OrderFilled) { filledCount++ }
		router.OnOrderCancel = func(_ context.Context, _ common.OrderCancelled) { cancelCount++ }

		tick := common.Tick{
			Symbol:    "EURUSD",
			Bid:       fixed.FromFloat64(1.1000),
			Ask:       fixed.FromFloat64(1.1002),
			BidVolume: fixed.FromInt(10, 0),
			AskVolume: fixed.FromFloat64(0.5),
		}

		sim.checkOrders(tick)
		_ = router.DrainEvents(context.Background())

		assert.Empty(t, sim.openOrders)
		assert.Len(t, sim.openPositions, 1)
		f1, _ := fixed.FromFloat64(0.5).Float64()
		f2, _ := sim.openPositions[0].Size.Float64()
		assert.Equal(t, f1, f2)
		assert.Equal(t, filledCount, 1)
		assert.Equal(t, cancelCount, 1)
	})

	t.Run("limit order becomes executable after price movement", func(t *testing.T) {
		sim, router := createTestSimulator(t)

		order := &common.Order{
			Symbol:      "EURUSD",
			Side:        common.OrderSideBuy,
			Type:        common.OrderTypeLimit,
			Price:       fixed.FromFloat64(1.0995),
			Size:        fixed.FromFloat64(0.5),
			Command:     common.OrderCommandPositionOpen,
			TimeInForce: common.TimeInForceGoodTillCancel,
		}
		sim.openOrders = append(sim.openOrders, order)

		tick1 := common.Tick{
			Symbol:    "EURUSD",
			Bid:       fixed.FromFloat64(1.1000),
			Ask:       fixed.FromFloat64(1.1002),
			BidVolume: fixed.FromInt(10, 0),
			AskVolume: fixed.FromInt(10, 0),
		}

		sim.checkOrders(tick1)
		assert.Len(t, sim.openOrders, 1)
		assert.Empty(t, sim.openPositions)

		tick2 := common.Tick{
			Symbol:    "EURUSD",
			Bid:       fixed.FromFloat64(1.0992),
			Ask:       fixed.FromFloat64(1.0994),
			BidVolume: fixed.FromInt(10, 0),
			AskVolume: fixed.FromInt(10, 0),
		}

		filledCount := 0
		router.OnOrderFilled = func(_ context.Context, _ common.OrderFilled) { filledCount++ }

		sim.checkOrders(tick2)
		_ = router.DrainEvents(context.Background())

		assert.Empty(t, sim.openOrders)
		assert.Len(t, sim.openPositions, 1)
		assert.Equal(t, filledCount, 1)
	})

	t.Run("position close with remaining size", func(t *testing.T) {
		sim, router := createTestSimulator(t)

		sim.openPositions = append(sim.openPositions, &common.Position{
			Id:     1,
			Symbol: "EURUSD",
			Side:   common.PositionSideLong,
			Size:   fixed.FromFloat64(1.0),
			Status: common.PositionStatusOpen,
		})

		order := &common.Order{
			Symbol:      "EURUSD",
			Side:        common.OrderSideSell,
			Type:        common.OrderTypeMarket,
			Size:        fixed.FromFloat64(0.6),
			Command:     common.OrderCommandPositionClose,
			PositionId:  1,
			TimeInForce: common.TimeInForceImmediateOrCancel,
		}
		sim.openOrders = append(sim.openOrders, order)

		tick := common.Tick{
			Symbol:    "EURUSD",
			Bid:       fixed.FromFloat64(1.1000),
			Ask:       fixed.FromFloat64(1.1002),
			BidVolume: fixed.FromFloat64(10),
			AskVolume: fixed.FromFloat64(10),
		}

		filledCount := 0
		router.OnOrderFilled = func(_ context.Context, _ common.OrderFilled) { filledCount++ }

		sim.checkOrders(tick)
		_ = router.DrainEvents(context.Background())

		assert.Empty(t, sim.openOrders)
		assert.Len(t, sim.openPositions, 2)
		assert.Equal(t, fixed.FromFloat64(0.4).String(), sim.openPositions[1].Size.String())
		assert.Equal(t, common.PositionStatusOpen, sim.openPositions[1].Status)
		assert.Equal(t, filledCount, 1)
	})
}

func TestSandboxSimulator_ErrorHandlingScenarios(t *testing.T) {
	t.Run("router post errors are handled gracefully", func(t *testing.T) {
		sim, router := createTestSimulator(t)

		for i := 0; i < 1001; i++ {
			_ = router.Post(bus.BalanceEvent, common.Balance{})
		}

		sim.postBalance()
		sim.postEquity()

		order := common.Order{
			Symbol: "EURUSD",
			Side:   common.OrderSideBuy,
		}
		sim.postOrderRejected(order, "test rejection")
		sim.postOrderFilled(order, 1)
		sim.postOrderCancel(order, fixed.One)
	})

	t.Run("margin call with router error retries", func(t *testing.T) {
		sim, router := createTestSimulator(t)
		sim.equity = fixed.FromFloat64(100)
		sim.balance = fixed.FromFloat64(100)
		sim.freeMargin = fixed.FromFloat64(4)
		sim.maintenanceMarginRate = fixed.FromFloat64(5)

		sim.openPositions = append(sim.openPositions, &common.Position{
			Id:        1,
			Symbol:    "EURUSD",
			Side:      common.PositionSideLong,
			Size:      fixed.FromFloat64(0.1),
			OpenPrice: fixed.FromFloat64(1.1000),
			Status:    common.PositionStatusOpen,
			Margin:    fixed.FromFloat64(110),
			NetProfit: fixed.FromFloat64(-10),
		})

		for i := 0; i < 1001; i++ {
			_ = router.Post(bus.BalanceEvent, common.Balance{})
		}

		tick := common.Tick{
			Symbol: "EURUSD",
			Bid:    fixed.FromFloat64(1.1000),
			Ask:    fixed.FromFloat64(1.1002),
		}

		sim.checkMargin(tick)
		assert.Len(t, sim.openPositions, 1)
	})
}

func TestSandboxSimulator_checkOrders_TimeInForce_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Simulator)
		orders   []*common.Order
		tick     common.Tick
		validate func(*testing.T, *Simulator, map[string]int)
	}{
		{
			name: "Market IOC - full fill",
			orders: []*common.Order{{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(0.5),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			}},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			validate: func(t *testing.T, sim *Simulator, events map[string]int) {
				assert.Empty(t, sim.openOrders)
				assert.Len(t, sim.openPositions, 1)
				assert.Equal(t, 1, events["filled"])
			},
		},
		{
			name: "Order with partial fill from previous attempt",
			orders: []*common.Order{{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(3.0),
				FilledSize:  fixed.FromFloat64(1.0),
				Command:     common.OrderCommandPositionOpen,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			}},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromFloat64(1.5),
			},
			validate: func(t *testing.T, sim *Simulator, events map[string]int) {
				assert.Empty(t, sim.openOrders)
				assert.Len(t, sim.openPositions, 1)
				f1, _ := fixed.FromFloat64(1.5).Float64()
				f2, _ := sim.openPositions[0].Size.Float64()
				assert.Equal(t, f1, f2)
				assert.Equal(t, 1, events["filled"])
				assert.Equal(t, 1, events["cancelled"])
			},
		},
		{
			name: "Position modify order - not affected by TIF",
			orders: []*common.Order{{
				Symbol:      "EURUSD",
				Command:     common.OrderCommandPositionModify,
				PositionId:  1,
				StopLoss:    fixed.FromFloat64(1.0950),
				TimeInForce: common.TimeInForceImmediateOrCancel,
			}},
			setup: func(sim *Simulator) {
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:     1,
					Symbol: "EURUSD",
					Side:   common.PositionSideLong,
					Size:   fixed.FromFloat64(1.0),
					Status: common.PositionStatusOpen,
				})
			},
			tick: common.Tick{
				Symbol: "EURUSD",
				Bid:    fixed.FromFloat64(1.1000),
				Ask:    fixed.FromFloat64(1.1002),
			},
			validate: func(t *testing.T, sim *Simulator, events map[string]int) {
				assert.Empty(t, sim.openOrders)
				assert.Equal(t, fixed.FromFloat64(1.0950).String(), sim.openPositions[0].StopLoss.String())
				assert.Equal(t, 0, events["filled"])
				assert.Equal(t, 0, events["cancelled"])
			},
		},
		{
			name: "Different symbols - only matching processed",
			orders: []*common.Order{
				{
					Symbol:      "EURUSD",
					Side:        common.OrderSideBuy,
					Type:        common.OrderTypeMarket,
					Size:        fixed.FromFloat64(1.0),
					Command:     common.OrderCommandPositionOpen,
					TimeInForce: common.TimeInForceImmediateOrCancel,
				},
				{
					Symbol:      "GBPUSD",
					Side:        common.OrderSideBuy,
					Type:        common.OrderTypeMarket,
					Size:        fixed.FromFloat64(1.0),
					Command:     common.OrderCommandPositionOpen,
					TimeInForce: common.TimeInForceImmediateOrCancel,
				},
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			validate: func(t *testing.T, sim *Simulator, events map[string]int) {
				assert.Len(t, sim.openOrders, 1)
				assert.Equal(t, "GBPUSD", sim.openOrders[0].Symbol)
				assert.Len(t, sim.openPositions, 1)
				assert.Equal(t, 1, events["filled"])
			},
		},
		{
			name: "Zero liquidity with different TIF behaviors",
			orders: []*common.Order{
				{
					Symbol:      "EURUSD",
					Side:        common.OrderSideBuy,
					Type:        common.OrderTypeMarket,
					Size:        fixed.FromFloat64(1.0),
					Command:     common.OrderCommandPositionOpen,
					TimeInForce: common.TimeInForceImmediateOrCancel,
				},
				{
					Symbol:      "EURUSD",
					Side:        common.OrderSideBuy,
					Type:        common.OrderTypeLimit,
					Price:       fixed.FromFloat64(1.1003),
					Size:        fixed.FromFloat64(1.0),
					Command:     common.OrderCommandPositionOpen,
					TimeInForce: common.TimeInForceGoodTillCancel,
				},
			},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.Zero,
			},
			validate: func(t *testing.T, sim *Simulator, events map[string]int) {
				assert.Empty(t, sim.openOrders)
				assert.Empty(t, sim.openPositions)
				assert.Equal(t, 0, events["filled"])
				assert.Equal(t, 0, events["cancelled"])
				assert.Equal(t, 2, events["rejected"])
			},
		},
		{
			name: "Short position close with buy order",
			setup: func(sim *Simulator) {
				sim.openPositions = append(sim.openPositions, &common.Position{
					Id:     1,
					Symbol: "EURUSD",
					Side:   common.PositionSideShort,
					Size:   fixed.FromFloat64(2.0),
					Status: common.PositionStatusOpen,
				})
			},
			orders: []*common.Order{{
				Symbol:      "EURUSD",
				Side:        common.OrderSideBuy,
				Type:        common.OrderTypeMarket,
				Size:        fixed.FromFloat64(2.0),
				Command:     common.OrderCommandPositionClose,
				PositionId:  1,
				TimeInForce: common.TimeInForceImmediateOrCancel,
			}},
			tick: common.Tick{
				Symbol:    "EURUSD",
				Bid:       fixed.FromFloat64(1.1000),
				Ask:       fixed.FromFloat64(1.1002),
				BidVolume: fixed.FromInt(10, 0),
				AskVolume: fixed.FromInt(10, 0),
			},
			validate: func(t *testing.T, sim *Simulator, events map[string]int) {
				assert.Empty(t, sim.openOrders)
				assert.Len(t, sim.openPositions, 1)
				assert.Equal(t, positionStatusPendingClose, sim.openPositions[0].Status)
				assert.Equal(t, 1, events["filled"])
			},
		},
		// ToDo: Enable this test when volume issues are solved
		//{
		//	name: "Complex scenario - multiple partially filled orders",
		//	setup: func(sim *Simulator) {
		//		sim.simulationTime = time.Now()
		//	},
		//	orders: []*common.Order{
		//		{
		//			Symbol:      "EURUSD",
		//			Side:        common.OrderSideBuy,
		//			Type:        common.OrderTypeMarket,
		//			Size:        fixed.FromFloat64(3.0),
		//			FilledSize:  fixed.FromFloat64(0.5),
		//			Command:     common.OrderCommandPositionOpen,
		//			TimeInForce: 0,
		//		},
		//		{
		//			Symbol:      "EURUSD",
		//			Side:        common.OrderSideBuy,
		//			Type:        common.OrderTypeLimit,
		//			Price:       fixed.FromFloat64(1.1003),
		//			Size:        fixed.FromFloat64(2.0),
		//			FilledSize:  fixed.FromFloat64(0.3),
		//			Command:     common.OrderCommandPositionOpen,
		//			TimeInForce: common.TimeInForceGoodTillDate,
		//			ExpireTime:  time.Now().Add(1 * time.Hour),
		//		},
		//	},
		//	tick: common.Tick{
		//		Symbol:    "EURUSD",
		//		Bid:       fixed.FromFloat64(1.1000),
		//		Ask:       fixed.FromFloat64(1.1002),
		//		BidVolume: fixed.FromInt(10, 0),
		//		AskVolume: fixed.FromFloat64(1.5),
		//	},
		//	validate: func(t *testing.T, sim *Simulator, events map[string]int) {
		//		assert.Len(t, sim.openOrders, 2)
		//		f1, _ := fixed.FromFloat64(2.0).Float64()
		//		f2, _ := sim.openOrders[0].FilledSize.Float64()
		//		assert.Equal(t, f1, f2)
		//		f3, _ := fixed.FromFloat64(0.3).Float64()
		//		f4, _ := sim.openOrders[1].FilledSize.Float64()
		//		assert.Equal(t, f3, f4)
		//		assert.Len(t, sim.openPositions, 1)
		//		f5, _ := fixed.FromFloat64(1.5).Float64()
		//		f6, _ := sim.openPositions[0].Size.Float64()
		//		assert.Equal(t, f5, f6)
		//		assert.Equal(t, 1, events["filled"])
		//	},
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, router := createTestSimulator(t)

			if tt.setup != nil {
				tt.setup(sim)
			}

			for _, order := range tt.orders {
				sim.openOrders = append(sim.openOrders, order)
			}

			events := map[string]int{
				"filled":    0,
				"cancelled": 0,
				"rejected":  0,
				"accepted":  0,
			}

			router.OnOrderFilled = func(_ context.Context, _ common.OrderFilled) {
				events["filled"]++
			}
			router.OnOrderCancel = func(_ context.Context, _ common.OrderCancelled) {
				events["cancelled"]++
			}
			router.OnOrderRejection = func(_ context.Context, _ common.OrderRejected) {
				events["rejected"]++
			}
			router.OnOrderAcceptance = func(_ context.Context, _ common.OrderAccepted) {
				events["accepted"]++
			}

			sim.checkOrders(tt.tick)
			_ = router.DrainEvents(context.Background())

			if tt.validate != nil {
				tt.validate(t, sim, events)
			}
		})
	}
}
