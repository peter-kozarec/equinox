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
		barHistory: make([]*model.Bar, 0, 3),
	}
}

func (s *Strategy) OnBar(bar *model.Bar) error {
	s.barHistory = append(s.barHistory, bar)
	if len(s.barHistory) > 3 {
		s.barHistory = s.barHistory[1:]
	}

	if len(s.barHistory) < 3 {
		return nil
	}

	close0 := s.barHistory[0].Close
	close1 := s.barHistory[1].Close
	close2 := s.barHistory[2].Close
	avg := close0.Add(close1).Add(close2).DivInt(3)

	if !s.inPosition && close2.Gt(avg) {
		order := model.Order{
			Command:   model.CmdOpen,
			OrderType: model.Market,
			Size:      utility.MustNewFixed(1, 0),
		}
		_ = s.router.Post(bus.OrderEvent, &order)
		s.inPosition = true
	} else if s.inPosition && close2.Lt(avg) {
		order := model.Order{
			Command:    model.CmdClose,
			OrderType:  model.Market,
			PositionId: s.positionId,
		}
		_ = s.router.Post(bus.OrderEvent, &order)
		s.inPosition = false
	}

	return nil
}

func (s *Strategy) OnPositionOpened(position *model.Position) error {
	s.positionId = position.Id
	return nil
}

func (s *Strategy) OnPositionClosed(position *model.Position) error {
	return nil
}

func (s *Strategy) OnTick(_ *model.Tick) error                   { return nil }
func (s *Strategy) OnBalance(_ *utility.Fixed) error             { return nil }
func (s *Strategy) OnEquity(_ *utility.Fixed) error              { return nil }
func (s *Strategy) OnPositionPnlUpdated(_ *model.Position) error { return nil }
