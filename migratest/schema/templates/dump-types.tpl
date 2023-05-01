{{- range .Types }}
TYPE {{.TypeName}} {{.Type.String | upper}}
{{- /* switch Type */}}
{{- $type := .}}
{{- with .Type.String}}
{{- if eq . "Base"}}
{{- else if eq . "Array" -}}
    (
        {{- with $type.ArrayType}}
            {{- with .ElemType}}
                {{- .TypeName}}
            {{- else -}}
                <ARRAY ELEMENT TYPE IS NOT SPECIFIED>
            {{- end}}
        {{- else -}}
            <ARRAY TYPE IS NOT SPECIFIED>
        {{- end -}}
    )
{{- else if eq . "Enum"}}
 (
        {{- with $type.EnumType -}}
            {{- $vallen := len $type.EnumType.Values}}
            {{- range $idx, $value := $type.EnumType.Values }}
    {{$value | quote}}{{if eq $vallen (add $idx 1)}}{{else}},{{end}}
            {{- end }}
        {{- else -}}
            <ENUM TYPE IS NOT SPECIFIED>
        {{- end}}
)
{{- else if eq . "Composite" -}}
{{- else if eq . "Domain" -}}
{{- else if eq . "Range" -}}
{{- end}}{{end}}
{{- /* switch type */}}
{{- end}}
{{- /* range types */}}
