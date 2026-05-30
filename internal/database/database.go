package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type PoolOptions struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

func Open(ctx context.Context, postgresURI string, pool PoolOptions) (*sql.DB, error) {
	db, err := sql.Open("postgres", postgresURI)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if pool.MaxOpenConns > 0 {
		db.SetMaxOpenConns(pool.MaxOpenConns)
	}

	if pool.MaxIdleConns > 0 {
		maxIdleConns := pool.MaxIdleConns
		if pool.MaxOpenConns > 0 && maxIdleConns > pool.MaxOpenConns {
			maxIdleConns = pool.MaxOpenConns
		}
		db.SetMaxIdleConns(maxIdleConns)
	}

	if pool.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(pool.ConnMaxIdleTime)
	}

	if pool.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(pool.ConnMaxLifetime)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

func Setup(ctx context.Context, db *sql.DB) error {
	query := `
CREATE TABLE IF NOT EXISTS whatsapp_subscribers (
    jid TEXT PRIMARY KEY,
    phone_number TEXT NOT NULL,
    push_name TEXT NOT NULL DEFAULT '',
    subscribed BOOLEAN NOT NULL DEFAULT TRUE,
    subscribed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unsubscribed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_subscribers_active
    ON whatsapp_subscribers (subscribed);
`

	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("create subscriber tables: %w", err)
	}

	return nil
}
