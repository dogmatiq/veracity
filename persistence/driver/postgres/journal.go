package postgres

import (
	"context"
	"database/sql"

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

func (j *journ) Bounds(ctx context.Context) (begin, end journal.Position, err error) {
	row := j.DB.QueryRowContext(
		ctx,
		`SELECT
			COALESCE(MIN(position),     0),
			COALESCE(MAX(position) + 1, 0)
		FROM veracity.journal
		WHERE name = $1`,
		j.Name,
	)

	err = row.Scan(&begin, &end)
	return begin, end, err
}

func (j *journ) Get(ctx context.Context, pos journal.Position) ([]byte, error) {
	row := j.DB.QueryRowContext(
		ctx,
		`SELECT record
		FROM veracity.journal
		WHERE name = $1
		AND position = $2`,
		j.Name,
		pos,
	)

	var rec []byte
	err := row.Scan(&rec)
	if err == sql.ErrNoRows {
		return nil, journal.ErrNotFound
	}

	return rec, err
}

func (j *journ) Range(
	ctx context.Context,
	begin journal.Position,
	fn journal.RangeFunc,
) error {
	expect := begin

	return j.rangeQuery(
		ctx,
		func(ctx context.Context, pos journal.Position, rec []byte) (bool, error) {
			if pos != expect {
				return false, journal.ErrNotFound
			}

			expect++

			return fn(ctx, pos, rec)
		},
		`SELECT position, record
		FROM veracity.journal
		WHERE name = $1
		AND position >= $2
		ORDER BY position`,
		j.Name,
		begin,
	)
}

func (j *journ) RangeAll(
	ctx context.Context,
	fn journal.RangeFunc,
) error {
	var (
		first  = true
		expect journal.Position
	)

	return j.rangeQuery(
		ctx,
		func(ctx context.Context, pos journal.Position, rec []byte) (bool, error) {
			if first {
				expect = pos
				first = false
			}

			if pos != expect {
				return false, journal.ErrNotFound
			}

			expect++

			return fn(ctx, pos, rec)
		},
		`SELECT position, record
		FROM veracity.journal
		WHERE name = $1
		ORDER BY position`,
		j.Name,
	)
}

func (j *journ) rangeQuery(
	ctx context.Context,
	fn journal.RangeFunc,
	query string,
	args ...any,
) error {
	// TODO: "paginate" results across multiple queries to avoid loading
	// everything into memory at once.
	rows, err := j.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			pos journal.Position
			rec []byte
		)

		if err := rows.Scan(&pos, &rec); err != nil {
			return err
		}

		ok, err := fn(ctx, pos, rec)
		if !ok || err != nil {
			return err
		}
	}

	return rows.Err()
}

func (j *journ) Append(ctx context.Context, end journal.Position, rec []byte) error {
	res, err := j.DB.ExecContext(
		ctx,
		`INSERT INTO veracity.journal
		(name, position, record) VALUES ($1, $2, $3)
		ON CONFLICT (name, position) DO NOTHING`,
		j.Name,
		end,
		rec,
	)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n != 1 {
		return journal.ErrConflict
	}

	return nil
}

func (j *journ) Truncate(ctx context.Context, end journal.Position) error {
	_, err := j.DB.ExecContext(
		ctx,
		`DELETE FROM veracity.journal
		WHERE name = $1
		AND position < $2`,
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
			position BIGINT NOT NULL,
			record   BYTEA NOT NULL,

			PRIMARY KEY (name, position)
		)`,
	); err != nil {
		return err
	}

	return nil
}
