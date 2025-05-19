package simulation

import (
	"go.uber.org/zap"
	"peter-kozarec/equinox/internal/model"
	"peter-kozarec/equinox/internal/utility"
	"time"
)

type accountSnapshot struct {
	balance utility.Fixed
	equity  utility.Fixed
	t       time.Time
}

type Audit struct {
	logger *zap.Logger

	minSnapshotInterval time.Duration

	accountSnapshots []accountSnapshot
	closedPositions  []model.Position
}

func NewAudit(logger *zap.Logger, minSnapshotInterval time.Duration) *Audit {
	return &Audit{
		logger:              logger,
		minSnapshotInterval: minSnapshotInterval,
	}
}

func (audit *Audit) SnapshotAccount(balance, equity utility.Fixed, t time.Time) {
	if len(audit.accountSnapshots) == 0 ||
		t.Sub(audit.accountSnapshots[len(audit.accountSnapshots)-1].t) < audit.minSnapshotInterval {
		audit.accountSnapshots = append(audit.accountSnapshots, accountSnapshot{
			balance: balance,
			equity:  equity,
			t:       t,
		})
	}
}

func (audit *Audit) AddClosedPosition(position model.Position) {
	audit.closedPositions = append(audit.closedPositions, position)
}

func (audit *Audit) GenerateReport() Report {
	return Report{}
}
