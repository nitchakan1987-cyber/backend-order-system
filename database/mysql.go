package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func Open(ctx context.Context, dataSourceName string) (*sql.DB, error) {
	pool, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(1)

	if err := pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect database: %w", err)
	}

	return pool, nil
}
