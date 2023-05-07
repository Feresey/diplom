package generate

import "github.com/Feresey/mtest/schema"

type Generator struct {
	g *schema.Graph
}

func New(g *schema.Graph) *Generator {
	return &Generator{g: g}
}

func (g *Generator) Generate() error {
	tablesOrder, err := g.buildTablesInsertOrder()
	if err != nil {
		return err
	}

	_ = tablesOrder
	return nil
}

func (g *Generator) buildTablesInsertOrder() ([]string, error) {
	// g.g.Graph[]
	return nil, nil
}
