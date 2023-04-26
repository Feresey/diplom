package config

type DBConfig string

type Config struct {
	DBConfig DBConfig `yaml:"db"`
}
