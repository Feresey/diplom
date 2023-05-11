package schema

import (
	"errors"
	"sort"
)

type Graph struct {
	// map[текущая_таблица][таблицы_для_которых_текущая_это_внешняя]внешняя_таблица
	Graph map[int]map[int]*Table
}

func NewGraph(tables map[int]*Table) *Graph {
	g := &Graph{}

	g.Graph = make(map[int]map[int]*Table, len(tables))
	for _, table := range tables {
		foreignTables := make(map[int]*Table, len(table.ForeignKeys))
		g.Graph[table.OID()] = foreignTables
		for _, ref := range table.ReferencedBy {
			foreignTables[ref.Table.OID()] = ref.Table
		}
	}
	return g
}

func (g *Graph) GetDepth() map[int]int {
	// Initialize indegrees
	inDegrees := make(map[int]int)
	for parent, neighbors := range g.Graph {
		for neighbor := range neighbors {
			if parent == neighbor {
				// ссылка сама на себя не считается циклом
				continue
			}
			inDegrees[neighbor]++
		}
	}
	return inDegrees
}

func (g *Graph) TopologicalSort() ([]int, error) {
	// Create a slice to store the result
	result := make([]int, 0, len(g.Graph))

	// Initialize indegrees
	inDegrees := g.GetDepth()

	// sorted order
	keys := make([]int, 0, len(g.Graph))
	for tableOID := range g.Graph {
		keys = append(keys, tableOID)
	}
	sort.Ints(keys)

	// Add all nodes with no incoming edges to the queue
	queue := make([]int, 0, len(g.Graph))
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

		var enqueue []int
		// Decrement the indegrees of all neighbors
		for neighbor := range g.Graph[node] {
			inDegrees[neighbor]--
			if inDegrees[neighbor] == 0 {
				enqueue = append(enqueue, neighbor)
			}
		}
		// sorted order
		sort.Ints(enqueue)
		queue = append(queue, enqueue...)
	}

	// Check if we encountered a cycle
	if len(result) != len(g.Graph) {
		return nil, errors.New("the graph contains a cycle")
	}

	return result, nil
}
