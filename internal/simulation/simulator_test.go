package simulation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/utility"
)

// TODO: Finish tests

func TestSimulator_OpenAndClosePosition(t *testing.T) {
	logger := zap.NewNop()
	router := bus.NewRouter(logger, 10)
	audit := NewAudit(logger, time.Minute)
	sim := NewSimulator(logger, router, audit)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture position opened/closed events
	router.PositionOpenedHandler = func(p *model.Position) error {
		t.Logf("Position opened: %+v", p)
		return nil
	}
	router.PositionClosedHandler = func(p *model.Position) error {
		t.Logf("Position closed: %+v", p)
		return nil
	}
	// Required handlers
	router.OrderHandler = sim.OnOrder
	router.TickHandler = sim.OnTick
	router.PositionOpenedHandler = func(p *model.Position) error { return nil }
	router.PositionClosedHandler = func(p *model.Position) error { return nil }
	router.PositionPnLUpdatedHandler = func(p *model.Position) error { return nil }
	router.BalanceHandler = func(b *utility.Fixed) error { return nil }
	router.EquityHandler = func(e *utility.Fixed) error { return nil }
	router.BarHandler = func(bar *model.Bar) error { return nil }

	go router.Exec(ctx, func(ctx context.Context) error {
		return nil
	})

	// Post open order
	order := &model.Order{
		Command:   model.CmdOpen,
		OrderType: model.Market,
		Size:      utility.MustNewFixed(1, 2), // 0.01 lot
	}

	// Tick that triggers open
	tick1 := &model.Tick{
		Bid:       utility.MustNewFixed(1_0000, 4),
		Ask:       utility.MustNewFixed(1_0001, 4),
		TimeStamp: time.Now().UnixNano(),
	}
	sim.OnOrder(order)
	sim.OnTick(tick1)
	require.Len(t, sim.OpenPositions(), 1)

	// Assign take profit to force close next tick
	sim.OpenPositions()[0].TakeProfit = utility.MustNewFixed(1_0003, 4)

	// Tick that triggers close
	tick2 := &model.Tick{
		Bid:       utility.MustNewFixed(1_0003, 4),
		Ask:       utility.MustNewFixed(1_0004, 4),
		TimeStamp: time.Now().Add(1 * time.Minute).UnixNano(),
	}
	sim.OnTick(tick2)
	require.Len(t, sim.OpenPositions(), 0)

	// Balance should be close to start +/- slippage & commission
	finalBalance := sim.Balance()
	t.Logf("Final balance: %s", finalBalance.String())
	require.True(t, finalBalance.Lte(utility.TenThousandFixed))
}

func (simulator *Simulator) OpenPositions() []*model.Position {
	return simulator.openPositions // ✅ correct field access
}

func (simulator *Simulator) Balance() utility.Fixed {
	return simulator.GetBalance()
}

// Optionally expose Balance for testability
func (simulator *Simulator) GetBalance() utility.Fixed {
	return simulator.balance
}
