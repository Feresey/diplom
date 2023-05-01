{{- range . }}
TYPE {{.TypeName}} {{.Type.String | upper}}
{{- /* switch Type */}}
{{- $type := .}}
{{- with .Type.String}}
{{- if eq . "Base"}}
{{- else if eq . "Array" -}}
    {{- with $type.ArrayType}}
        {{- with .ElemType}}{{" " -}}
            {{- .TypeName}}
        {{- else}}{{" " -}}
            <ARRAY ELEMENT TYPE IS NOT SPECIFIED>
        {{- end}}
    {{- else}}{{" " -}}
        <ARRAY TYPE IS NOT SPECIFIED>
    {{- end -}}
{{- else if eq . "Enum"}} (
    {{- with $type.EnumType -}}
        {{- $vallen := len $type.EnumType.Values}}
        {{- range $idx, $value := $type.EnumType.Values }}
    {{$value | quote}}{{if eq $vallen (add $idx 1)}}{{else}},{{end}}
        {{- end }}
    {{- else}} {{" " -}}
        <ENUM TYPE IS NOT SPECIFIED>
    {{- end}}
)
{{- else if eq . "Composite"}} ({{/*TODO*/}})
{{- else if eq . "Domain" -}}
    {{- with $type.DomainType}}
        {{- with .ElemType}}{{" " -}}
            {{.TypeName}}
        {{- else}}{{" " -}}
            <DOMAIN BASE TYPE IS NOT SPECIFIED>
        {{- end}}
        {{- with .Attributes}}
            {{- if .HasCharMaxLength}}({{.CharMaxLength}})
            {{- end}}
            {{- if gt .ArrayDims 0}}
                {{- repeat .ArrayDims "[]"}}
            {{- end}}
            {{- if not .NotNullable}} NOT NULL
            {{- end}}
        {{- end}}
    {{- else}}{{" " -}}
        <DOMAIN TYPE IS NOT SPECIFIED>
    {{- end -}}
{{- else if eq . "Range" -}}
    {{- with $type.RangeType}}
        {{- with .ElemType}}{{" " -}}
            {{.TypeName}}
        {{- else}}{{" " -}}
            <RANGE ELEMENT TYPE IS NOT SPECIFIED>
        {{- end}}
    {{- else}}{{" " -}}
        <RANGE TYPE IS NOT SPECIFIED>
    {{- end -}}
{{- /* switch type */}}
{{- end}}{{end}};
{{- /* range types */}}
{{- end}}
