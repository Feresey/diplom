package driver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Feresey/diplom/migratest/schema"
)

type UserInfo struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
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

func NewDefaultSchemaConfig() *schema.Config {
	return &schema.Config{
		Patterns: schema.SchemaPatterns{
			Whitelist: []string{"public"},
			Blacklist: []string{"pg_*", "information_schema"},
		},
		ConcreteConfig: []schema.SchemaSettings{
			{SchemaName: "public"},
		},
	}
}
