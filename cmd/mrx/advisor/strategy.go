package advisor

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/bus"
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/utility"
)

type Strategy struct {
	logger     *zap.Logger
	router     *bus.Router
	barHistory []*model.Bar
	inPosition bool
	positionId model.PositionId
}

func NewStrategy(logger *zap.Logger, router *bus.Router) *Strategy {
	return &Strategy{
		logger:     logger,
		router:     router,
		barHistory: make([]*model.Bar, 0, 60),
	}
}

func (s *Strategy) OnBar(bar *model.Bar) error {
	s.barHistory = append(s.barHistory, bar)
	if len(s.barHistory) > 60 {
		s.barHistory = s.barHistory[1:]
	}

	if len(s.barHistory) < 60 {
		return nil
	}

	// Calculate mean and standard deviation of Close prices
	var (
		sum    utility.Fixed
		closes []utility.Fixed
	)
	for _, b := range s.barHistory {
		sum = sum.Add(b.Close)
		closes = append(closes, b.Close)
	}
	mean := sum.DivInt(len(s.barHistory))

	variance := utility.ZeroFixed
	for _, c := range closes {
		diff := c.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	stdDev := variance.DivInt(len(closes)).Sqrt()

	price := bar.Close

	// Entry: price << mean - 2×stdDev
	if !s.inPosition && price.Lt(mean.Sub(stdDev.MulInt(2))) {
		order := model.Order{
			Command:   model.CmdOpen,
			OrderType: model.Market,
			Size:      utility.MustNewFixed(1, 2),
		}
		_ = s.router.Post(bus.OrderEvent, &order)
		s.inPosition = true
		s.logger.Info("Mean reversion long entry", zap.String("price", price.String()), zap.String("mean", mean.String()))
		return nil
	}

	// Exit: price >= mean
	if s.inPosition && price.Gte(mean) {
		order := model.Order{
			Command:    model.CmdClose,
			OrderType:  model.Market,
			PositionId: s.positionId,
		}
		_ = s.router.Post(bus.OrderEvent, &order)
		s.inPosition = false
		s.logger.Info("Mean reversion exit", zap.String("price", price.String()), zap.String("mean", mean.String()))
	}

	return nil
}

func (s *Strategy) OnPositionOpened(position *model.Position) error {
	s.positionId = position.Id
	return nil
}
func (s *Strategy) OnPositionClosed(_ *model.Position) error     { return nil }
func (s *Strategy) OnTick(_ *model.Tick) error                   { return nil }
func (s *Strategy) OnBalance(_ *utility.Fixed) error             { return nil }
func (s *Strategy) OnEquity(_ *utility.Fixed) error              { return nil }
func (s *Strategy) OnPositionPnlUpdated(_ *model.Position) error { return nil }
