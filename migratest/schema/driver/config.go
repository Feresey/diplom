package driver

import (
	"fmt"
	"strconv"
	"strings"
)

type UserInfo struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type SchemaConfig struct {
	SchemaName string   `json:"schema_name,omitempty"`
	Blacklist  []string `json:"blacklist,omitempty"`
	Whitelist  []string `json:"whitelist,omitempty"`
}

type Config struct {
	Credentials UserInfo `json:"credentials,omitempty"`
	Host        string   `json:"host,omitempty"`
	Port        int      `json:"port,omitempty"`
	DBName      string   `json:"db-name,omitempty"`

	Params map[string]string `json:"params,omitempty"`
}

// user=jack password=secret host=pg.example.com port=5432 dbname=mydb sslmode=verify-ca pool_max_conns=10
func (c *Config) FormatDSN() string {
	formatDSNItem := func(name, value string) string {
		return fmt.Sprintf("%s=%s", name, value)
	}
	values := []string{
		formatDSNItem("host", c.Host),
		formatDSNItem("user", c.Credentials.Username),
		formatDSNItem("password", c.Credentials.Password),
	}

	if c.Port != 0 {
		values = append(values, formatDSNItem("port", strconv.Itoa(c.Port)))
	}
	if c.DBName != "" {
		values = append(values, formatDSNItem("dbname", c.DBName))
	}
	for key, value := range c.Params {
		values = append(values, formatDSNItem(key, value))
	}

	return strings.Join(values, " ")
}

func NewDefaultConfig() *Config {
	return &Config{
		Credentials: UserInfo{
			Username: "postgres",
			Password: "pass",
		},
		Host:   "0.0.0.0",
		Port:   5432,
		DBName: "test",
		Params: map[string]string{
			"sslmode": "disable",
		},
	}
}

func NewDefaultSchemaConfig() *SchemaConfig {
	return &SchemaConfig{
		SchemaName: "public",
	}
}
