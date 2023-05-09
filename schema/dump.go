package schema

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

//go:embed templates/*.tpl
var dumptpl embed.FS

type TemplateName string

const (
	DumpSchemaTemplate TemplateName = "schema.sql.tpl"
	DumpTypesTemplate  TemplateName = "types.tpl"
	DumpGrapthTemplate TemplateName = "grapth.puml.tpl"
)

func (g Graph) MarshalJSON() ([]byte, error) {
	type typ struct {
		Name string   `json:"name,omitempty"`
		Type DataType `json:"type,omitempty"`
	}
	type index struct {
		Name    string   `json:"name,omitempty"`
		Columns []string `json:"columns,omitempty"`

		Unique           bool `json:"unique,omitempty"`
		Primary          bool `json:"primary,omitempty"`
		NullsNotDistinct bool `json:"nulls_not_distinct,omitempty"`
	}
	type constraint struct {
		Type    ConstraintType `json:"type,omitempty"`
		Columns []string       `json:"columns,omitempty"`
		Index   *index         `json:"index,omitempty"`
	}
	type attributes struct {
		NotNullable      bool   `json:"not_nullable,omitempty"`
		HasCharMaxLength bool   `json:"has_char_max_length,omitempty"`
		CharMaxLength    int    `json:"char_max_length,omitempty"`
		ArrayDims        int    `json:"array_dims,omitempty"`
		HasDefault       bool   `json:"has_default,omitempty"`
		IsGenerated      bool   `json:"is_generated,omitempty"`
		Default          string `json:"default,omitempty"`
		IsNumeric        bool   `json:"is_numeric,omitempty"`
		NumericPrecision int    `json:"numeric_precision,omitempty"`
		NumericScale     int    `json:"numeric_scale,omitempty"`
	}
	type column struct {
		Type typ        `json:"type,omitempty"`
		Attr attributes `json:"attr,omitempty"`
	}
	type table struct {
		Columns     map[string]column     `json:"columns,omitempty"`
		Constraints map[string]constraint `json:"constraints,omitempty"`
		Indexes     map[string]index      `json:"indexes,omitempty"`
	}

	type schema struct {
		Tables map[string]table    `json:"tables,omitempty"`
		Graph  map[string][]string `json:"graph,omitempty"`
	}

	makeColumns := func(c map[string]*Column) map[string]column {
		res := make(map[string]column, len(c))
		for name, col := range c {
			res[name] = column{
				Type: typ{
					Name: col.Type.TypeName.String(),
					Type: col.Type.Type,
				},
				Attr: attributes{
					NotNullable:      col.Attributes.NotNullable,
					HasCharMaxLength: col.Attributes.HasCharMaxLength,
					CharMaxLength:    col.Attributes.CharMaxLength,
					ArrayDims:        col.Attributes.ArrayDims,
					HasDefault:       col.Attributes.HasDefault,
					IsGenerated:      col.Attributes.IsGenerated,
					Default:          col.Attributes.Default,
					IsNumeric:        col.Attributes.IsNumeric,
					NumericPrecision: col.Attributes.NumericPrecision,
					NumericScale:     col.Attributes.NumericScale,
				},
			}
		}
		return res
	}

	mapKeys := func(c map[string]*Column) (res []string) {
		for key := range c {
			res = append(res, key)
		}
		return res
	}

	tables := make(map[string]table, len(g.Schema.Tables))

	for _, t := range g.Schema.Tables {
		tbl := table{
			Columns:     makeColumns(t.Columns),
			Constraints: make(map[string]constraint, len(t.Constraints)),
			Indexes:     make(map[string]index, len(t.Indexes)),
		}

		for name, c := range t.Constraints {
			cc := constraint{
				Type:    c.Type,
				Columns: mapKeys(c.Columns),
			}
			if c.Index != nil {
				i := &index{
					Name:             c.Index.Name.String(),
					Columns:          mapKeys(c.Index.Columns),
					Unique:           c.Index.IsUnique,
					Primary:          c.Index.IsPrimary,
					NullsNotDistinct: c.Index.IsNullsNotDistinct,
				}
				cc.Index = i
			}

			tbl.Constraints[name] = cc
		}

		for name, idx := range t.Indexes {
			tbl.Indexes[name] = index{
				Name:             idx.Name.String(),
				Columns:          mapKeys(idx.Columns),
				Unique:           idx.IsUnique,
				Primary:          idx.IsPrimary,
				NullsNotDistinct: idx.IsNullsNotDistinct,
			}
		}

		tables[t.Name.String()] = tbl
	}

	graph := make(map[string][]string, len(tables))
	for tbl, neighbors := range g.Graph {
		for _, neighbor := range neighbors {
			graph[tbl] = append(graph[tbl], neighbor.Name.String())
		}
	}

	return json.Marshal(schema{
		Tables: tables,
		Graph:  graph,
	})
}

func (g *Graph) Dump(w io.Writer, tplName TemplateName) error {
	var data any
	switch tplName {
	case DumpSchemaTemplate:
		data = g.Schema
	case DumpTypesTemplate:
		data = g.Schema.Types
	case DumpGrapthTemplate:
		data = g
	default:
		return fmt.Errorf("undefined template name: %s", tplName)
	}

	return g.dump(w, tplName, data)
}

func (g *Graph) dump(w io.Writer, tplName TemplateName, data any) error {
	t := template.New("").
		Funcs(sprig.TxtFuncMap()).
		Funcs(template.FuncMap{
			"columnNames": func(cols map[string]*Column) string {
				names := make([]string, 0, len(cols))
				for name := range cols {
					names = append(names, name)
				}
				sort.Strings(names)
				return strings.Join(names, ", ")
			},
			"space": func(namelen int, maxlen int) string {
				return strings.Repeat(" ", maxlen-namelen)
			},
			"isPK": func(t *Table, col *Column) bool {
				if t.PrimaryKey == nil {
					return false
				}
				for colname := range t.PrimaryKey.Columns {
					if colname == col.Name {
						return true
					}
				}
				return false
			},
			"isFK": func(t *Table, col *Column) bool {
				for _, fk := range t.ForeignKeys {
					for colname := range fk.Foreign.Columns {
						if colname == col.Name {
							return true
						}
					}
				}
				return false
			},
		})
	tpl, err := t.ParseFS(dumptpl, "templates/*.tpl")
	if err != nil {
		return err
	}

	err = tpl.ExecuteTemplate(w, string(tplName), data)
	if err != nil {
		return err
	}

	return err
}
