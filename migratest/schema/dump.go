package schema

import (
	"bytes"
	"embed"
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
)

func (s *Schema) Dump(w io.Writer, tplName TemplateName) error {
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
		})
	tpl, err := t.ParseFS(dumptpl, "templates/*.tpl")
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	err = tpl.ExecuteTemplate(&buf, string(tplName), s)
	if err != nil {
		_, _ = buf.WriteTo(w)
		return err
	}

	_, err = buf.WriteTo(w)
	return err
}
