package schema

import (
	"embed"
	"io"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"golang.org/x/xerrors"
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
			Graph:  s.NewGraph(),
		}
	default:
		return xerrors.Errorf("undefined template name: %s", tplName)
	}

	return dump(w, tplName, data)
}

func dump(w io.Writer, tplName TemplateName, data any) error {
	t := template.New("").
		Funcs(sprig.TxtFuncMap()).
		Funcs(template.FuncMap{
			"space": func(namelen int, maxlen int) string {
				return strings.Repeat(" ", maxlen-namelen)
			},
			"isPK": func(t Table, col string) bool {
				if t.PrimaryKey == nil {
					return false
				}
				for _, colname := range t.PrimaryKey.Columns {
					if colname == col {
						return true
					}
				}
				return false
			},
			"isFK": func(t Table, col string) bool {
				for _, fk := range t.ForeignKeys {
					for _, colname := range fk.Constraint.Columns {
						if colname == col {
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
