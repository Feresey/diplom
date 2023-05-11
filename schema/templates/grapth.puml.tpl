@startuml grapth
{{- /* range tables */}}
{{- range $.Schema.Tables}}
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
  {{- range .Columns}}
    {{- if or (isPK $table .) (isFK $table .)}}
    {{- else}}
  {{.Name}}: {{template "smalltype" .}}
    {{- end}}
  {{- /* range columns */}}
  {{- end}}
}
{{/* range tables */}}
{{- end}}

{{- $degrees := ($.Graph.GetDepth)}}
{{- range $rel_from := ($.Graph.TopologicalSort)}}
{{- $relations := index $.Graph.Graph .}}
{{- range $rel_to, $table := $relations}}
{{index $.Schema.Tables $rel_from}}
{{- " "}}{{- repeat (index $degrees $rel_to) "-" -}}{{"{ "}}
{{- index $.Schema.Tables $rel_to}}
{{- end}}
{{- end}}

@enduml
