package postgres

import (
	"context"
	"database/sql"
)

// CreateKVStoreSchema creates a PostgreSQL schema for storing key-value store
// records.
func CreateKVStoreSchema(
	ctx context.Context,
	db *sql.DB,
) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint:errcheck

	if _, err := db.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS veracity`); err != nil {
		return err
	}

	if _, err := db.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS veracity.kvstore (
			keyspace TEXT NOT NULL,
			key      BYTEA NOT NULL,
			value    BYTEA NOT NULL,

			PRIMARY KEY (keyspace, key)
		)`,
	); err != nil {
		return err
	}

	return nil
}
