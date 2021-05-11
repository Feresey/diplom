package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTable(t *testing.T) {
	t.Parallel()

	tables := []Table{
		{Name: "one"},
	}

	tbl := GetTable(tables, "one")
	require.Equal(t, "one", tbl.Name)
}

func TestGetTableMissing(t *testing.T) {
	t.Parallel()

	tables := []Table{
		{Name: "one"},
	}

	require.Panics(t, func() { GetTable(tables, "missing") })
}

func TestGetColumn(t *testing.T) {
	t.Parallel()

	table := Table{
		Columns: []Column{
			{Name: "one"},
		},
	}

	c := table.GetColumn("one")
	require.Equal(t, "one", c.Name)
}

func TestGetColumnMissing(t *testing.T) {
	t.Parallel()

	table := Table{
		Columns: []Column{
			{Name: "one"},
		},
	}

	require.Panics(t, func() { table.GetColumn("missing") })
}
