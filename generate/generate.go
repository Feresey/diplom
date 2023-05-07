package generate

import (
	"errors"
	"sort"

	"github.com/Feresey/mtest/schema"
	"go.uber.org/zap"
)

type Generator struct {
	log *zap.Logger
	g   *schema.Graph
}

func New(
	log *zap.Logger,
	g *schema.Graph,
) *Generator {
	return &Generator{
		log: log,
		g:   g,
	}
}

func (g *Generator) Generate() error {
	order, err := g.TopologicalSort()
	if err != nil {
		return err
	}
	g.log.Info("tables insert order", zap.Strings("order", order))
	for _, tableName := range order {
		table := g.g.Schema.Tables[tableName]

		g.genTableRules(table)
	}
	return nil
}

func (g *Generator) genTableRules(table *schema.Table) {
	for _, col := range table.Columns {
		g.genColumnChecks(col)
	}
}

func (g *Generator) genColumnChecks(col *schema.Column) {
	switch col.Type.Type {
	case schema.DataTypeBase:
	case schema.DataTypeArray:
	case schema.DataTypeEnum:
	case schema.DataTypeDomain:
	case schema.DataTypeComposite,
		schema.DataTypeRange,
		schema.DataTypeMultiRange,
		schema.DataTypePseudo:
	default:
	}
}

func (g *Generator) TopologicalSort() ([]string, error) {
	// Create a slice to store the result
	result := make([]string, 0, len(g.g.Graph))

	// Initialize indegrees
	inDegrees := make(map[string]int)
	for parent, neighbors := range g.g.Graph {
		for neighbor := range neighbors {
			if parent == neighbor {
				// ссылка сама на себя не считается циклом
				continue
			}
			inDegrees[neighbor]++
		}
	}

	// sorted order
	keys := make([]string, 0, len(g.g.Graph))
	for tableName := range g.g.Graph {
		keys = append(keys, tableName)
	}
	sort.Strings(keys)

	// Add all nodes with no incoming edges to the queue
	queue := make([]string, 0, len(g.g.Graph))
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
		for neighbor := range g.g.Graph[node] {
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
	if len(result) != len(g.g.Graph) {
		return nil, errors.New("the graph contains a cycle")
	}

	return result, nil
}
