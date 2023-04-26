package schema

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

type SQLizer interface {
	ToSql() (query string, args []any, err error)
}

type Error struct {
	Err   error
	State string

	Query string
	Args  []any
}

func (e Error) Error() string {
	if e.Query == "" {
		return fmt.Sprintf("%s: %v", e.State, e.Err)
	}
	return fmt.Sprintf("%s: %v: query: `%s` args: %+#v", e.State, e.Err, e.Query, e.Args)
}

type Scanner interface {
	Scan(dest ...any) error
}

func QueryOne[T any](
	ctx context.Context,
	exec Executor,
	sb SQLizer,
	scan func(Scanner, *T) error,
) (result T, err error) {
	query, args, err := sb.ToSql()
	if err != nil {
		return result, Error{
			Err:   err,
			State: "build query",
		}
	}

	row := exec.QueryRow(ctx, query, args...)
	if err := scan(row, &result); err != nil {
		return result, Error{
			Err:   err,
			State: "scan results",
			Query: query,
			Args:  args,
		}
	}
	return result, nil
}

func QueryAll[T any](
	ctx context.Context,
	exec Executor,
	sb SQLizer,
	scan func(Scanner, *T) error,
) (results []T, err error) {
	query, args, err := sb.ToSql()
	if err != nil {
		return nil, Error{
			Err:   err,
			State: "build query",
		}
	}

	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, Error{
			Err:   err,
			State: "exec query",
			Query: query,
			Args:  args,
		}
	}
	defer rows.Close()

	for rows.Next() {
		var row T
		if err := scan(rows, &row); err != nil {
			return nil, Error{
				Err:   err,
				State: "scan results",
				Query: query,
				Args:  args,
			}
		}
	}
	return results, nil
}
