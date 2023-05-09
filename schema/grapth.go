package schema

import (
	"errors"
	"sort"
)

type Graph struct {
	Schema *Schema
	// map[текущая_таблица][таблицы_для_которых_текущая_это_внешняя]внешняя_таблица
	Graph map[string]map[string]*Table
}

func NewGraph(schema *Schema) *Graph {
	g := &Graph{
		Schema: schema,
	}
	g.build()
	return g
}

func (g *Graph) build() {
	g.Graph = make(map[string]map[string]*Table, len(g.Schema.Tables))
	for tablename, table := range g.Schema.Tables {
		foreignTables := make(map[string]*Table, len(table.ForeignKeys))
		g.Graph[tablename] = foreignTables
		for _, fk := range table.ForeignKeys {
			foreignTables[fk.Foreign.Table.Name.String()] = fk.Reference
		}
	}
}

//nolint:gocyclo // algo
func (g *Graph) TopologicalSort() ([]string, error) {
	// Create a slice to store the result
	result := make([]string, 0, len(g.Graph))

	// Initialize indegrees
	inDegrees := make(map[string]int)
	for parent, neighbors := range g.Graph {
		for neighbor := range neighbors {
			if parent == neighbor {
				// ссылка сама на себя не считается циклом
				continue
			}
			inDegrees[neighbor]++
		}
	}

	// sorted order
	keys := make([]string, 0, len(g.Graph))
	for tableName := range g.Graph {
		keys = append(keys, tableName)
	}
	sort.Strings(keys)

	// Add all nodes with no incoming edges to the queue
	queue := make([]string, 0, len(g.Graph))
	for _, node := range keys {
		inDegree := inDegrees[node]
		if inDegree == 0 {
			queue = append(queue, node)
		}
	}

	// Process the queue until it is empty
	for len(queue) > 0 {
		// Dequeue a node and add it to the result
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		var enqueue []string
		// Decrement the indegrees of all neighbors
		for neighbor := range g.Graph[node] {
			inDegrees[neighbor]--
			if inDegrees[neighbor] == 0 {
				enqueue = append(enqueue, neighbor)
			}
		}
		// sorted order
		sort.Strings(enqueue)
		queue = append(queue, enqueue...)
	}

	// Check if we encountered a cycle
	if len(result) != len(g.Graph) {
		return nil, errors.New("the graph contains a cycle")
	}

	return result, nil
}
