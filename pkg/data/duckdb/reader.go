package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/marcboeker/go-duckdb"
	"peter-kozarec/equinox/pkg/model"
	"time"
)

type Reader struct {
	dataSourceName string
	db             *sql.DB
}

func NewReader(dataSourceName string) *Reader {
	return &Reader{
		dataSourceName: dataSourceName,
	}
}

func (r *Reader) Connect() error {
	db, err := sql.Open("duckdb", r.dataSourceName)
	if err != nil {
		return fmt.Errorf("sql.CmdOpen: %v", err)
	}
	r.db = db
	return nil
}

func (r *Reader) Close() {
	_ = r.db.Close()
}

func (r *Reader) LoadTicks(ctx context.Context, symbol string, from, to time.Time, handler func(tick model.Tick) error) error {

	query := fmt.Sprintf(`SELECT ts, ask, bid, ask_volume, bid_volume FROM %s_ticks WHERE ts BETWEEN ? AND ?`, symbol)

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return fmt.Errorf("error preparing query: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			panic(err)
		}
	}(rows)

	for rows.Next() {
		var tick model.Tick
		timeStamp := time.Time{}
		err := rows.Scan(&timeStamp, &tick.Ask, &tick.Bid, &tick.AskVolume, &tick.BidVolume)
		tick.TimeStamp = timeStamp.UnixNano()
		if err != nil {
			return fmt.Errorf("error scanning row: %w", err)
		}
		err = handler(tick)
		if err != nil {
			return fmt.Errorf("error processing tick: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error scanning rows: %w", err)
	}

	return nil
}
