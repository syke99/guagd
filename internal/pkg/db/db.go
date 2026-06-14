package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type database struct {
	pool *pgxpool.Pool
}

type DB interface {
	Exec(ctx context.Context, sql string, args ...any) error
	Query(ctx context.Context, sql string, result Results, args ...any) error
	QueryRow(ctx context.Context, sql string, result Result, args ...any) error
	BeginTx(ctx context.Context) (Tx, error)
}

type Tx interface {
	Exec(ctx context.Context, sql string, args ...any) error
	Query(ctx context.Context, sql string, result Results, args ...any) error
	QueryRow(ctx context.Context, sql string, result Result, args ...any) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

func Connect(url string) (DB, error) {
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		return nil, err
	}

	return &database{pool: pool}, nil
}

type Results func(pgx.Rows) error
type Result func(pgx.Rows) error

func WithResultsOf[T any](dest *[]T) Results {
	return func(rows pgx.Rows) error {
		results, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
		if err != nil {
			return err
		}
		*dest = results
		return nil
	}
}

func WithResultOf[T any](dest *T) Result {
	return func(rows pgx.Rows) error {
		result, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
		if err != nil {
			return err
		}
		*dest = result
		return nil
	}
}

func (d *database) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := d.pool.Exec(ctx, sql, args...)
	return err
}

func (d *database) Query(ctx context.Context, sql string, result Results, args ...any) error {
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return result(rows)
}

func (d *database) QueryRow(ctx context.Context, sql string, result Result, args ...any) error {
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return result(rows)
}

func (d *database) BeginTx(ctx context.Context) (Tx, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &dbTx{tx: tx}, nil
}

type dbTx struct {
	tx pgx.Tx
}

func (t *dbTx) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := t.tx.Exec(ctx, sql, args...)
	return err
}

func (t *dbTx) Query(ctx context.Context, sql string, result Results, args ...any) error {
	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return result(rows)
}

func (t *dbTx) QueryRow(ctx context.Context, sql string, result Result, args ...any) error {
	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return result(rows)
}

func (t *dbTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *dbTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}
