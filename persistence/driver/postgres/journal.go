package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dogmatiq/veracity/persistence/journal"
)

// JournalStore is an implementation of [journal.Store] that contains journals
// that persist records in a PostgresSQL table.
type JournalStore struct {
	// DB is the PostgreSQL database connection.
	DB *sql.DB
}

// Open returns the journal with the given name.
func (s *JournalStore) Open(ctx context.Context, name string) (journal.Journal, error) {
	return &journ{
		Name: name,
		DB:   s.DB,
	}, nil
}

// journ is an implementation of journal.Journal that stores records in
// a DynamoDB table.
type journ struct {
	Name string
	DB   *sql.DB
}

func (j *journ) Get(ctx context.Context, offset uint64) ([]byte, bool, error) {
	row := j.DB.QueryRowContext(
		ctx,
		`SELECT record
		FROM veracity.journal
		WHERE name = $1
		AND "offset" = $2`,
		j.Name,
		offset,
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
	begin uint64,
	fn journal.RangeFunc,
) error {
	// TODO: use a limit and offset to "page" through records
	rows, err := j.DB.QueryContext(
		ctx,
		`SELECT "offset", record
		FROM veracity.journal
		WHERE name = $1
		AND "offset" >= $2
		ORDER BY "offset"`,
		j.Name,
		begin,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	expectedOffset := begin

	for rows.Next() {
		var (
			offset uint64
			rec    []byte
		)
		if err = rows.Scan(&offset, &rec); err != nil {
			return err
		}
		if offset != expectedOffset {
			return errors.New("cannot range over truncated records")
		}
		expectedOffset++

		ok, err := fn(ctx, offset, rec)
		if !ok || err != nil {
			return err
		}
	}

	return rows.Err()
}

func (j *journ) RangeAll(
	ctx context.Context,
	fn journal.RangeFunc,
) error {
	// TODO: use a limit and offset to "page" through records
	rows, err := j.DB.QueryContext(
		ctx,
		`SELECT "offset", record
		FROM veracity.journal
		WHERE name = $1
		ORDER BY "offset"`,
		j.Name,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var (
		firstIteration = true
		expectedOffset uint64
	)
	for rows.Next() {
		var (
			offset uint64
			rec    []byte
		)
		if err = rows.Scan(&offset, &rec); err != nil {
			return err
		}

		if firstIteration {
			expectedOffset = offset
			firstIteration = false
		} else if offset != expectedOffset {
			return errors.New("cannot range over truncated records")
		}

		expectedOffset++

		ok, err := fn(ctx, offset, rec)
		if !ok || err != nil {
			return err
		}
	}

	return rows.Err()
}

func (j *journ) Append(ctx context.Context, offset uint64, rec []byte) (bool, error) {
	res, err := j.DB.ExecContext(
		ctx,
		`INSERT INTO veracity.journal
		(name, "offset", record) VALUES ($1, $2, $3)
		ON CONFLICT (name, "offset") DO NOTHING`,
		j.Name,
		offset,
		rec,
	)
	if err != nil {
		return false, err
	}

	ra, err := res.RowsAffected()
	return ra == 1, err
}

func (j *journ) Truncate(ctx context.Context, end uint64) error {
	_, err := j.DB.ExecContext(
		ctx,
		`DELETE FROM veracity.journal
		WHERE name = $1
		AND "offset" < $2`,
		j.Name,
		end,
	)

	return err
}

func (j *journ) Close() error {
	return nil
}

// CreateJournalStoreSchema creates the PostgreSQL schema elements required by
// [JournalStore].
func CreateJournalStoreSchema(
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
			name     TEXT NOT NULL,
			"offset" BIGINT NOT NULL,
			record   BYTEA NOT NULL,

			PRIMARY KEY (name, "offset")
		)`,
	); err != nil {
		return err
	}

	return nil
}
