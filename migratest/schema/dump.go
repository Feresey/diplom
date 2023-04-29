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

//go:embed dump.tpl
var dumptpl embed.FS

func (s *Schema) Dump(w io.Writer) error {
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
	tpl, err := t.ParseFS(dumptpl, "*.tpl")
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	err = tpl.ExecuteTemplate(&buf, "dump.tpl", s)
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(w)
	return err
}
