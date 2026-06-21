// Package db establishes the pgxpool connection used for all database
// access, with pgvector type registration applied to every connection.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

// Connect creates a pgxpool.Pool for databaseURL, registering pgvector's
// types on every new connection, and verifies connectivity with a ping.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}

	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Tolerate a missing "vector" type: on a brand-new database it isn't
		// registered until `gosim migrate` runs CREATE EXTENSION vector, and
		// migrate needs a working connection to get that far in the first
		// place. Once the extension exists, the next pool (next command
		// invocation) will register it normally.
		_ = pgxvec.RegisterTypes(ctx, conn)
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: connect: %w", err)
	}

	return pool, nil
}
