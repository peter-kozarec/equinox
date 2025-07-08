package middleware

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/peter-kozarec/equinox/pkg/bus"
	"github.com/peter-kozarec/equinox/pkg/common"
	"github.com/peter-kozarec/equinox/pkg/data/db/psql"
)

type Ledger struct {
	ctx       context.Context
	db        *sql.DB
	appId     int64
	accountId int64
}

func NewLedger(ctx context.Context, db *sql.DB, appId, accountId int64) *Ledger {
	return &Ledger{
		ctx:       ctx,
		db:        db,
		appId:     appId,
		accountId: accountId,
	}
}

func (l *Ledger) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(position common.Position) {
		go func() {
			if err := psql.InsertPosition(l.ctx, l.db, l.appId, l.accountId, position); err != nil {
				slog.Warn("unable to insert position", "error", err)
			}
		}()
		handler(position)
	}
}
