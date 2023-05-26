package main

import (
	"os"
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/Feresey/mtest/db"
	"github.com/Feresey/mtest/parse"
)

type FileConfig struct {
	DBConn   string   `yaml:"dbconn"`
	Patterns []string `yaml:"parser"`
}

type AppConfig struct {
	DB     db.Config
	Parser parse.Config
}

func (fc FileConfig) Build() (*AppConfig, error) {
	patterns, err := fc.parsePatterns(fc.Patterns)
	if err != nil {
		return nil, xerrors.Errorf("parse patterns failed: %w", err)
	}
	return &AppConfig{
		DB: db.Config{
			Conn: fc.DBConn,
		},
		Parser: parse.Config{
			Patterns: patterns,
		},
	}, nil
}

func ReadConfig(confPath string) (*AppConfig, error) {
	var fc FileConfig
	file, err := os.ReadFile(confPath)
	if err != nil {
		return nil, xerrors.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(file, &fc); err != nil {
		return nil, xerrors.Errorf("parse config: %w", err)
	}

	c, err := fc.Build()
	if err != nil {
		return nil, xerrors.Errorf("process config data: %w", err)
	}
	return c, nil
}

func (fc FileConfig) parsePatterns(
	patterns []string,
) ([]parse.Pattern, error) {
	res := make([]parse.Pattern, 0, len(patterns))
	for _, pattern := range patterns {
		// TODO unquote
		parts := strings.Split(pattern, ".")

		var p parse.Pattern
		switch {
		case len(parts) == 1:
			p.Schema = parts[0]
		case len(parts) == 2:
			p.Schema = parts[0]
			p.Tables = parts[1]
		default:
			return nil, xerrors.Errorf("wrong pattern: %q", pattern)
		}
		res = append(res, p)
	}

	return res, nil
}
