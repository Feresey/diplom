package parse

import (
	"context"
	"testing"

	queries "github.com/Feresey/mtest/parse/queries"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestParse(t *testing.T) {
	tests := []*struct {
		name    string
		tables  []queries.Tables
		columns []queries.Column
		tc      []queries.Constraint
		types   []queries.Type
		indexes []queries.Index
	}{
		{
			name:    "simple",
			tables:  []queries.Tables{{Table: "table", OID: 1}},
			columns: []queries.Column{{ColumnNum: 1, ColumnName: "col", TableOID: 1, TypeOID: 2}},
			types:   []queries.Type{{TypeOID: 2, TypeName: "type", TypeType: "b"}},
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
			q.EXPECT().Tables(mock.Anything, mock.Anything, mock.Anything).Return(tt.tables, nil)
			q.EXPECT().Columns(mock.Anything, mock.Anything, mock.Anything).Return(tt.columns, nil)
			q.EXPECT().Constraints(mock.Anything, mock.Anything, mock.Anything).Return(tt.tc, nil)
			q.EXPECT().Types(mock.Anything, mock.Anything, mock.Anything).Return(tt.types, nil)
			q.EXPECT().Indexes(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(tt.indexes, nil)

			p := Parser{
				conn: nil,
				log:  log.Named(tt.name),
				q:    q,
			}
			schema, err := p.LoadSchema(context.Background(), Config{})
			_ = schema
			r.NoError(err)
		})
	}
}
