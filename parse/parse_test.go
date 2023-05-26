package parse

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Feresey/mtest/parse/query"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestParse(t *testing.T) {
	type colQuery struct {
		tables  []int
		columns []query.Column
	}
	type conQuery struct {
		tables      []int
		constraints []query.Constraint
	}
	type indQuery struct {
		tables      []int
		constraints []int
		indexes     []query.Index
	}
	type typeQuery struct {
		types    []int
		typesRet []query.Type
	}
	type enumsQuery struct {
		enums    []int
		enumsRet []query.Enum
	}
	tests := []*struct {
		name        string
		tables      []query.Table
		columns     colQuery
		constraints conQuery
		indexes     indQuery
		types       []typeQuery
		enums       enumsQuery
	}{
		{
			name: "simple",
			tables: []query.Table{
				{Table: "table1", OID: 1},
				{Table: "table2", OID: 2},
			},
			columns: colQuery{
				tables: []int{1, 2},
				columns: []query.Column{
					{TableOID: 1, ColumnNum: 1, ColumnName: "col1", TypeOID: 12},
					{TableOID: 1, ColumnNum: 2, ColumnName: "col2", TypeOID: 13},
					{TableOID: 1, ColumnNum: 3, ColumnName: "col3", TypeOID: 14, CharacterMaxLength: sql.NullInt32{Int32: 255, Valid: true}},
					{TableOID: 1, ColumnNum: 4, ColumnName: "col4", TypeOID: 15},

					{TableOID: 2, ColumnNum: 1, ColumnName: "col1", TypeOID: 12},
				},
			},
			constraints: conQuery{
				tables: []int{1, 2},
				constraints: []query.Constraint{
					{ConstraintOID: 22, TableOID: 1, Colnums: []int{1}, ConstraintName: "pk", ConstraintType: "p"},
					{ConstraintOID: 23, TableOID: 2, Colnums: []int{1}, ConstraintName: "uniq", ConstraintType: "u"},
					{
						ConstraintOID: 23, TableOID: 1, Colnums: []int{1}, ConstraintName: "fk", ConstraintType: "f",
						ForeignTableOID: sql.NullInt32{Int32: 2, Valid: true},
						ForeignColnums:  []int{1},
					},
				},
			},
			types: []typeQuery{
				{
					types: []int{12, 13, 14, 15},
					typesRet: []query.Type{
						{TypeOID: 12, TypeName: "base_type", TypeType: "b"},
						{TypeOID: 13, TypeName: "custom_enum", TypeType: "e"},
						{
							TypeOID: 14, TypeName: "custom_domain1", TypeType: "d",
							DomainTypeOID: sql.NullInt32{Int32: 16, Valid: true},
						},
						{
							TypeOID: 15, TypeName: "custom_domain2", TypeType: "d",
							DomainTypeOID: sql.NullInt32{Int32: 12, Valid: true},
						},
					},
				},
				{
					types: []int{16},
					typesRet: []query.Type{
						{TypeOID: 16, TypeName: "custom_domain_elem", TypeType: "b"},
					},
				},
			},
			indexes: indQuery{
				tables:      []int{1, 2},
				constraints: []int{22, 23},
				indexes: []query.Index{
					{
						TableOID: 1, IndexOID: 4, IndexName: "idx",
						ConstraintOID: sql.NullInt32{Int32: int32(23), Valid: true},
						Columns:       []int{1},
						IsUnique:      true,
						IsPrimary:     true,
					},
				},
			},
			enums: enumsQuery{
				enums:    []int{13},
				enumsRet: []query.Enum{{TypeOID: 13, Values: []string{"val1", "val2"}}},
			},
		},
	}

	lc := zap.NewDevelopmentConfig()
	lc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	log, err := lc.Build(zap.AddStacktrace(zap.WarnLevel))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			q := NewMockQueries(t)
			anyCtx := mock.Anything
			anyExec := mock.Anything
			expect := q.EXPECT()
			expect.Tables(anyCtx, anyExec, mock.Anything).Return(tt.tables, nil)
			expect.Columns(anyCtx, anyExec, tt.columns.tables).Return(tt.columns.columns, nil)
			expect.Constraints(anyCtx, anyExec, tt.constraints.tables).Return(tt.constraints.constraints, nil)
			expect.Indexes(anyCtx, anyExec, tt.indexes.tables, tt.indexes.constraints).Return(tt.indexes.indexes, nil)
			for _, typQ := range tt.types {
				q.EXPECT().Types(anyCtx, anyExec, typQ.types).Return(typQ.typesRet, nil)
			}
			q.EXPECT().Enums(anyCtx, anyExec, tt.enums.enums).Return(tt.enums.enumsRet, nil)

			p := NewParser(nil, log.Named(tt.name))
			p.q = q

			schema, err := p.LoadSchema(context.Background(), Config{})
			_ = schema
			r.NoError(err)
		})
	}
}
