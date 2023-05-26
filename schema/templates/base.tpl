{{- define "basetype"}}
  {{- if eq .TypeName.Schema "pg_catalog"}}
    {{- .TypeName.Name}}
  {{- else}}
    {{- .}}
  {{- end}}
{{- end}}

{{- define "basecolattrs"}}
  {{- if .HasCharMaxLength}}({{.CharMaxLength}}){{end}}
  {{- if .IsNumeric -}}({{.NumericPrecision}},{{.NumericScale}}){{end}}
  {{- repeat .ArrayDims "[]"}}
  {{- if .NotNullable}} NOT NULL{{end}}
{{- end}}

{{- define "colattrs"}}
  {{- template "basecolattrs" .}}
  {{- if .HasDefault}}
    {{- if .IsGenerated}} GENERATED ALWAYS {{.Default}} STORED
    {{- else}} DEFAULT {{.Default}}
    {{- end}}
  {{- end}}
{{- end}}

{{- define "coltype"}}
  {{- $column := $}}
  {{- $type := $column.Type}}
  {{- /* if array */}}
  {{- if eq $type.Type.String "Array"}}{{" "}}
    {{- with $type.ElemType}}
      {{- template "basetype" .}}
      {{- with .DomainAttributes}}{{template "basecolattrs" .}}{{end}}
    {{- else -}}
      <ARRAY ELEMENT TYPE IS NOT SPECIFIED>
    {{- end}}
  {{- else}}
    {{- $type.TypeName}}
  {{- /* if array */}}
  {{- end}}
  {{- template "colattrs" $column.Attributes}}
{{- end}}

{{- define "smalltype"}}
  {{- $column := $}}
  {{- $type := $column.Type}}
  {{- if eq $type.TypeName.Schema "pg_catalog"}}
    {{- $type.TypeName.Name}}
  {{- else}}
    {{- $type}}
  {{- end}}
  {{- with $column.Attributes}}
    {{- if .HasCharMaxLength}}({{.CharMaxLength}}){{end}}
    {{- if .IsNumeric -}}({{.NumericPrecision}},{{.NumericScale}}){{end}}
    {{- repeat .ArrayDims "[]"}}
  {{- end}}
{{- end}}