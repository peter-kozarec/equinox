package simulation

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/model"
	"time"
)

type AccountSnapshot struct {
	Balance model.Balance
	Equity  model.Equity
	Time    time.Time
}

type Audit struct {
	logger *zap.Logger

	closedPositions  []model.Position
	accountSnapshots []AccountSnapshot
}

func NewAudit(logger *zap.Logger) *Audit {
	return &Audit{
		logger: logger,
	}
}

func (audit *Audit) ClosePosition(position model.Position) {
	audit.closedPositions = append(audit.closedPositions, position)
}

func (audit *Audit) SnapshotAccountState(balance model.Balance, equity model.Equity, t time.Time) {

	if len(audit.accountSnapshots) != 0 {
		lastSnapshotTime := audit.accountSnapshots[len(audit.accountSnapshots)-1].Time

		if t.Sub(lastSnapshotTime) < AccountSnapshotInterval {
			// Do nothing if snapshot interval has not passed
			return
		}
	}

	audit.accountSnapshots = append(audit.accountSnapshots, AccountSnapshot{
		Balance: balance,
		Equity:  equity,
		Time:    t,
	})
}
