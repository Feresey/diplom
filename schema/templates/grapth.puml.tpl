@startuml grapth
{{- /* range tables */}}
{{- range .Schema.Tables}}
{{- $table := .}}
class {{$table.Name}} {
  {{- /* with pk */}}
  {{- with .PrimaryKey}}
    {{- if eq (len .Columns) 1}}
  *   {{- range .Columns}}
        {{- template "coltype" .}}
      {{- else -}}
        <PK COLUMN NOT FOUND>
      {{- end}}
    {{- else}}
      {{- range .Columns}}
  *     {{- template "coltype" .}}
      {{- end}}
    {{- end}}
  --
  {{- /* with pk */}}
  {{- end}}

  {{- /* range fk */}}
  {{- range $fk := $table.ForeignKeys}}
    {{- range $fk.Foreign.Columns}}
  *   {{- template "coltype" .}} REFERENCES {{$fk.Reference.Name}}({{$fk.ReferenceColumns | columnNames}})
    {{- end}}
  {{- /* range fk */}}
  {{- end}}
  {{- /* range columns */}}
  {{- range .ColumnNames}}{{with index $table.Columns .}}
    {{- if or (isPK $table .) (isFK $table .)}}
    {{- else}}
  {{template "coltype" .}}
    {{- end}}
  {{- /* range columns */}}
  {{- end}}{{end}}
{{- /* range constraints */}}
{{- with $table.Constraints}}
  --
{{- range $table.Constraints }}
  {{- $t := .Type.String}}
  {{- if eq $t "PK"}}
  {{- else if eq $t "FK"}}
  {{- else}}
  **{{.Name}}**: CONSTRAINT {{$t | upper}} ({{.Columns | columnNames}})
  {{- end}}
  {{- if eq $t "Unique"}}
    {{- with .Index}}{{if .IsNullsNotDistinct}} NULLS NOT DISTINCT{{end}}{{end}}
  {{- else if eq $t "Check" }}
    {{- ""}} {{.Definition}}
  {{- end}}
{{- /* range constraints */}}
{{- end}}{{end}}
{{- /* range indexes */}}
{{- with $table.Indexes}}
  --
{{- range $table.Indexes }}
  **{{.Name}}**: {{.Definition}}
{{- /* range indexes */}}
{{- end}}{{end}}
}
{{/* range tables */}}
{{- end}}

{{- range $parentName, $relations := .Graph}}
{{- $parent := index $.Schema.Tables $parentName}}
{{- range $child := $relations}}
{{$parent.Name}} --{ {{$child.Name}}
{{- end}}
{{- end}}

@enduml
