package queries

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Queries struct{}

type Tables struct {
	OID    int
	Schema string
	Table  string
}

type TablesPattern struct {
	Schema string
	Tables string
}

type queryBuiler struct {
	queries []string
	argnum  int
	args    []any
}

func (q *queryBuiler) NextArgNum() int {
	q.argnum++
	return q.argnum
}

func (q *queryBuiler) Append(query string, args ...any) {
	q.queries = append(q.queries, query)
	q.args = append(q.args, args...)
}

func (Queries) Tables(ctx context.Context, exec Executor, p []TablesPattern) ([]Tables, error) {
	const queryTablesSQL = `-- list tables
SELECT
	c.oid::INT AS table_oid,
	ns.nspname AS schema_name,
	c.relname AS table_name
FROM
	pg_class c
	JOIN pg_namespace ns ON ns.oid = c.relnamespace
WHERE
	c.relkind = 'r'`

	var qb queryBuiler

	for _, pattern := range p {
		args := []any{pattern.Schema}
		paramIndex := qb.NextArgNum()
		schema := fmt.Sprintf("ns.nspname LIKE $%d", paramIndex)
		if pattern.Tables != "" {
			args = append(args, pattern.Tables)
			paramIndex := qb.NextArgNum()
			schema = fmt.Sprintf("%s AND c.relname LIKE $%d", schema, paramIndex)
		}

		qb.Append(schema, args...)
	}

	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Tables) error {
			return scan.Scan(
				&v.OID,
				&v.Schema,
				&v.Table,
			)
		},
		fmt.Sprintf(
			"%s AND (%s) ORDER BY c.oid ASC",
			queryTablesSQL,
			strings.Join(qb.queries, " OR "),
		),
		qb.args...)
}

//go:embed sql/columns.sql
var queryColumnsSQL string

type Column struct {
	SchemaName string
	TableName  string
	ColumnName string

	TypeSchema string
	TypeName   string

	IsNullable         bool
	HasDefault         bool
	ArrayDims          int
	IsGenerated        bool
	DefaultExpr        sql.NullString
	CharacterMaxLength sql.NullInt32
	IsNumeric          bool
	NumericPriecision  sql.NullInt32
	NumericScale       sql.NullInt32
}

func (Queries) Columns(ctx context.Context, exec Executor, tableNames []string) ([]Column, error) {
	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Column) error {
			return scan.Scan(
				&v.SchemaName,
				&v.TableName,
				&v.ColumnName,

				&v.TypeSchema,
				&v.TypeName,

				&v.IsNullable,
				&v.HasDefault,
				&v.ArrayDims,
				&v.IsGenerated,
				&v.DefaultExpr,
				&v.CharacterMaxLength,
				&v.IsNumeric,
				&v.NumericPriecision,
				&v.NumericScale,
			)
		},
		queryColumnsSQL, tableNames)
}

//go:embed sql/constraints.sql
var queryTableConstraintsSQL string

type Constraint struct {
	SchemaName string

	ConstraintOID  int
	ConstraintName string

	ConstraintType   string
	NullsNotDistinct bool
	ConstraintDef    string

	TableOID  int
	TableName string
	Columns   []string

	ForeignTableOID   sql.NullInt32
	ForeignSchemaName sql.NullString
	ForeignTableName  sql.NullString
	ForeignColumns    []sql.NullString
}

func (Queries) Constraints(
	ctx context.Context,
	exec Executor,
	tableNames []string,
) ([]Constraint, error) {
	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Constraint) error {
			return scan.Scan(
				&v.SchemaName,

				&v.ConstraintOID,
				&v.ConstraintName,

				&v.ConstraintType,
				&v.NullsNotDistinct,
				&v.ConstraintDef,

				&v.TableOID,
				&v.TableName,
				&v.Columns,

				&v.ForeignTableOID,
				&v.ForeignSchemaName,
				&v.ForeignTableName,
				&v.ForeignColumns,
			)
		},
		queryTableConstraintsSQL, tableNames)
}

//go:embed sql/types.sql
var querySelectTypesSQL string

type Type struct {
	SchemaName             string
	TypeName               string
	TypeType               string
	IsArray                bool
	ElemTypeSchema         sql.NullString
	ElemTypeName           sql.NullString
	DomainIsNotNullable    bool
	DomainSchema           sql.NullString
	DomainType             sql.NullString
	DomainCharacterMaxSize sql.NullInt32
	DomainIsNumeric        bool
	DomainNumericPrecision sql.NullInt32
	DomainNumericScale     sql.NullInt32
	DomainArrayDims        int
	RangeElementTypeSchema sql.NullString
	RangeElementTypeName   sql.NullString
	MultiRangeTypeSchema   sql.NullString
	MultiRangeTypeName     sql.NullString
	EnumValues             []string
}

func (Queries) Types(ctx context.Context, exec Executor, typeNames []string) ([]Type, error) {
	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Type) error {
			return scan.Scan(
				&v.SchemaName,
				&v.TypeName,
				&v.TypeType,

				&v.IsArray,
				&v.ElemTypeSchema,
				&v.ElemTypeName,

				&v.DomainIsNotNullable,
				&v.DomainSchema,
				&v.DomainType,
				&v.DomainCharacterMaxSize,
				&v.DomainIsNumeric,
				&v.DomainNumericPrecision,
				&v.DomainNumericScale,
				&v.DomainArrayDims,

				&v.RangeElementTypeSchema,
				&v.RangeElementTypeName,
				&v.MultiRangeTypeSchema,
				&v.MultiRangeTypeName,

				&v.EnumValues,
			)
		},
		querySelectTypesSQL, typeNames)
}

//go:embed sql/indexes.sql
var queryIndexesSQL string

type Index struct {
	TableOID    int
	TableSchema string
	TableName   string

	IndexOID    int
	IndexSchema string
	IndexName   string

	ConstraintOID    sql.NullInt32
	ConstraintSchema sql.NullString
	ConstraintName   sql.NullString

	IsUnique           bool
	IsPrimary          bool
	IsNullsNotDistinct bool
	Columns            []string
	IndexDefinition    string
}

func (Queries) Indexes(
	ctx context.Context,
	exec Executor,
	tableNames []string,
	constraints []string,
) ([]Index, error) {
	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Index) error {
			return scan.Scan(
				&v.TableOID,
				&v.TableSchema,
				&v.TableName,

				&v.IndexOID,
				&v.IndexSchema,
				&v.IndexName,

				&v.ConstraintOID,
				&v.ConstraintSchema,
				&v.ConstraintName,

				&v.IsUnique,
				&v.IsPrimary,
				&v.IsNullsNotDistinct,
				&v.Columns,
				&v.IndexDefinition,
			)
		},
		queryIndexesSQL, tableNames, constraints)
}
