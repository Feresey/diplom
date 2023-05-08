package generate

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/Feresey/mtest/schema"
	"go.uber.org/zap"
)

type Generator struct {
	log   *zap.Logger
	g     *schema.Graph
	order []string
}

func New(
	log *zap.Logger,
	graph *schema.Graph,
) (*Generator, error) {
	g := &Generator{
		log: log,
		g:   graph,
	}

	order, err := g.TopologicalSort()
	if err != nil {
		return g, err
	}
	g.log.Info("tables insert order", zap.Strings("order", order))
	return g, nil
}

type Check struct {
	Columns []*schema.Column
	Values  []string
}

func (c *Check) AddColumn(col *schema.Column) {
	c.Columns = append(c.Columns, col)
}

func (c *Check) AddValues(vals ...string) {
	c.Values = append(c.Values, vals...)
}

func (c *Check) AddValuesQuote(vals ...string) {
	c.AddValuesProcess(strconv.Quote, vals...)
}

func (c *Check) AddValuesProcess(f func(string) string, vals ...string) {
	for _, v := range vals {
		c.Values = append(c.Values, f(v))
	}
}

func (g *Generator) Generate() error {
	for _, tableName := range g.order {
		table := g.g.Schema.Tables[tableName]

		checks := g.genTableRules(table)
	}
	return nil
}

func (g *Generator) genTableRules(table *schema.Table) []Check {
	var checks []Check
	for _, col := range table.Columns {
		// TODO для FK колонок только брать значения
		var check Check
		check.AddColumn(col)
		g.genColumnChecks(&check, col)
		if !col.Attributes.NotNullable {
			check.AddValuesQuote("NULL")
		}

		if col.Type.TypeName.Name == "text" || Aliases[col.Type.TypeName.Name] == "text" {
			if col.Attributes.HasCharMaxLength {
				check.AddValues(fmt.Sprintf("makestrlen(%d)", col.Attributes.CharMaxLength))
			}
		}

		checks = append(checks, check)
	}


	uniqueIndexes := make(map[string]*schema.Index)

	for indexName, index := range table.Indexes {
		if index.IsUnique {
			uniqueIndexes[indexName] = index
		}
	}

	// TODO для всех уникальных индексов
	/*
	1. для каждого индекса перебрать все возможные сочетания уникальных значений его колонок
	2. каждое такое сочетание проверить на то что оно не нарушает этот индекс
	3. если такого сочетания нет, то исключить этот индекс из перебора и перейти к шагу 1
	4. если найдено такое сочетание то запомнить это сочетание и перейти к следующему индексу
	5. для следующего индекса перебрать все возможные сочетания уникальных значений его колонок, за исключением выбранных колонок
	6. если для следующего индекса такого сочетания нет, то вернуться к предыдущему индексу.
	7. если предыдущего индекса нет, то исключить текущий индекс из перебора и перейти к шагу 1
	8. если следующего индекса нет, выбранные значения колонок - собраны из значений колонок, которые уже есть в таблице.

	*/
	for _, index := range uniqueIndexes {

	}

	return checks
}

func (g *Generator) baseTypesChecks(check *Check, typeName string) {
	check.AddValuesQuote(Checks[Aliases[typeName]]...)
	check.AddValuesQuote(Checks[typeName]...)
}

func (g *Generator) genColumnChecks(check *Check, col *schema.Column) {
	switch col.Type.Type {
	case schema.DataTypeBase:
		g.baseTypesChecks(check, col.Type.TypeName.Name)
	case schema.DataTypeArray:
		// dims := col.Attributes.ArrayDims
	case schema.DataTypeEnum:
		check.AddValuesProcess(func(s string) string {
			return fmt.Sprintf("'%s'::%s", s, col.Type.EnumType.TypeName)
		}, col.Type.EnumType.Values...)
	case schema.DataTypeDomain:
		g.baseTypesChecks(check, col.Type.DomainType.ElemType.TypeName.Name)
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
