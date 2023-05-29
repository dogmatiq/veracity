package postgres

import (
	"context"
	"database/sql"

	"github.com/dogmatiq/veracity/persistence/kv"
)

// KeyValueStore is an implementation of [kv.Store] that stores keyspaces in a
// PostgreSQL database.
type KeyValueStore struct {
	DB *sql.DB
}

// Open returns the keyspace with the given name.
func (s *KeyValueStore) Open(ctx context.Context, name string) (kv.Keyspace, error) {
	return &keyspace{
		Name: name,
		DB:   s.DB,
	}, nil
}

type keyspace struct {
	Name string
	DB   *sql.DB
}

func (ks *keyspace) Get(ctx context.Context, k []byte) (v []byte, err error) {
	row := ks.DB.QueryRowContext(
		ctx,
		`SELECT
			value
		FROM veracity.kv
		WHERE keyspace = $1
		AND key = $2`,
		ks.Name,
		k,
	)

	var value []byte
	err = row.Scan(&value)
	if err == sql.ErrNoRows {
		err = nil
	}

	return value, err
}

func (ks *keyspace) Has(ctx context.Context, k []byte) (ok bool, err error) {
	row := ks.DB.QueryRowContext(
		ctx,
		`SELECT
			1
		FROM veracity.kv
		WHERE keyspace = $1
		AND key = $2`,
		ks.Name,
		k,
	)

	var value []byte
	err = row.Scan(&value)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return true, err
}

func (ks *keyspace) Set(ctx context.Context, k, v []byte) error {
	if len(v) == 0 {
		_, err := ks.DB.ExecContext(
			ctx,
			`DELETE FROM veracity.kv
			WHERE keyspace = $1
			AND key = $2`,
			ks.Name,
			k,
		)

		return err
	}

	_, err := ks.DB.ExecContext(
		ctx,
		`INSERT INTO veracity.kv AS o (
			keyspace,
			key,
			value
		) VALUES (
			$1, $2, $3
		) ON CONFLICT (keyspace, key) DO UPDATE SET
			value = $3
		`,
		ks.Name,
		k,
		v,
	)

	return err
}

func (ks *keyspace) RangeAll(
	ctx context.Context,
	fn kv.RangeFunc,
) error {
	rows, err := ks.DB.QueryContext(
		ctx,
		`SELECT
			key,
			value
		FROM veracity.kv
		WHERE keyspace = $1`,
		ks.Name,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			k []byte
			v []byte
		)
		if err = rows.Scan(&k, &v); err != nil {
			return err
		}

		ok, err := fn(ctx, k, v)
		if !ok || err != nil {
			return err
		}
	}

	return rows.Err()
}

func (ks *keyspace) Close() error {
	return nil
}

// CreateKeyValueStoreSchema creates the PostgreSQL schema elements required by
// [KeyValueStore].
func CreateKeyValueStoreSchema(
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
		`CREATE TABLE IF NOT EXISTS veracity.kv (
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
