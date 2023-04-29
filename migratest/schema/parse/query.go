package parse

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

func QueryAll[T any](
	ctx context.Context,
	exec Executor,
	query string,
	scan func(s pgx.Rows, q *T) error,
	args ...any,
) ([]T, error) {
	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, Error{
			Err:     err,
			Message: "query",
			Query:   query,
			Args:    args,
		}
	}
	defer rows.Close()

	var results []T

	var rowNum int
	for rows.Next() {
		rowNum++
		var value T
		if err := scan(rows, &value); err != nil {
			return nil, Error{
				Err:     err,
				Message: fmt.Sprintf("scan %d", rowNum),
				Query:   query,
				Args:    args,
			}
		}
		results = append(results, value)
	}
	return results, nil
}
