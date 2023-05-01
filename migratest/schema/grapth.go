package schema

type Graph struct {
	schema *Schema

	Grapth map[string]map[string]*Table
}

func NewGraph(schema *Schema) *Graph {
	g := &Graph{schema: schema}
	g.build()
	return g
}

func (g *Graph) build() {
	// TODO пора делать граф
}
