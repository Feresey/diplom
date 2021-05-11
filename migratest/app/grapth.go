package app

import "github.com/Feresey/diplom/migratest/schema"

type (
	Relation struct {
		Table  string
		Column *schema.Column
		One    *schema.ToOneRelationship
		Many   *schema.ToManyRelationship
	}

	// MultiGrapth - матрица смежности, для каждых двух таблиц записываются все их связи.
	MultiGrapth [][][]Relation

	grapthBuilder struct {
		tables []schema.Table
		matrix MultiGrapth
	}
)

func BuildGrapth(tables []schema.Table) MultiGrapth {
	m := make([][][]Relation, len(tables))
	for i := range tables {
		m[i] = make([][]Relation, len(tables))
	}
	g := &grapthBuilder{
		tables: tables,
		matrix: m,
	}
	g.Build()

	return g.matrix
}

func (b *grapthBuilder) Build() {
}
