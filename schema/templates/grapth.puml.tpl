@startuml grapth
{{- /* range tables */}}
{{- range $.Schema.Tables}}
{{- $table := .}}
class {{$table.Name}} {
  {{- /* with pk */}}
  {{- with .PrimaryKey}}
  {{- range .Columns}}
  * {{.}}: {{template "smalltype" (index $table.Columns .)}}
  {{- else}}
  <PK COLUMN NOT FOUND>
  {{- end}}
  --
  {{- /* with pk */}}
  {{- end}}

  {{- /* range fk */}}
  {{- range $fk := $table.ForeignKeys}}
    {{- range $fk.Constraint.Columns}}
  * {{.}}: {{template "smalltype" (index $table.Columns .)}} REFERENCES {{$fk.ReferenceTable}}({{join "," $fk.ReferenceColumns}})
    {{- end}}
  {{- /* range fk */}}
  {{- end}}
  {{- /* range columns */}}
  {{- range .Columns}}
    {{- if or (isPK $table .Name) (isFK $table .Name)}}
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
