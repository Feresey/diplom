package generate

import (
	"testing"

	"github.com/Feresey/mtest/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopologicalSort(t *testing.T) {
	tables := []struct {
		name     string
		graph    map[string]map[string]*schema.Table
		expected []string
	}{
		{
			"Single Table",
			map[string]map[string]*schema.Table{
				"table1": {},
			},
			[]string{"table1"},
		},
		{
			"No Tables",
			map[string]map[string]*schema.Table{},
			[]string{},
		},
		{
			"No Relationships",
			map[string]map[string]*schema.Table{
				"table1": {},
				"table2": {},
			},
			[]string{"table1", "table2"},
		},
		{
			"Simple Graph with One Cycle",
			map[string]map[string]*schema.Table{
				"table1": {
					"table2": nil,
					"table3": nil,
				},
				"table2": {
					"table1": nil, // This is the cycle
					"table4": nil,
				},
				"table3": {
					"table4": nil,
				},
				"table4": {},
			},
			nil,
		},
		{
			"Simple Graph with Two Cycles",
			map[string]map[string]*schema.Table{
				"table1": {
					"table2": nil,
					"table3": nil,
				},
				"table2": {
					"table4": nil,
				},
				"table3": {
					"table2": nil,
					"table4": nil,
				},
				"table4": {
					"table1": nil, // This is the first cycle
					"table5": nil,
				},
				"table5": {
					"table4": nil, // This is the second cycle
				},
			},
			nil,
		},
		{
			"Graph with All Tables Connected to One",
			map[string]map[string]*schema.Table{
				"table1": {
					"table2": nil,
					"table3": nil,
					"table4": nil,
				},
				"table2": {
					"table3": nil,
					"table4": nil,
				},
				"table3": {
					"table4": nil,
				},
				"table4": {},
			},
			[]string{"table1", "table2", "table3", "table4"},
		},
		{
			"Graph with Tables Connected in a Chain",
			map[string]map[string]*schema.Table{
				"table1": {
					"table2": nil,
				},
				"table2": {
					"table3": nil,
				},
				"table3": {
					"table4": nil,
				},
				"table4": {},
			},
			[]string{"table1", "table2", "table3", "table4"},
		},
		{
			"Graph with Multiple Independent Cycles",
			map[string]map[string]*schema.Table{
				"table1": {
					"table2": nil,
					"table3": nil,
				},
				"table2": {
					"table4": nil,
				},
				"table3": {
					"table5": nil,
				},
				"table4": {
					"table1": nil,
				},
				"table5": {
					"table3": nil,
				},
			},
			nil,
		},
		{
			"Graph with Self-Referencing Table",
			map[string]map[string]*schema.Table{
				"table1": {
					"table1": nil,
				},
			},
			[]string{"table1"},
		},
		{
			"Graph with Many Tables and Relationships",
			map[string]map[string]*schema.Table{
				"table1": {
					"table2": nil,
					"table3": nil,
				},
				"table2": {
					"table4": nil,
					"table5": nil,
				},
				"table3": {
					"table6": nil,
					"table7": nil,
				},
				"table4": {
					"table8": nil,
					"table9": nil,
				},
				"table5": {
					"table9": nil,
				},
				"table6": {
					"table10": nil,
					"table11": nil,
				},
				"table7": {
					"table11": nil,
				},
				"table8":  {},
				"table9":  {},
				"table10": {},
				"table11": {},
			},
			[]string{
				"table1", "table2", "table3",
				"table4", "table5", "table6",
				"table7", "table8", "table9",
				"table10", "table11",
			},
		},
	}

	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{g: &schema.Graph{
				Graph: tt.graph,
			}}

			result, err := g.TopologicalSort()
			if tt.expected == nil {
				require.Error(t, err, "result: %+#v", result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
