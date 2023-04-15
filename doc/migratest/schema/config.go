package schema

// Settings настройки для конкретной схемы.
type Settings struct {
	// SchemaName логично что имя схемы.
	SchemaName string `json:"name,omitempty"`
	// Blacklist список таблиц, про которые знать не надо.
	Blacklist []string `json:"blacklist,omitempty"`
	// Blacklist список таблиц, про которые знать надо.
	Whitelist []string `json:"whitelist,omitempty"`
}

// Patterns работают как обычный glob.
type Patterns struct {
	Whitelist []string
	Blacklist []string
}
