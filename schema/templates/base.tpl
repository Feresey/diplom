{{- define "coltype"}}
    {{- $column := $}}
    {{- $type := $column.Type}}
    {{- /* if array */}}
    {{- if eq $type.Type.String "Array"}}{{" "}}
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
    {{- /* if array */}}
    {{- end}}
    {{- /* with attributes */}}
    {{- with $column.Attributes}}
        {{- if .HasCharMaxLength}}({{.CharMaxLength}}){{end}}
        {{- if .IsNumeric -}}({{.NumericPrecision}},{{.NumericScale}}){{end}}
        {{- repeat .ArrayDims "[]"}}
        {{- if .NotNullable}} NOT NULL{{end}}
        {{- if .HasDefault}}
            {{- if .IsGenerated}} GENERATED ALWAYS
            {{- else}} DEFAULT
            {{- end}}
            {{- " "}}{{.Default}}
            {{- if .IsGenerated}} STORED{{end}}
        {{- end}}
    {{- /* with attributes */}}
    {{- end}}
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
    {{- end}}
{{- end}}