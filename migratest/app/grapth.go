package app

import "github.com/Feresey/diplom/migratest/schema"

type (
	Relation struct {
		Column schema.Column
		One    *schema.ToOneRelationship
		Many   *schema.ToManyRelationship
	}

	// MultiGrapth - матрица смежности, для каждых двух таблиц записываются все их связи.
	MultiGrapth struct {
		Rels             [][][]Relation
		TableIndexByName map[string]int
		TableValByIndex  map[int]*schema.Table
	}

	grapthBuilder struct {
		tables []*schema.Table
		grapth *MultiGrapth
	}
)

func BuildGrapth(tables []*schema.Table) *MultiGrapth {
	m := make([][][]Relation, len(tables))
	for i := range tables {
		m[i] = make([][]Relation, len(tables))
	}
	g := &grapthBuilder{
		tables: tables,
		grapth: &MultiGrapth{
			Rels:             m,
			TableIndexByName: make(map[string]int),
			TableValByIndex:  make(map[int]*schema.Table),
		},
	}
	for idx, t := range tables {
		g.grapth.TableIndexByName[t.Name] = idx
		g.grapth.TableValByIndex[idx] = t
	}
	g.Build()

	return g.grapth
}

func (m *MultiGrapth) Index(s string) int        { return m.TableIndexByName[s] }
func (m *MultiGrapth) Value(i int) *schema.Table { return m.TableValByIndex[i] }

func (b *grapthBuilder) Build() {
	for _, t := range b.tables {
		for _, rel := range t.ToOneRelationships {
			rels := b.grapth.Rels[b.grapth.Index(rel.Table)][b.grapth.Index(rel.ForeignTable)]
			rels = append(rels, Relation{
				Column: b.grapth.TableValByIndex[b.grapth.Index(rel.Table)].GetColumn(rel.Column),
				One:    rel,
			})
			b.grapth.Rels[b.grapth.Index(rel.Table)][b.grapth.Index(rel.ForeignTable)] = rels
		}
		for _, rel := range t.ToManyRelationships {
			rels := b.grapth.Rels[b.grapth.Index(rel.Table)][b.grapth.Index(rel.ForeignTable)]
			rels = append(rels, Relation{
				Column: b.grapth.TableValByIndex[b.grapth.Index(rel.Table)].GetColumn(rel.Column),
				Many:   rel,
			})
			b.grapth.Rels[b.grapth.Index(rel.Table)][b.grapth.Index(rel.ForeignTable)] = rels
		}
	}
}
