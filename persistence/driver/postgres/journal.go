package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dogmatiq/veracity/persistence/internal/pathkey"
	"github.com/dogmatiq/veracity/persistence/journal"
)

// JournalStore is an implementation of journal.Store that contains journals
// that persist records in a PostgresSQL table.
type JournalStore struct {
	// DB is the PostgreSQL database connection.
	DB *sql.DB
}

// Open returns the journal at the given path.
//
// The path uniquely identifies the journal. It must not be empty. Each element
// must be a non-empty UTF-8 string consisting solely of printable Unicode
// characters, excluding whitespace. A printable character is any character from
// the Letter, Mark, Number, Punctuation or Symbol categories.
func (s *JournalStore) Open(ctx context.Context, path ...string) (journal.Journal, error) {
	return &journ{
		Path: pathkey.New(path),
		DB:   s.DB,
	}, nil
}

// journ is an implementation of journal.Journal that stores records in
// a DynamoDB table.
type journ struct {
	Path string
	DB   *sql.DB
}

func (j *journ) Get(ctx context.Context, ver uint64) ([]byte, bool, error) {
	row := j.DB.QueryRowContext(
		ctx,
		`SELECT
			record
		FROM veracity.journal
		WHERE path = $1
		AND version = $2`,
		j.Path,
		ver,
	)

	var rec []byte
	err := row.Scan(&rec)
	if err == sql.ErrNoRows {
		err = nil
	}

	return rec, len(rec) > 0, err
}

func (j *journ) Range(
	ctx context.Context,
	ver uint64,
	fn func(context.Context, []byte) (bool, error),
) error {
	rows, err := j.DB.QueryContext(
		ctx,
		`SELECT
			version,
			record
		FROM veracity.journal
		WHERE path = $1
		AND version >= $2
		ORDER BY version`,
		j.Path,
		ver,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	expectedVersion := ver
	for rows.Next() {
		var (
			v   uint64
			rec []byte
		)
		if err = rows.Scan(&v, &rec); err != nil {
			return err
		}
		if v != expectedVersion {
			return errors.New("cannot range over truncated records")
		}
		expectedVersion++

		ok, err := fn(ctx, rec)
		if !ok || err != nil {
			return err
		}
	}

	return rows.Err()
}

func (j *journ) RangeAll(
	ctx context.Context,
	fn func(context.Context, uint64, []byte) (bool, error),
) error {
	rows, err := j.DB.QueryContext(
		ctx,
		`SELECT
		 	version,
			record
		FROM veracity.journal
		WHERE path = $1
		ORDER BY version`,
		j.Path,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var (
		firstIteration  = true
		expectedVersion uint64
	)
	for rows.Next() {
		var (
			v   uint64
			rec []byte
		)
		if err = rows.Scan(&v, &rec); err != nil {
			return err
		}

		if firstIteration {
			expectedVersion = v
			firstIteration = false
		} else if v != expectedVersion {
			return errors.New("cannot range over truncated records")
		}
		expectedVersion++

		ok, err := fn(ctx, v, rec)
		if !ok || err != nil {
			return err
		}
	}

	return rows.Err()
}

func (j *journ) Append(ctx context.Context, ver uint64, rec []byte) (bool, error) {
	res, err := j.DB.ExecContext(
		ctx,
		`INSERT INTO veracity.journal (
			path,
			version,
			record
		) VALUES (
			$1, $2, $3
		) ON CONFLICT (path, version) DO NOTHING`,
		j.Path,
		ver,
		rec,
	)
	if err != nil {
		return false, err
	}

	ra, err := res.RowsAffected()
	return ra == 1, err
}

func (j *journ) Truncate(ctx context.Context, ver uint64) error {
	_, err := j.DB.ExecContext(
		ctx,
		`DELETE FROM veracity.journal
		WHERE path = $1
		AND version < $2`,
		j.Path,
		ver,
	)

	return err
}

func (j *journ) Close() error {
	return nil
}

// CreateJournalSchema creates a PostgreSQL schema for storing journal records.
func CreateJournalSchema(
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
		`CREATE TABLE IF NOT EXISTS veracity.journal (
			path    TEXT NOT NULL,
			version BIGINT NOT NULL,
			record  BYTEA NOT NULL,

			PRIMARY KEY (path, version)
		)`,
	); err != nil {
		return err
	}

	return nil
}
