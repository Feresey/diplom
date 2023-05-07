package schema

type Graph struct {
	Schema *Schema
	Graph  map[string]map[string]*Table
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
			foreignTables[fk.Reference.Name.String()] = fk.Reference
		}
	}
}