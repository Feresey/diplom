{{- range . }}
CREATE TYPE {{.TypeName}} {{.Type.String | upper}}
{{- /* switch Type */}}
{{- $type := .}}
{{- with .Type.String}}
{{- if eq . "Base"}}
{{- else if eq . "Array" -}}
    {{- with $type.ElemType}}{{" " -}}
        {{- .TypeName}}
    {{- else}}{{" " -}}
        <ARRAY ELEMENT TYPE IS NOT SPECIFIED>
    {{- end}}
{{- else if eq . "Enum"}} (
    {{- $vallen := len $type.EnumValues}}
    {{- range $idx, $value := $type.EnumValues }}
    {{$value | quote}}{{if eq $vallen (add $idx 1)}}{{else}},{{end}}
    {{- end }}
)
{{- else if eq . "Composite"}} ({{/*TODO*/}})
{{- else if eq . "Domain" -}}
    {{- with $type.ElemType}}{{" " -}}
        {{.TypeName}}
    {{- else}}{{" " -}}
        <DOMAIN BASE TYPE IS NOT SPECIFIED>
    {{- end}}
    {{- with $type.DomainAttributes}}
        {{- if .HasCharMaxLength}}({{.CharMaxLength}})
        {{- end}}
        {{- if gt .ArrayDims 0}}
            {{- repeat .ArrayDims "[]"}}
        {{- end}}
        {{- if not .NotNullable}} NOT NULL
        {{- end}}
    {{- end}}
{{- else if eq . "Range" -}}
    {{- with $type.ElemType}}{{" " -}}
        {{.TypeName}}
    {{- else}}{{" " -}}
        <RANGE ELEMENT TYPE IS NOT SPECIFIED>
    {{- end}}
{{- /* switch type */}}
{{- end}}{{end}};
{{- /* range types */}}
{{- end}}
