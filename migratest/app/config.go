package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/Feresey/diplom/migratest/schema"
	"github.com/Feresey/diplom/migratest/schema/driver"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/fx"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Migrations MigratorConfig `json:"migrations"`
	DB         driver.Config  `json:"db"`
	Schema     schema.Config  `json:"schema"`
}

type AppConfig struct {
	fx.Out

	Migrations MigratorConfig
	DB         driver.Config
	Schema     schema.Config
}

type Flags struct {
	OutFile       string
	MigrationsDir string
	ConfigPath    string
}

func ParseFlags() *Flags {
	var flags Flags
	flag.StringVar(&flags.ConfigPath, "c", "config.yaml", "path to config file")
	flag.StringVar(&flags.OutFile, "o", "schema.json", "path to output schema file")
	flag.StringVar(&flags.MigrationsDir, "m", ".", "path to migrations folder")
	flag.Parse()
	return &flags
}

func GetConfig(flags *Flags) (*AppConfig, error) {
	c := &Config{
		DB:     *driver.NewDefaultConfig(),
		Schema: *driver.NewDefaultSchemaConfig(),
		Migrations: MigratorConfig{
			Path: "./migrations",
		},
	}

	configBytes, err := os.ReadFile(flags.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}

	err = yaml.Unmarshal(configBytes, c)
	if err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	spew.Dump(c)
	return &AppConfig{
		DB:         c.DB,
		Schema:     c.Schema,
		Migrations: c.Migrations,
	}, nil
}
