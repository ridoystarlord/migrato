package database

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ridoystarlord/migrato/utils"
)

var (
	pool     *pgxpool.Pool
	poolOnce sync.Once
	poolErr  error
)

// GetPool returns a singleton connection pool for the application
func GetPool() (*pgxpool.Pool, error) {
	poolOnce.Do(func() {
		utils.LoadEnv()
		connStr := os.Getenv("DATABASE_URL")
		if connStr == "" {
			poolErr = fmt.Errorf("DATABASE_URL not set in environment")
			return
		}

		ctx := context.Background()
		pool, poolErr = pgxpool.New(ctx, connStr)
		if poolErr != nil {
			poolErr = fmt.Errorf("unable to create connection pool: %v", poolErr)
			return
		}

		// Test the connection
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			poolErr = fmt.Errorf("unable to ping database: %v", err)
			return
		}
	})

	return pool, poolErr
}

// GetConnection returns a single connection from the pool
func GetConnection(ctx context.Context) (*pgx.Conn, error) {
	pool, err := GetPool()
	if err != nil {
		return nil, err
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to acquire connection: %v", err)
	}

	return conn.Conn(), nil
}

// ClosePool closes the connection pool (should be called on application shutdown)
func ClosePool() {
	if pool != nil {
		pool.Close()
	}
} 