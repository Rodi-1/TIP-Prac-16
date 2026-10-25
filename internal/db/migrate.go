package db

import (
	"context"
	"database/sql"
	_ "embed"
	"time"
)

//go:embed schema.sql
var Schema string

func ApplyMigrations(d *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, e := d.ExecContext(ctx, Schema)
	return e
}
