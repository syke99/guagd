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
