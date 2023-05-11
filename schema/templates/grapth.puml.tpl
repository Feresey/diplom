@startuml grapth
{{- /* range tables */}}
{{- range .Schema.Tables}}
{{- $table := .}}
class {{$table.Name}} {
  {{- /* with pk */}}
  {{- with .PrimaryKey}}
  {{- range .Columns}}
  * {{.Name}}: {{template "smalltype" .}}
  {{- else}}
  <PK COLUMN NOT FOUND>
  {{- end}}
  --
  {{- /* with pk */}}
  {{- end}}

  {{- /* range fk */}}
  {{- range $fk := $table.ForeignKeys}}
    {{- range $fk.Foreign.Columns}}
  * {{.Name}}: {{template "smalltype" .}} REFERENCES {{$fk.Reference.Name}}({{$fk.ReferenceColumns | columnNames}})
    {{- end}}
  {{- /* range fk */}}
  {{- end}}
  {{- /* range columns */}}
  {{- range .ColumnNames}}{{with index $table.Columns .}}
    {{- if or (isPK $table .) (isFK $table .)}}
    {{- else}}
  {{.Name}}: {{template "smalltype" .}}
    {{- end}}
  {{- /* range columns */}}
  {{- end}}{{end}}
}
{{/* range tables */}}
{{- end}}

{{- $degrees := (.GetDepth)}}
{{- range $rel_from := ($.TopologicalSort)}}
{{- $relations := index $.Graph .}}
{{- range $rel_to, $table := $relations}}
{{$rel_from}} {{ repeat (index $degrees $rel_to) "-"}}{ {{$rel_to}}
{{- end}}
{{- end}}

@enduml
