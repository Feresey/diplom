package schema

import (
	"embed"
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
	DumpGrapthTemplate TemplateName = "grapth.puml.tpl"
)

func (s *Schema) Dump(w io.Writer, tplName TemplateName) error {
	var data any
	switch tplName {
	case DumpSchemaTemplate:
		data = s
	case DumpGrapthTemplate:
		data = struct {
			Schema *Schema
			Graph  *Graph
		}{
			Schema: s,
			Graph:  NewGraph(s.Tables),
		}
	default:
		return fmt.Errorf("undefined template name: %s", tplName)
	}

	return dump(w, tplName, data)
}

func dump(w io.Writer, tplName TemplateName, data any) error {
	t := template.New("").
		Funcs(sprig.TxtFuncMap()).
		Funcs(template.FuncMap{
			"columnNames": func(cols map[int]*Column) string {
				names := make([]string, 0, len(cols))
				for _, col := range cols {
					names = append(names, col.Name)
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
				for colnum := range t.PrimaryKey.Columns {
					if colnum == col.ColNum {
						return true
					}
				}
				return false
			},
			"isFK": func(t *Table, col *Column) bool {
				for _, fk := range t.ForeignKeys {
					for colnum := range fk.Foreign.Columns {
						if colnum == col.ColNum {
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
