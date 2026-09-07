package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Connect(ctx context.Context, databaseURL string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// Kept low on purpose: on a serverless host each cold start opens its own
	// pool, so a handful of connections per instance avoids exhausting the
	// database's (or its pooler's) connection limit under concurrent cold
	// starts. Harmless on a persistent host too.
	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(5 * time.Minute)

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err := runMigrations(ctx, conn); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return conn, nil
}

func runMigrations(ctx context.Context, conn *sql.DB) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := conn.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}
