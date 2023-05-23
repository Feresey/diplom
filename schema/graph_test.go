package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopologicalSort(t *testing.T) {
	tables := []struct {
		name     string
		graph    map[string][]string
		expected []string
		wantErr  error
	}{
		{
			"Single Table",
			map[string][]string{
				"1": nil,
			},
			[]string{"1"},
			nil,
		},
		{
			"No Tables",
			nil,
			nil,
			nil,
		},
		{
			"No Relationships",
			map[string][]string{
				"1": nil,
				"2": nil,
			},
			[]string{"1", "2"},
			nil,
		},
		{
			"Simple Graph with One Cycle",
			map[string][]string{
				"1": {"2", "3"},
				"2": {
					"1", // This is the cycle
					"4",
				},
				"3": {"4"},
				"4": {},
			},
			nil,
			ErrCycle,
		},
		{
			"Simple Graph with Two Cycles",
			map[string][]string{
				"1": {"2", "3"},
				"2": {"4"},
				"3": {"2", "4"},
				"4": {
					"1", // This is the first cycle
					"5",
				},
				"5": {
					"4", // This is the second cycle
				},
			},
			nil,
			ErrCycle,
		},
		{
			"Graph with All Tables Connected to One",
			map[string][]string{
				"1": {"2", "3", "4"},
				"2": {"3", "4"},
				"3": {"4"},
				"4": {},
			},
			[]string{"1", "2", "3", "4"},
			nil,
		},
		{
			"Graph with Tables Connected in a Chain",
			map[string][]string{
				"1": {"2"},
				"2": {"3"},
				"3": {"4"},
				"4": {},
			},
			[]string{"1", "2", "3", "4"},
			nil,
		},
		{
			"Graph with Multiple Independent Cycles",
			map[string][]string{
				"1": {"2", "3"},
				"2": {"4"},
				"3": {"5"},
				"4": {"1"},
				"5": {"3"},
			},
			nil,
			ErrCycle,
		},
		{
			"Graph with Self-Referencing Table",
			map[string][]string{
				"1": {"1"},
			},
			[]string{"1"},
			nil,
		},
		{
			"Graph with Many Tables and Relationships",
			map[string][]string{
				"1":  {"2", "3"},
				"2":  {"4", "5"},
				"3":  {"6", "7"},
				"4":  {"8", "9"},
				"5":  {"9"},
				"6":  {"10", "11"},
				"7":  {"11"},
				"8":  {},
				"9":  {},
				"10": {},
				"11": {},
			},
			[]string{
				"1", "2", "3",
				"4", "5", "6",
				"7", "8", "9",
				"10", "11",
			},
			nil,
		},
	}

	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			g := &Graph{
				Graph: tt.graph,
			}

			result, err := g.TopologicalSort()
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.wantErr.Error())
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
