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
	db        *sql.DB
	appId     int64
	accountId int64
}

func NewLedger(db *sql.DB, appId, accountId int64) *Ledger {
	return &Ledger{
		db:        db,
		appId:     appId,
		accountId: accountId,
	}
}

func (l *Ledger) WithPositionClosed(handler bus.PositionClosedEventHandler) bus.PositionClosedEventHandler {
	return func(ctx context.Context, position common.Position) {
		go func() {
			if err := psql.InsertPosition(ctx, l.db, l.appId, l.accountId, position); err != nil {
				slog.Warn("unable to insert position", "error", err)
			}
		}()
		handler(ctx, position)
	}
}
