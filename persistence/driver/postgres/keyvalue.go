package postgres

import (
	"context"
	"database/sql"

	"github.com/dogmatiq/veracity/persistence/internal/pathkey"
	"github.com/dogmatiq/veracity/persistence/kv"
)

// KeyValueStore is an implementation of kv.Store that stores keyspaces in
// PostgreSQL DB.
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
	name := pathkey.New(path)

	return &keyspaceHandle{
		Name: name,
		DB:   s.DB,
	}, nil
}

type keyspaceHandle struct {
	Name string
	DB   *sql.DB
}

func (h *keyspaceHandle) Get(ctx context.Context, k []byte) (v []byte, err error) {
	row := h.DB.QueryRowContext(
		ctx,
		`SELECT
			value
		FROM veracity.kvstore
		WHERE keyspace = $1
		AND key = $2`,
		h.Name,
		k,
	)

	var value []byte
	err = row.Scan(&value)
	if err == sql.ErrNoRows {
		err = nil
	}

	return value, err
}

func (h *keyspaceHandle) Has(ctx context.Context, k []byte) (ok bool, err error) {
	row := h.DB.QueryRowContext(
		ctx,
		`SELECT
			1
		FROM veracity.kvstore
		WHERE keyspace = $1
		AND key = $2`,
		h.Name,
		k,
	)

	var value []byte
	err = row.Scan(&value)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return true, err
}

func (h *keyspaceHandle) Set(ctx context.Context, k, v []byte) error {
	if len(v) == 0 {
		_, err := h.DB.ExecContext(
			ctx,
			`DELETE FROM veracity.kvstore
			WHERE keyspace = $1
			AND key = $2`,
			h.Name,
			k,
		)

		return err
	}

	_, err := h.DB.ExecContext(
		ctx,
		`INSERT INTO veracity.kvstore AS o (
			keyspace,
			key,
			value
		) VALUES (
			$1, $2, $3
		) ON CONFLICT (keyspace, key) DO UPDATE SET
			value = $3
		`,
		h.Name,
		k,
		v,
	)

	return err
}

func (h *keyspaceHandle) RangeAll(
	ctx context.Context,
	fn func(ctx context.Context, k, v []byte) (bool, error),
) error {
	rows, err := h.DB.QueryContext(
		ctx,
		`SELECT
			key,
			value
		FROM veracity.kvstore
		WHERE keyspace = $1`,
		h.Name,
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

func (h *keyspaceHandle) Close() error {
	return nil
}
