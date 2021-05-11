package schema

type SchemaSettings struct {
	SchemaName string   `json:"schema_name,omitempty"`
	Blacklist  []string `json:"blacklist,omitempty"`
	Whitelist  []string `json:"whitelist,omitempty"`
}

type SchemaPatterns struct {
	Whitelist []string
	Blacklist []string
}

type Config struct {
	Patterns       SchemaPatterns
	ConcreteConfig []SchemaSettings
}
