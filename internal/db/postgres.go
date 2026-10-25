package db

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

func Open(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}
	d, e := sql.Open("pgx", dsn)
	if e != nil {
		return nil, e
	}
	d.SetMaxOpenConns(10)
	d.SetMaxIdleConns(5)
	d.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e = d.PingContext(ctx); e != nil {
		d.Close()
		return nil, e
	}
	return d, nil
}
