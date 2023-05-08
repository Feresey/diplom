{{- /* range tables */}}
{{- range .TableNames }}
{{- $table := index $.Tables .}}
CREATE TABLE {{$table.Name}} (
{{- /* range columns */}}
{{- $maxlen := 0}}{{range $table.ColumnNames}}{{if gt (len .) $maxlen}}{{$maxlen = len .}}{{end}}{{end}}
{{- range $table.ColumnNames }}{{$column := index $table.Columns .}}
    {{$column.Name}} {{space (len $column.Name) $maxlen}}{{" "}}
    {{- template "coltype" $column}}
    {{- /* range foreign keys */}}
    {{- range $fk_id, $fk := $table.ForeignKeys}}
        {{- with index $fk.Foreign.Columns $column.Name}}
            {{- ""}} CONSTRAINT {{ $fk.Foreign.Name}} REFERENCES {{$fk.Reference.Name}}({{$fk.ReferenceColumns | columnNames}})
        {{- end}}
    {{- /* range foreign keys */}}
    {{- end}}
    {{- /* with primary key */}}
    {{- with $table.PrimaryKey}}{{with index .Columns $column.Name}}
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
