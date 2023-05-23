package schema

import (
	"errors"
	"sort"
)

var ErrCycle = errors.New("graph contains a cycle")

type Graph struct {
	// map[текущая_таблица][таблицы_для_которых_текущая_это_внешняя]внешняя_таблица
	Graph map[string][]string
}

func (s *Schema) NewGraph() *Graph {
	graph := make(map[string][]string, len(s.Tables))
	for _, table := range s.Tables {
		refs := make([]string, 0, len(table.ReferencedBy))
		for ref := range table.ReferencedBy {
			refs = append(refs, ref)
		}
		graph[table.String()] = refs
	}
	return &Graph{
		Graph: graph,
	}
}

func (g *Graph) GetDepth() map[string]int {
	// Initialize indegrees
	inDegrees := make(map[string]int)
	for parent, neighbors := range g.Graph {
		for _, neighbor := range neighbors {
			if parent == neighbor {
				// ссылка сама на себя не считается циклом
				continue
			}
			inDegrees[neighbor]++
		}
	}
	return inDegrees
}

func (g *Graph) TopologicalSort() ([]string, error) {
	// Create a slice to store the result
	result := make([]string, 0, len(g.Graph))

	// Initialize indegrees
	inDegrees := g.GetDepth()

	// sorted order
	keys := make([]string, 0, len(g.Graph))
	for tableOID := range g.Graph {
		keys = append(keys, tableOID)
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
		for _, neighbor := range g.Graph[node] {
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
		return nil, ErrCycle
	}

	if len(result) == 0 {
		result = nil
	}
	return result, nil
}
