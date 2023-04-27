Tables:
{{- range .TableNames }}
  {{ . }}:
    Primary Key: {{ with index $.Tables .PrimaryKey.Name.Name }}{{ .Columns | columnNames }}{{ end }}
    Foreign Keys:
    {{- range $fkName, $fk := .ForeignKeys }}
      {{ $fkName }}:
        Table: {{ $fk.Uniq.Name.Schema }}.{{ $fk.Uniq.Name.Name }}
        Columns: {{ $fk.Uniq.Columns | columnNames }}
    {{- end }}
    Referenced By:
    {{- range $refName, $ref := .ReferencedBy }}
      {{ $refName }}:
        Table: {{ $ref.Table.Name.Schema }}.{{ $ref.Table.Name.Name }}
        Columns: {{ $ref.Columns | columnNames }}
    {{- end }}
    Constraints:
    {{- range $cName, $c := .Constraints }}
      {{ $cName }}:
        Type: {{ $c.Type.String }}
        Columns: {{ $c.Columns | columnNames }}
    {{- end }}
{{- end }}

Constraints:
{{- range .ConstraintNames }}
  {{ . }}:
    Type: {{ index $.Constraints .Type.String }}
    Table: {{ .Table.Name.Schema }}.{{ .Table.Name.Name }}
    Columns: {{ .Columns | columnNames }}
{{- end }}