package schema

import (
	"bytes"
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
	DumpSchemaTemplate TemplateName = "dump-schema.tpl"
	DumpTypesTemplate  TemplateName = "dump-types.tpl"
	DumpGrapthTemplate TemplateName = "dump-grapth.uml.tpl"
)

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

	var buf bytes.Buffer

	err = tpl.ExecuteTemplate(&buf, string(tplName), data)
	if err != nil {
		_, _ = buf.WriteTo(w)
		return err
	}

	// TODO частичная запись это правильно?
	_, err = buf.WriteTo(w)
	return err
}
