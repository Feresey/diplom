package query

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Queries struct{}

type Table struct {
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

func (Queries) Tables(ctx context.Context, exec Executor, p []TablesPattern) ([]Table, error) {
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
		func(scan pgx.Rows, v *Table) error {
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
	TableOID           int
	ColumnNum          int
	ColumnName         string
	TypeOID            int
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

func (Queries) Columns(
	ctx context.Context, exec Executor,
	tableOIDs []int,
) ([]Column, error) {
	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Column) error {
			return scan.Scan(
				&v.TableOID,
				&v.ColumnNum,
				&v.ColumnName,
				&v.TypeOID,
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
		queryColumnsSQL, tableOIDs)
}

//go:embed sql/constraints.sql
var queryTableConstraintsSQL string

type Constraint struct {
	ConstraintOID    int
	ConstraintName   string
	SchemaName       string
	ConstraintType   string
	NullsNotDistinct bool
	ConstraintDef    string
	TableOID         int
	Colnums          []int
	ForeignTableOID  sql.NullInt32
	ForeignColnums   []int
}

func (Queries) Constraints(
	ctx context.Context,
	exec Executor,
	tableOIDs []int,
) ([]Constraint, error) {
	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Constraint) error {
			return scan.Scan(
				&v.ConstraintOID,
				&v.ConstraintName,
				&v.SchemaName,
				&v.ConstraintType,
				&v.NullsNotDistinct,
				&v.ConstraintDef,
				&v.TableOID,
				&v.Colnums,
				&v.ForeignTableOID,
				&v.ForeignColnums,
			)
		},
		queryTableConstraintsSQL, tableOIDs)
}

//go:embed sql/types.sql
var querySelectTypesSQL string

type Type struct {
	TypeOID                int
	TypeSchema             string
	TypeName               string
	TypeType               string
	IsArray                bool
	ElemTypeOID            sql.NullInt32
	DomainTypeOID          sql.NullInt32
	DomainIsNotNullable    bool
	DomainCharacterMaxSize sql.NullInt32
	DomainIsNumeric        bool
	DomainNumericPrecision sql.NullInt32
	DomainNumericScale     sql.NullInt32
	DomainArrayDims        int
	RangeElementTypeOID    sql.NullInt32
}

func (Queries) Types(
	ctx context.Context, exec Executor,
	typeOIDs []int,
) ([]Type, error) {
	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Type) error {
			return scan.Scan(
				&v.TypeOID,
				&v.TypeSchema,
				&v.TypeName,
				&v.TypeType,
				&v.IsArray,
				&v.ElemTypeOID,
				&v.DomainTypeOID,
				&v.DomainIsNotNullable,
				&v.DomainCharacterMaxSize,
				&v.DomainIsNumeric,
				&v.DomainNumericPrecision,
				&v.DomainNumericScale,
				&v.DomainArrayDims,
				&v.RangeElementTypeOID,
			)
		},
		querySelectTypesSQL, typeOIDs)
}

//go:embed sql/indexes.sql
var queryIndexesSQL string

type Index struct {
	TableOID           int
	IndexOID           int
	IndexSchema        string
	IndexName          string
	ConstraintOID      sql.NullInt32
	IsUnique           bool
	IsPrimary          bool
	IsNullsNotDistinct bool
	Columns            []int
	IndexDefinition    string
}

func (Queries) Indexes(
	ctx context.Context,
	exec Executor,
	tableOIDs []int,
	constraintOIDs []int,
) ([]Index, error) {
	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Index) error {
			return scan.Scan(
				&v.TableOID,
				&v.IndexOID,
				&v.IndexSchema,
				&v.IndexName,
				&v.ConstraintOID,
				&v.IsUnique,
				&v.IsPrimary,
				&v.IsNullsNotDistinct,
				&v.Columns,
				&v.IndexDefinition,
			)
		},
		queryIndexesSQL, tableOIDs, constraintOIDs)
}

//go:embed sql/enums.sql
var queryEnumsSQL string

type Enum struct {
	TypeOID int
	Values  []string
}

func (Queries) Enums(
	ctx context.Context,
	exec Executor,
	enumTypeOIDs []int,
) ([]Enum, error) {
	return QueryAll(
		ctx, exec,
		func(scan pgx.Rows, v *Enum) error {
			return scan.Scan(
				&v.TypeOID,
				&v.Values,
			)
		},
		queryEnumsSQL, enumTypeOIDs)
}
