{{- /* range tables */}}
{{- range $table := $.Tables }}
CREATE TABLE {{$table.Name}} (
{{- /* range columns */}}
{{- $maxlen := 0}}{{range $table.Columns}}{{$curlen := len .Name}}{{if gt $curlen $maxlen}}{{$maxlen = $curlen}}{{end}}{{end}}
{{- range $column := $table.Columns }}
    {{$column.Name}} {{space (len $column.Name) $maxlen}}{{" "}}
    {{- template "coltype" $column}}
    {{- /* range foreign keys */}}
    {{- range $fk_id, $fk := $table.ForeignKeys}}
        {{- with index $fk.Foreign.Columns $column.ColNum}}
            {{- ""}} CONSTRAINT {{ $fk.Foreign.Name}} REFERENCES {{$fk.Reference.Name}}({{$fk.ReferenceColumns | columnNames}})
        {{- end}}
    {{- /* range foreign keys */}}
    {{- end}}
    {{- /* with primary key */}}
    {{- with $table.PrimaryKey}}{{with index .Columns $column.ColNum}}
        {{- ""}} PRIMARY KEY
    {{- /* with primary key */}}
    {{- end}}{{end}}
{{- /* range columns */}}
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
        {{- with .Index}}{{if .IsNullsNotDistinct}} NULLS NOT DISTINCT{{end}}{{end}}
    {{- else if eq $t "Check" }}
        {{- ""}} {{.Definition}}
    {{- end}}
{{- /* range constraints */}}
{{- end}}
);
{{- /* range indexes */}}
{{- range $table.Indexes }}
{{.Definition}};
{{- /* range indexes */}}
{{- end}}
{{/* range tables */}}{{end}}

{{- template "types.tpl" .Types}}