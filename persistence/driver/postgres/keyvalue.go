package postgres

import (
	"context"
	"database/sql"

	"github.com/dogmatiq/veracity/persistence/internal/pathkey"
	"github.com/dogmatiq/veracity/persistence/kv"
)

// KeyValueStore is an implementation of kv.Store that stores keyspaces in
// a PostgreSQL database.
type KeyValueStore struct {
	DB *sql.DB
}

// Open returns the keyspace at the given path.
//
// The path uniquely identifies the keyspace. It must not be empty. Each
// element must be a non-empty UTF-8 string consisting solely of printable
// Unicode characters, excluding whitespace. A printable character is any
// character from the Letter, Mark, Number, Punctuation or Symbol
// categories.
func (s *KeyValueStore) Open(ctx context.Context, path ...string) (kv.Keyspace, error) {
	return &keyspace{
		Path: pathkey.New(path),
		DB:   s.DB,
	}, nil
}

type keyspace struct {
	Path string
	DB   *sql.DB
}

func (ks *keyspace) Get(ctx context.Context, k []byte) (v []byte, err error) {
	row := ks.DB.QueryRowContext(
		ctx,
		`SELECT
			value
		FROM veracity.keyspace
		WHERE path = $1
		AND key = $2`,
		ks.Path,
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
		FROM veracity.keyspace
		WHERE path = $1
		AND key = $2`,
		ks.Path,
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
			`DELETE FROM veracity.keyspace
			WHERE path = $1
			AND key = $2`,
			ks.Path,
			k,
		)

		return err
	}

	_, err := ks.DB.ExecContext(
		ctx,
		`INSERT INTO veracity.keyspace AS o (
			path,
			key,
			value
		) VALUES (
			$1, $2, $3
		) ON CONFLICT (path, key) DO UPDATE SET
			value = $3
		`,
		ks.Path,
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
		FROM veracity.keyspace
		WHERE path = $1`,
		ks.Path,
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
		`CREATE TABLE IF NOT EXISTS veracity.keyspace (
			path  TEXT NOT NULL,
			key   BYTEA NOT NULL,
			value BYTEA NOT NULL,

			PRIMARY KEY (path, key)
		)`,
	); err != nil {
		return err
	}

	return nil
}
