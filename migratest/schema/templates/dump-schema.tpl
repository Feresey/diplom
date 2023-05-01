{{- /* range tables */}}
{{- range .TableNames }}
{{- $table := index $.Tables .}}
TABLE {{$table.Name}} (
{{- /* range columns */}}
{{- $maxlen := 0}}{{range $table.ColumnNames}}{{if gt (len .) $maxlen}}{{$maxlen = len .}}{{end}}{{end}}
{{- range $table.ColumnNames }}{{$column := index $table.Columns .}}
    {{$column.Name}} {{space (len $column.Name) $maxlen}}
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
        {{- if .NotNullable}}{{" " -}}
            NOT NULL
        {{- end}}
        {{- if .HasDefault}}{{" " -}}
            {{.Default}}
        {{- end}}
    {{- /* with attributes */}}
    {{- end}}
    {{- /* range foreign keys */}}
    {{- range $fk_id, $fk := $table.ForeignKeys}}
        {{- with index $fk.Foreign.Columns $column.Name}}
            {{- ""}} CONSTRAINT {{ $fk.Foreign.Name}} REFERENCES {{$fk.Uniq.Table.Name}}({{$fk.Uniq.Columns | columnNames}})
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
        {{- if .NullsNotDistinct}} NULLS NOT DISTINCT{{end}}
    {{- else if eq $t "Check" }}
        {{- ""}} {{.Definition}}
    {{- end}}
{{- end}}
);
{{/* range tables */}}{{end}}
