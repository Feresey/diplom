{{- range .TableNames }}
{{- with index $.Tables .}}
TABLE {{.Name.String}} (
    PK: {{with .PrimaryKey -}}{{.Name.String}} ({{.Columns | columnNames}}){{else}}NOT FOUND{{end}}
    FK: {{if eq (len .ForeignKeys) 0 }}NOT FOUND{{else}}
    {{- range $fkName, $fk := .ForeignKeys }}
      {{ $fkName }}:
        Table: {{ $fk.Uniq.Name.Schema }}.{{ $fk.Uniq.Name.Name }}
        Columns: {{ $fk.Uniq.Columns | columnNames }}
    {{- end }}
    {{- end }}
    Referenced By:
    {{- range $refName, $ref := .ReferencedBy }}
      {{ $refName }}:
        Table: {{ $ref.Table.Name.Schema }}.{{ $ref.Table.Name.Name }}
        Columns: {{ $ref.Columns | columnNames }}
    {{- end }}
    {{- /*
    Constraints:
    {{- range $cName, $c := .Constraints }}
      {{ $cName }}:
        Type: {{ $c.Type.String }}
        Columns: {{ $c.Columns | columnNames }}
    {{- end }}
    */}}
)
{{ end }}
{{- end }}

{{- /* 
Constraints:
{{- range .ConstraintNames }}
  {{ . }}:
    Type: {{ index $.Constraints .Type.String }}
    Table: {{ .Table.Name.Schema }}.{{ .Table.Name.Name }}
    Columns: {{ .Columns | columnNames }}
{{end}}
*/}}