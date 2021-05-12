package schema

// SchemaSettings настройки для конкретной схемы.
type SchemaSettings struct {
	// SchemaName логично что имя схемы.
	SchemaName string `json:"name,omitempty"`
	// Blacklist список таблиц, про которые знать не надо.
	Blacklist []string `json:"blacklist,omitempty"`
	// Blacklist список таблиц, про которые знать надо.
	Whitelist []string `json:"whitelist,omitempty"`
}

// SchemaPatterns работают как обычный glob.
type SchemaPatterns struct {
	Whitelist []string
	Blacklist []string
}
