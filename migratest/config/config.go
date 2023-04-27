package config

import "go.uber.org/fx"

type DBConfig string

type Config struct {
	fx.Out

	DBConn DBConfig `yaml:"db"`
}
