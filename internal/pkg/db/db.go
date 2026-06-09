package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type database struct {
	conn *pgx.Conn
}

type DB interface {
	Exec(ctx context.Context, sql string, args ...any) error
	Query(ctx context.Context, sql string, result Results, args ...any) error
	QueryRow(ctx context.Context, sql string, result Result, args ...any) error
}

func Connect(url string) (DB, error) {
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}

	return &database{
		conn: conn,
	}, nil
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
	_, err := d.conn.Exec(ctx, sql, args...)
	return err
}

func (d *database) Query(ctx context.Context, sql string, result Results, args ...any) error {
	queryArgs := make([]any, len(args))

	copy(queryArgs, args)

	rows, err := d.conn.Query(ctx, sql, queryArgs)
	if err != nil {
		return err
	}
	err = result(rows)
	if err != nil {
		return err
	}
	return nil
}

func (d *database) QueryRow(ctx context.Context, sql string, result Result, args ...any) error {
	queryArgs := make([]any, len(args))

	copy(queryArgs, args)

	rows, err := d.conn.Query(ctx, sql, queryArgs)
	if err != nil {
		return err
	}

	err = result(rows)
	if err != nil {
		return err
	}
	return nil
}
