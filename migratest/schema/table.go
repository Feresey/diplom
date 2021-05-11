package schema

import "fmt"

// Table metadata from the database schema.
type Table struct {
	Name string `json:"name"`
	// For dbs with real schemas, like Postgres.
	// Example value: "schema_name"."table_name"
	SchemaName string   `json:"schema_name"`
	Columns    []Column `json:"columns"`

	PKey  *PrimaryKey  `json:"p_key"`
	FKeys []ForeignKey `json:"f_keys"`

	IsJoinTable bool `json:"is_join_table"`

	ToOneRelationships  []ToOneRelationship  `json:"to_one_relationships"`
	ToManyRelationships []ToManyRelationship `json:"to_many_relationships"`
}

// GetTable by name. Panics if not found (for use in templates mostly).
func GetTable(tables []Table, name string) (tbl Table) {
	for _, t := range tables {
		if t.Name == name {
			return t
		}
	}

	panic(fmt.Sprintf("could not find table name: %s", name))
}

// GetColumn by name. Panics if not found (for use in templates mostly).
func (t Table) GetColumn(name string) (col Column) {
	for _, c := range t.Columns {
		if c.Name == name {
			return c
		}
	}

	panic(fmt.Sprintf("could not find column name: %s", name))
}
