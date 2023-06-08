package checks

import (
	"testing"

	"github.com/Feresey/mtest/schema"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestLuaChecks(t *testing.T) {
	l := lua.NewState()
	t.Cleanup(l.Close)
	RegisterModule(l)

	err := l.DoString(`checks = require("checks")`)
	require.NoError(t, err)
	tests := []*struct {
		name    string
		table   schema.Table
		want    map[string][]string
		wantErr bool
	}{
		{
			name: "simple",
			table: schema.Table{
				Name: schema.Identifier{
					OID:    1,
					Schema: "public",
					Name:   "table",
				},
				Columns: map[string]schema.Column{
					"col1": {
						ColNum: 1,
						Name:   "col1",
						Type: &schema.DBType{
							TypeName: schema.Identifier{
								OID:    11,
								Schema: "pg_catalog",
								Name:   "int4",
							},
							Type: schema.DataTypeBase,
						},
					},
				},
				PrimaryKey: &schema.Constraint{
					OID:  22,
					Name: "pk",
					Type: schema.ConstraintTypePK,
					Index: &schema.Index{
						OID:       33,
						Name:      "pk_index",
						Columns:   []string{"col1"},
						IsPrimary: true,
						IsUnique:  true,
					},
					Columns: []string{"col1"},
				},
				Constraints: map[string]*schema.Constraint{
					"pk": {
						OID:  22,
						Name: "pk",
						Type: schema.ConstraintTypePK,
						Index: &schema.Index{
							OID:       33,
							Name:      "pk_index",
							Columns:   []string{"col1"},
							IsPrimary: true,
							IsUnique:  true,
						},
						Columns: []string{"col1"},
					},
				},
				Indexes: map[string]schema.Index{
					"pk_index": {
						OID:       33,
						Name:      "pk_index",
						Columns:   []string{"col1"},
						IsPrimary: true,
						IsUnique:  true,
					},
				},
				ForeignKeys:  map[string]schema.ForeignKey{},
				ReferencedBy: map[string]*schema.Constraint{},
			},
			want: map[string][]string{
				"col1": {
					"0", "-1", "1",
					"-2147483648", "2147483647",
				},
			},
		},
	}

	checks, err := NewLuaChecks(l, l.GetGlobal("checks"))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)

			values, err := checks.GetTableChecks(tt.table)
			if err != nil {
				r.True(tt.wantErr, "get checks, unexpected error: %+v", err)
			}
			r.Equal(tt.want, values)
		})
	}
}
