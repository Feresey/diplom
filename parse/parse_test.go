package parse

import (
	"context"
	"testing"

	"github.com/Feresey/mtest/parse/query"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestParse(t *testing.T) {
	tests := []*struct {
		name    string
		tables  []query.Table
		columns []query.Column
		tc      []query.Constraint
		types   []query.Type
		indexes []query.Index
	}{
		{
			name:    "simple",
			tables:  []query.Table{{Table: "table", OID: 1}},
			columns: []query.Column{{ColumnNum: 1, ColumnName: "col", TableOID: 1, TypeOID: 2}},
			types:   []query.Type{{TypeOID: 2, TypeName: "type", TypeType: "b"}},
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
