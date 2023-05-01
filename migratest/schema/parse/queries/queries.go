package queries

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v5"
	"github.com/volatiletech/null/v8"
)

//go:embed sql/tables.sql
var queryTablesSQL string

type Tables struct {
	Schema string
	Table  string
}

func QueryTables(ctx context.Context, exec Executor, schemas []string) ([]Tables, error) {
	return QueryAll(
		ctx, exec, queryTablesSQL,
		func(scan pgx.Rows, v *Tables) error {
			return scan.Scan(
				&v.Schema,
				&v.Table,
			)
		}, schemas)
}

//go:embed sql/columns.sql
var queryColumnsSQL string

type Column struct {
	SchemaName         string
	TableName          string
	ColumnName         string
	TypeSchema         string
	TypeName           string
	TypeType           string
	IsNullable         bool
	HasDefault         bool
	ArrayDims          int
	IsGenerated        bool
	DefaultExpr        null.String
	CharacterMaxLength null.Int
}

func QueryColumns(ctx context.Context, exec Executor, tableNames []string) ([]Column, error) {
	return QueryAll(
		ctx, exec, queryColumnsSQL,
		func(scan pgx.Rows, v *Column) error {
			return scan.Scan(
				&v.SchemaName,
				&v.TableName,
				&v.ColumnName,
				&v.TypeSchema,
				&v.TypeName,
				&v.TypeType,
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

func QueryTableConstraints(ctx context.Context, exec Executor, tableNames []string) ([]TableConstraint, error) {
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

//go:embed sql/foreign_keys.sql
var queryForeignKeysSQL string

type ForeignKey struct {
	ConstraintSchema       string
	ConstraintName         string
	UniqueConstraintSchema string
	UniqueConstraintName   string
	MatchOption            string
	UpdateRule             string
	DeleteRule             string
}

func QueryForeignKeys(ctx context.Context, exec Executor, constraintNames []string) ([]ForeignKey, error) {
	return QueryAll(
		ctx, exec, queryForeignKeysSQL,
		func(scan pgx.Rows, v *ForeignKey) error {
			err := scan.Scan(
				&v.ConstraintSchema,
				&v.ConstraintName,
				&v.UniqueConstraintSchema,
				&v.UniqueConstraintName,
				&v.MatchOption,
				&v.UpdateRule,
				&v.DeleteRule,
			)
			if err != nil {
				return err
			}
			return nil
		}, constraintNames)
}

//go:embed sql/constraint_columns.sql
var queryConstraintColumnsSQL string

type ConstraintColumn struct {
	SchemaName     string
	ConstraintName string
	TableName      string
	ColumnName     string
}

func QueryConstraintColumns(ctx context.Context, exec Executor, tableNames []string, constraintNames []string) ([]ConstraintColumn, error) {
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
	ElemTypeSchema         null.String
	ElemTypeName           null.String
	DomainIsNotNullable    bool
	DomainSchema           null.String
	DomainType             null.String
	DomainCharacterMaxSize null.Int
	DomainArrayDims        int
	RangeElementTypeSchema null.String
	RangeElementTypeName   null.String
	MultiRangeTypeSchema   null.String
	MultiRangeTypeName     null.String
}

func QueryTypes(ctx context.Context, exec Executor, typeNames []string) ([]Type, error) {
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

func QueryArrayTypes(ctx context.Context, exec Executor, arrayNames []string) ([]ArrayType, error) {
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

func QueryEnums(ctx context.Context, exec Executor, enums []string) ([]Enum, error) {
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
