@startuml grapth

{{- define "dump-column-uml"}}
  {{- .Name}}:{{" "}}
  {{- $column := .}}
  {{- $type := $column.Type}}
  {{- /* if array */}}
  {{- if eq $type.Type.String "Array"}}
      {{- with $type.ArrayType}}
          {{- with .ElemType}}
              {{- .TypeName.String}}
              {{- if $column.Attributes.HasCharMaxLength -}}
              ({{$column.Attributes.CharMaxLength}})
              {{- end}}
              {{- repeat $column.Attributes.ArrayDims "[]"}}
          {{- else -}}
              <ARRAY ELEMENT TYPE IS NOT SPECIFIED>
          {{- end}}
      {{- else -}}
          <ARRAY TYPE IS NOT SPECIFIED>
      {{- end}}
  {{- else}}
      {{- $type.TypeName.String}}
      {{- if $column.Attributes.HasCharMaxLength -}}
      ({{$column.Attributes.CharMaxLength}})
      {{- end}}
      {{- repeat $column.Attributes.ArrayDims "[]"}}
  {{- /* if array */}}
  {{- end}}
  {{- with .Attributes}}
    {{- if .NotNullable}} NOT NULL{{end}}
    {{- if .HasDefault}} DEFAULT {{.Default}}{{end}}
  {{- end}}
{{- end}}

{{- /* range tables */}}
{{- range .Schema.Tables}}
{{- $table := .}}
class {{$table.Name}} {
  {{- /* with pk */}}
  {{- with .PrimaryKey}}
    {{- if eq (len .Columns) 1}}
  *   {{- range .Columns}}
        {{- template "dump-column-uml" .}}
      {{- else -}}
        <PK COLUMN NOT FOUND>
      {{- end}}
    {{- else}}
      {{- range .Columns}}
  *     {{- template "dump-column-uml" .}}
      {{- end}}
    {{- end}}
  --
  {{- /* with pk */}}
  {{- end}}

  {{- /* range fk */}}
  {{- range .ForeignKeys}}
    {{- $fk := .}}
    {{- range .Foreign.Columns}}
  *   {{- template "dump-column-uml" .}} REFERENCES {{$fk.Reference.Name}}({{$fk.ReferenceColumns | columnNames}})
    {{- end}}
  {{- /* range fk */}}
  {{- end}}
  {{- /* range columns */}}
  {{- range .ColumnNames}}{{with index $table.Columns .}}
    {{- if or (isPK $table .) (isFK $table .)}}
    {{- else}}
  {{template "dump-column-uml" .}}
    {{- end}}
  {{- /* range columns */}}
  {{- end}}{{end}}
}
{{/* range tables */}}
{{- end}}

{{- range $childName, $relations := .Grapth}}
{{- $child := index $.Schema.Tables $childName}}
{{- range $parentName, $parent := $relations}}
{{$parent.Name}} --{ {{$child.Name}}
{{- end}}
{{- end}}

@enduml
