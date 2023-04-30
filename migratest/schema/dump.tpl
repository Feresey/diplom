{{- define "dump-typename"}}
{{.TypeName}}{{if .HasMaxCharLength}}({{.MaxCharLength}}){{end}}
{{- end}}

{{- define "dump-type"}}
    {{- with .Type}}
        {{- if eq . "BUILTIN"}}
            {{- template "dump-typename" .}}
        {{- else if eq . "ARRAY"}}
            {{- with .ArrayType}}
                {{- template "dump-typename" .}}
            {{- else -}}
                <ARRAY TYPE IS NOT SPECIFIED>
            {{- end}}
        {{- else if eq . "ENUM"}}
            {{- template "dump-typename" .}}
        {{- else if eq . "RANGE"}}
            {{- template "dump-typename" .}}
        {{- else if eq . "COMPOSITE"}}
            {{- template "dump-typename" .}}
        {{- else if eq . "DOMAIN"}}
            {{- template "dump-typename" .}}
        {{- else -}}
            <DATA TYPE IS UNDEFINED>
        {{- end}}
    {{- else -}}
        <DATA TYPE IS NOT SPECIFIED>
    {{- end}}
{{- end}}

{{- /* range tables */}}
{{- range .TableNames }}
{{- $table := index $.Tables .}}
TABLE {{$table.Name}} (
{{- /* 
{{- with $table.PrimaryKey}}
    PRIMARY KEY {{.Name}} ({{.Columns | columnNames}})
{{- end}}
{{- range $fk_id, $fk := $table.ForeignKeys}}
    FOREIGN KEY {{$fk.Foreign.Name}} {{$fk.Foreign.Name}}({{$fk.Uniq.Columns | columnNames}}) {{"" -}}
    REFERENCES {{$fk.Uniq.Name}} ({{$fk.Foreign.Columns | columnNames}})
{{- end}}
*/}}
{{- /* range columns */}}
{{- $maxlen := 0}}{{range $table.ColumnNames}}{{if gt (len .) $maxlen}}{{$maxlen = len .}}{{end}}{{end}}
{{- range $table.ColumnNames }}{{$column := index $table.Columns .}}
    {{$column.Name}} {{- space (len $column.Name) $maxlen }} {{template "dump-type"}} {{$column.Attributes}}
    {{- range $fk_id, $fk := $table.ForeignKeys}}
    {{- with index $fk.Foreign.Columns $column.Name}}
    {{- ""}} {{ $fk.Foreign.Name}} REFERENCES {{$fk.Uniq.Table.Name}}({{$fk.Uniq.Columns | columnNames}})
    {{- end}}
    {{- end}}
    {{- with $table.PrimaryKey}}{{with index .Columns $column.Name}}
    {{- ""}} PRIMARY KEY
    {{- end}}{{end}}
{{- end}}
{{- /* range constraints */}}
{{- range $table.Constraints }}
    {{- $t := .Type.String}}
    {{- if eq $t "PK"}}
    {{- else if eq $t "FK"}}
    {{- else}}
    CONSTRAINT {{$t | upper}} {{.Name}} ({{.Columns | columnNames}})
    {{- end}}
    {{- if eq $t "Unique"}}
        {{- if .NullsNotDistinct}} NULLS NOT DISTINCT{{end}}
    {{- else if eq $t "Check" }}
        {{- ""}} {{.Definition}}
    {{- end}}
{{- end}}
)
{{/* range tables */}}{{end}}
