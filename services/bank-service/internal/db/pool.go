package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps pgxpool.Pool to provide database connection pooling.
type Pool struct {
	*pgxpool.Pool
}

// NewPool creates a new database connection pool.
// The connection string should be in the format:
// postgres://username:password@host:port/database?sslmode=disable
func NewPool(ctx context.Context, connString string) (*Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure pool settings
	config.MaxConns = 25       // Maximum number of connections
	config.MinConns = 5        // Minimum number of connections to maintain
	config.MaxConnLifetime = 0 // Unlimited connection lifetime
	config.MaxConnIdleTime = 0 // Unlimited idle time
	// config.HealthCheckPeriod uses default (1 minute) if not set

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Pool{Pool: pool}, nil
}

// Close closes the database connection pool.
func (p *Pool) Close() {
	p.Pool.Close()
}
