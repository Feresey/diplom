package schema

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
