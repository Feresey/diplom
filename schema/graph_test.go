package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopologicalSort(t *testing.T) {
	tables := []struct {
		name     string
		graph    map[int]map[int]*Table
		expected []int
	}{
		{
			"Single Table",
			map[int]map[int]*Table{
				1: {},
			},
			[]int{1},
		},
		{
			"No Tables",
			map[int]map[int]*Table{},
			[]int{},
		},
		{
			"No Relationships",
			map[int]map[int]*Table{
				1: {},
				2: {},
			},
			[]int{1, 2},
		},
		{
			"Simple Graph with One Cycle",
			map[int]map[int]*Table{
				1: {
					2: nil,
					3: nil,
				},
				2: {
					1: nil, // This is the cycle
					4: nil,
				},
				3: {
					4: nil,
				},
				4: {},
			},
			nil,
		},
		{
			"Simple Graph with Two Cycles",
			map[int]map[int]*Table{
				1: {
					2: nil,
					3: nil,
				},
				2: {
					4: nil,
				},
				3: {
					2: nil,
					4: nil,
				},
				4: {
					1: nil, // This is the first cycle
					5: nil,
				},
				5: {
					4: nil, // This is the second cycle
				},
			},
			nil,
		},
		{
			"Graph with All Tables Connected to One",
			map[int]map[int]*Table{
				1: {
					2: nil,
					3: nil,
					4: nil,
				},
				2: {
					3: nil,
					4: nil,
				},
				3: {
					4: nil,
				},
				4: {},
			},
			[]int{1, 2, 3, 4},
		},
		{
			"Graph with Tables Connected in a Chain",
			map[int]map[int]*Table{
				1: {
					2: nil,
				},
				2: {
					3: nil,
				},
				3: {
					4: nil,
				},
				4: {},
			},
			[]int{1, 2, 3, 4},
		},
		{
			"Graph with Multiple Independent Cycles",
			map[int]map[int]*Table{
				1: {
					2: nil,
					3: nil,
				},
				2: {
					4: nil,
				},
				3: {
					5: nil,
				},
				4: {
					1: nil,
				},
				5: {
					3: nil,
				},
			},
			nil,
		},
		{
			"Graph with Self-Referencing Table",
			map[int]map[int]*Table{
				1: {
					1: nil,
				},
			},
			[]int{1},
		},
		{
			"Graph with Many Tables and Relationships",
			map[int]map[int]*Table{
				1: {
					2: nil,
					3: nil,
				},
				2: {
					4: nil,
					5: nil,
				},
				3: {
					6: nil,
					7: nil,
				},
				4: {
					8: nil,
					9: nil,
				},
				5: {
					9: nil,
				},
				6: {
					10: nil,
					11: nil,
				},
				7: {
					11: nil,
				},
				8:  {},
				9:  {},
				10: {},
				11: {},
			},
			[]int{
				1, 2, 3,
				4, 5, 6,
				7, 8, 9,
				10, 11,
			},
		},
	}

	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			g := &Graph{
				Graph: tt.graph,
			}

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
