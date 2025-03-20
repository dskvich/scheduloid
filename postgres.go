package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/uptrace/bun/driver/pgdriver"
)

const (
	dbName = "app"

	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 25
	defaultConnMaxLifetime = 5 * time.Minute
)

func NewPostgres(url, host string) (*sql.DB, error) {
	if url == "" {
		url = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbName, dbName, host, dbName)
	}
	slog.Info("database connection string", "url", url)

	db := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(url)))
	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return db, nil
}
