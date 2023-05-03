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

func TestTypes(t *testing.T) {
	tests := []*struct {
		name    string
		tables  []queries.Tables
		columns []queries.Column
		tc      []queries.TableConstraint
		ccols   []queries.ConstraintColumn
		fk      []queries.ForeignKey
		types   []queries.Type
		enums   []queries.Enum
	}{
		{
			name:    "simple",
			tables:  []queries.Tables{{Table: "table"}},
			columns: []queries.Column{{TableName: "table", ColumnName: "col", TypeName: "type"}},
			types:   []queries.Type{{TypeName: "type", TypeType: "b"}},
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
			q.EXPECT().TableConstraints(mock.Anything, mock.Anything, mock.Anything).Return(tt.tc, nil)
			q.EXPECT().ConstraintColumns(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tt.ccols, nil)
			q.EXPECT().ForeignKeys(mock.Anything, mock.Anything, mock.Anything).Return(tt.fk, nil)
			q.EXPECT().Types(mock.Anything, mock.Anything, mock.Anything).Return(tt.types, nil)
			q.EXPECT().Enums(mock.Anything, mock.Anything, mock.Anything).Return(tt.enums, nil)

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
