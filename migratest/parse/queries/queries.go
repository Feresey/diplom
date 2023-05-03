package queries

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Queries struct{}

//go:embed sql/tables.sql
var queryTablesSQL string

type Tables struct {
	OID    int
	Schema string
	Table  string
}

type TablesPattern struct {
	Schema string
	Tables string
}

func (Queries) Tables(ctx context.Context, exec Executor, p []TablesPattern) ([]Tables, error) {
	var where string
	var paramIndex int
	var args []any
	for _, pattern := range p {
		paramIndex++
		args = append(args, pattern.Schema)
		schema := fmt.Sprintf("ns.nspname LIKE $%d", paramIndex)
		if pattern.Tables != "" {
			paramIndex++
			args = append(args, pattern.Tables)
			schema = fmt.Sprintf("%s AND c.relname LIKE $%d", schema, paramIndex)
		}

		if where != "" {
			where += " OR "
		}
		where += schema
	}

	return QueryAll(
		ctx, exec, fmt.Sprintf("%s AND (%s)", queryTablesSQL, where),
		func(scan pgx.Rows, v *Tables) error {
			return scan.Scan(
				&v.OID,
				&v.Schema,
				&v.Table,
			)
		}, args...)
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
}

func (Queries) Columns(ctx context.Context, exec Executor, tableNames []string) ([]Column, error) {
	return QueryAll(
		ctx, exec, queryColumnsSQL,
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
			)
		}, tableNames)
}

//go:embed sql/table_constraints.sql
var queryTableConstraintsSQL string

type TableConstraint struct {
	TableSchema      string
	TableName        string
	ConstraintSchema string
	ConstraintName   string
	ConstraintType   string
	NullsNotDistinct bool
	ConstraintDef    string
}

func (Queries) TableConstraints(ctx context.Context, exec Executor, tableNames []string) ([]TableConstraint, error) {
	return QueryAll(
		ctx, exec, queryTableConstraintsSQL,
		func(scan pgx.Rows, v *TableConstraint) error {
			return scan.Scan(
				&v.TableSchema,
				&v.TableName,
				&v.ConstraintSchema,
				&v.ConstraintName,
				&v.ConstraintType,
				&v.NullsNotDistinct,
				&v.ConstraintDef,
			)
		}, tableNames)
}

//go:embed sql/constraint_columns.sql
var queryConstraintColumnsSQL string

type ConstraintIdentifier struct {
	OID            int
	SchemaName     string
	ConstraintName string
	TableName      string
}

type ConstraintColumn struct {
	ColumnName     string
}

func (Queries) ConstraintColumns(
	ctx context.Context, exec Executor,
	tableNames []string, constraintNames []string,
) ([]ConstraintColumn, error) {
	return QueryAll(
		ctx, exec, queryConstraintColumnsSQL,
		func(scan pgx.Rows, v *ConstraintColumn) error {
			return scan.Scan(
				&v.SchemaName,
				&v.ConstraintName,
				&v.TableName,
				&v.ColumnName,
			)
		}, tableNames, constraintNames)
}

//go:embed sql/types/types.sql
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
	DomainArrayDims        int
	RangeElementTypeSchema sql.NullString
	RangeElementTypeName   sql.NullString
	MultiRangeTypeSchema   sql.NullString
	MultiRangeTypeName     sql.NullString
}

func (Queries) Types(ctx context.Context, exec Executor, typeNames []string) ([]Type, error) {
	return QueryAll(
		ctx, exec, querySelectTypesSQL,
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
				&v.DomainArrayDims,

				&v.RangeElementTypeSchema,
				&v.RangeElementTypeName,
				&v.MultiRangeTypeSchema,
				&v.MultiRangeTypeName,
			)
		}, typeNames)
}

//go:embed sql/types/array.sql
var queryArrayTypesSQL string

type ArrayType struct {
	SchemaName string
	ArrayName  string
	ElemName   string
}

func (Queries) ArrayTypes(ctx context.Context, exec Executor, arrayNames []string) ([]ArrayType, error) {
	return QueryAll(
		ctx, exec, queryArrayTypesSQL,
		func(scan pgx.Rows, v *ArrayType) error {
			return scan.Scan(
				&v.SchemaName,
				&v.ArrayName,
				&v.ElemName,
			)
		}, arrayNames)
}

//go:embed sql/types/enum.sql
var querySelectEnumsSQL string

type Enum struct {
	SchemaName string
	EnumName   string
	EnumValues []string
}

func (Queries) Enums(ctx context.Context, exec Executor, enums []string) ([]Enum, error) {
	return QueryAll(
		ctx, exec, querySelectEnumsSQL,
		func(scan pgx.Rows, v *Enum) error {
			return scan.Scan(
				&v.SchemaName,
				&v.EnumName,
				&v.EnumValues,
			)
		}, enums)
}
