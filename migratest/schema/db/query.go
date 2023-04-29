package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Executor interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
}

type Error struct {
	Err     error
	Message string

	Query string
	Args  []any
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e Error) Pretty() string {
	return fmt.Sprintf("%s: %v:\nquery:\n%s\n\n===\nargs: %+#v", e.Message, e.Err, e.Query, e.Args)
}

type Querier[T any] struct {
	query string
	args  []any
}

func NewQuery[T any](query string, args ...any) *Querier[T] {
	return &Querier[T]{
		query: query,
		args:  args,
	}
}

func (q Querier[T]) Error(msg string, err error) error {
	return Error{
		Message: msg,
		Err:     err,
		Query:   q.query,
		Args:    q.args,
	}
}

func (q Querier[T]) All(
	ctx context.Context,
	exec Executor,
	scan func(pgx.Rows, *T) error,
) (results []T, err error) {
	rows, err := exec.Query(ctx, q.query, q.args...)
	if err != nil {
		return nil, q.Error("exec", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row T
		if err := scan(rows, &row); err != nil {
			return nil, q.Error("scan", err)
		}
		results = append(results, row)
	}
	return results, nil
}

func (q Querier[T]) AllRet(
	ctx context.Context,
	exec Executor,
	scan func(pgx.Rows) (T, error),
) (results []T, err error) {
	rows, err := exec.Query(ctx, q.query, q.args...)
	if err != nil {
		return nil, q.Error("exec", err)
	}
	defer rows.Close()

	for rows.Next() {
		if row, err := scan(rows); err != nil {
			return nil, q.Error("scan", err)
		} else {
			results = append(results, row)
		}
	}
	return results, nil
}
