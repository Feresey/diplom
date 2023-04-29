{{- /* range tables */}}
{{- range .TableNames }}
{{- $table := index $.Tables .}}
TABLE {{$table.Name}} (
    PRIMARY KEY {{with $table.PrimaryKey -}}{{.Name}} ({{.Columns | columnNames}}){{else}}NOT FOUND{{end}}
    {{- /* range fk */}}
    {{- range $fk_id, $fk := $table.ForeignKeys }}
    FOREIGN KEY {{$fk.Foreign.Name}} {{$fk.Uniq.Name}}({{$fk.Uniq.Columns | columnNames}}) REFERENCES {{$fk.Uniq.Name}} ({{$fk.Foreign.Columns | columnNames}})
    {{- /* range fk */}}{{end}}
    {{ $maxlen := 0 }}
    {{- range $table.ColumnNames }}{{- if gt (len .) $maxlen }}{{ $maxlen = len . }}{{end}}{{end}}
    {{- /* range columns */}}
    {{- range $table.ColumnNames }}
    {{- $column := index $table.Columns .}} {{- /* TODO single column constraint */}}
    {{$column.Name}} {{- space (len $column.Name) $maxlen }} {{$column.Type}} {{$column.Attributes}}
    {{- /* range columns */}}{{end}}
    {{- /* range constraints */}}
    {{- range $c_id, $con := $table.Constraints }}
    {{- $contype := $con.Type.String}}
    {{- /* switch contype */}}
    {{- if or (eq $contype "PK") (eq $contype "FK")}}
    {{- else}}
    CONSTRAINT {{$contype | upper}} {{$con.Name}} ({{$con.Columns | columnNames}})
    {{- if and (eq $contype "Unique") $con.NullsNotDistinct }} NULLS NOT DISTINCT{{end}}
    {{- if eq $contype "Check" }} DEFINITION {{.Definition}}{{end}}
    {{- /* switch contype */}}{{end}}
    {{- /* range constraints */}}{{end}}
)
{{/* range tables */}}{{end}}
