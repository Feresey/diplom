{{- /* range tables */}}
{{- range $table := $.Tables }}
CREATE TABLE {{$table.Name}} (
{{- /* range columns */}}
{{- $maxlen := 0}}{{range $table.Columns}}{{$curlen := len .Name}}{{if gt $curlen $maxlen}}{{$maxlen = $curlen}}{{end}}{{end}}
{{- range $column := $table.Columns }}
    {{$column.Name}} {{space (len $column.Name) $maxlen}}{{" "}}
    {{- template "coltype" $column}}
{{- /* range columns */}}
{{- end}}
{{- /* range constraints */}}
{{- range $table.Constraints }}
    {{- $t := .Type.String}}
    {{- if eq $t "PK"}}
    PRIMARY KEY ({{join "," .Columns}})
    {{- else if eq $t "FK"}}
    {{- with index $table.ForeignKeys .Name}}
    FOREIGN KEY {{.Constraint.Name}}({{join "," .Constraint.Columns}}) REFERENCES {{.ReferenceTable}}({{join "," .ReferenceColumns}})
    {{- else}}
    CONSTRAINT {{.Name}} {{$t | upper}} ({{join "," .Columns}})
    {{- end}}
    {{- if eq $t "Unique"}}
        {{- with .Index}}{{if .IsNullsNotDistinct}} NULLS NOT DISTINCT{{end}}{{end}}
    {{- else if eq $t "Check" }}
        {{- ""}} {{.Definition}}
    {{- end}}
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