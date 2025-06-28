package psql

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/peter-kozarec/equinox/pkg/common"
)

func Connect(ctx context.Context, host, port, user, pass, db string) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, db)
	dbConn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := dbConn.PingContext(ctx); err != nil {
		return nil, err
	}

	return dbConn, nil
}

func InsertPosition(ctx context.Context, db *sql.DB, appId, accountId int64, position common.Position) error {
	query := `
	INSERT INTO fx_positions (
		position_id,
		app_id,
		account_id,
		open_time,
		close_time,
		size,
		open_price,
		close_price,
	    gross_profit,
	    net_profit
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	ON CONFLICT (position_id, app_id, account_id) DO NOTHING;
	`

	_, err := db.ExecContext(
		ctx,
		query,
		position.Id,
		appId,
		accountId,
		position.OpenTime,
		position.CloseTime,
		position.Size.Float64(),
		position.OpenPrice.Float64(),
		position.ClosePrice.Float64(),
		position.GrossProfit.Float64(),
		position.NetProfit.Float64(),
	)

	return err
}
