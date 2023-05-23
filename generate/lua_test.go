package generate

import (
	_ "embed"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Feresey/mtest/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

//go:embed testdata/dump.json
var testSchemaRaw []byte

func initTestSchema(t *testing.T) *schema.Schema {
	t.Helper()
	var res schema.Schema
	require.NoError(t, json.Unmarshal(testSchemaRaw, &res))
	return &res
}

func TestNewLuaDomain(t *testing.T) {
	source := strings.NewReader(`
		function init(table, col)
		end

		function reset()
		end

		function next()
			return true
		end

		function value()
			return "test"
		end
	`)

	log := zap.NewExample()
	name := "test-lua-domain"

	s := initTestSchema(t)

	ld, err := NewLuaDomain(log, s, source, name)
	require.NoError(t, err)

	err = ld.Init("table", "col")
	require.NoError(t, err)

	ld.Reset()

	next := ld.Next()
	assert.True(t, next)

	value := ld.Value()
	assert.Equal(t, "test", value)
}

func TestIntIteration(t *testing.T) {
	source := strings.NewReader(`
		local i
		local top

		function init(table, col)
			i = 0
			top = schema.tables[table].columns[col].col_num
		end

		function reset()
			i = 0
		end

		function next()
			i = i + 1
			return i < top
		end

		function value()
			return tostring(i)
		end
	`)

	log := zap.NewExample()
	name := "int-lua-domain"

	s := initTestSchema(t)

	ld, err := NewLuaDomain(log, s, source, name)
	require.NoError(t, err)

	err = ld.Init("table", "col")
	require.NoError(t, err)

	var values []string
	for ld.Next() {
		value := ld.Value()
		values = append(values, value)
	}

	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, values)
}
