{{- range .TableNames }}
{{- with index $.Tables .}}
TABLE {{.Name}} (
    PRIMARY KEY {{with .PrimaryKey -}}{{.Name}} ({{.Columns | columnNames}}){{else}}NOT FOUND{{end}}
    {{- range .ForeignKeys }}
    FOREIGN KEY {{.Foreign.Name}} {{.Uniq.Name}}({{.Uniq.Columns | columnNames}})
    {{- end }}
    {{- $maxlen := 0 }}
    {{- range .Columns }}
    {{- if gt (len .Name) $maxlen }}
    {{- $maxlen = len .Name }}
    {{- end}}
    {{- end }}
    {{- range .Columns }}
    {{/* TODO constraint */}}
    {{.Name}} {{- space (len .Name) $maxlen }} {{.Type}} {{.Attributes}}
    {{- end }}
)
{{ end }}
{{- end }}
