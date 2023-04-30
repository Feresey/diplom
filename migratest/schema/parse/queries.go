package parse

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v5"
	"github.com/volatiletech/null/v8"
)

//go:embed queries/tables.sql
var queryTablesSQL string

type queryTables struct {
	Schema string
	Table  string
}

func QueryTables(ctx context.Context, exec Executor, schemas []string) ([]queryTables, error) {
	return QueryAll(
		ctx, exec, queryTablesSQL,
		func(scan pgx.Rows, v *queryTables) error {
			return scan.Scan(
				&v.Schema,
				&v.Table,
			)
		}, schemas)
}

//go:embed queries/tables_columns.sql
var queryTablesColumnsSQL string

type queryTablesColumns struct {
	TableSchema         string
	TableName           string
	ColumnName          string
	TypeSchema          string
	TypeName            string
	TypeType            string
	IsNullable          bool
	HasDefault          bool
	IsGenerated         bool
	DefaultExpr         null.String
	ArrayDims           int
	TypeMaxLength       null.Int
	CompositeSchema     null.String
	CompositeType       null.String
	DomainIsNotNullable bool
	DomainType          null.String
	DomainTypeMaxLength null.Int
	DomainArrayDims     int
}

func QueryTablesColumns(
	ctx context.Context,
	exec Executor,
	tableNames []string,
) ([]queryTablesColumns, error) {
	return QueryAll(
		ctx, exec, queryTablesColumnsSQL,
		func(s pgx.Rows, q *queryTablesColumns) error {
			return s.Scan(
				&q.TableSchema,
				&q.TableName,
				&q.ColumnName,
				&q.TypeSchema,
				&q.TypeName,
				&q.TypeType,
				&q.IsNullable,
				&q.HasDefault,
				&q.IsGenerated,
				&q.DefaultExpr,
				&q.ArrayDims,
				&q.TypeMaxLength,
				&q.CompositeSchema,
				&q.CompositeType,
				&q.DomainIsNotNullable,
				&q.DomainType,
				&q.DomainTypeMaxLength,
				&q.DomainArrayDims,
			)
		}, tableNames)
}

//go:embed queries/tables_constraints.sql
var queryTablesConstraintsSQL string

type queryTablesConstraints struct {
	TableSchema      string
	TableName        string
	ConstraintSchema string
	ConstraintName   string
	ConstraintType   string
	NullsNotDistinct bool
	ConstraintDef    string
}

func QueryTablesConstraints(
	ctx context.Context,
	exec Executor,
	tableNames []string,
) ([]queryTablesConstraints, error) {
	return QueryAll(
		ctx, exec, queryTablesConstraintsSQL,
		func(s pgx.Rows, q *queryTablesConstraints) error {
			return s.Scan(
				&q.TableSchema,
				&q.TableName,
				&q.ConstraintSchema,
				&q.ConstraintName,
				&q.ConstraintType,
				&q.NullsNotDistinct,
				&q.ConstraintDef,
			)
		}, tableNames)
}

//go:embed queries/foreign_constraints.sql
var queryForeignKeysSQL string

type queryForeignKeys struct {
	ConstraintSchema       string
	ConstraintName         string
	UniqueConstraintSchema string
	UniqueConstraintName   string
	MatchOption            string
	UpdateRule             string
	DeleteRule             string
}

func QueryForeignKeys(
	ctx context.Context,
	exec Executor,
	constraintNames []string,
) ([]queryForeignKeys, error) {
	return QueryAll(
		ctx, exec, queryForeignKeysSQL,
		func(s pgx.Rows, q *queryForeignKeys) error {
			return s.Scan(
				&q.ConstraintSchema,
				&q.ConstraintName,
				&q.UniqueConstraintSchema,
				&q.UniqueConstraintName,
				&q.MatchOption,
				&q.UpdateRule,
				&q.DeleteRule,
			)
		}, constraintNames)
}

//go:embed queries/constraints_columns.sql
var queryConstraintsColumnsSQL string

type queryConstraintsColumns struct {
	SchemaName     string
	ConstraintName string
	TableName      string
	ColumnName     string
}

func QueryConstraintsColumns(
	ctx context.Context,
	exec Executor,
	tableNames []string,
	constraintNames []string,
) ([]queryConstraintsColumns, error) {
	return QueryAll(
		ctx, exec, queryConstraintsColumnsSQL,
		func(s pgx.Rows, q *queryConstraintsColumns) error {
			return s.Scan(
				&q.SchemaName,
				&q.ConstraintName,
				&q.TableName,
				&q.ColumnName,
			)
		}, tableNames, constraintNames)
}
