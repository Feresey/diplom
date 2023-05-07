package main

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/fx"
	"gopkg.in/yaml.v3"

	"github.com/Feresey/mtest/db"
	"github.com/Feresey/mtest/parse"
)

type FileConfig struct {
	DBConn   string   `yaml:"dbconn"`
	Patterns []string `yaml:"parser"`
}

func ReadConfig(confPath string) (FileConfig, error) {
	var fc FileConfig
	file, err := os.ReadFile(confPath)
	if err != nil {
		return fc, err
	}

	err = yaml.Unmarshal(file, &fc)
	return fc, err
}

type FxConfig struct {
	fx.Out

	DB     db.Config
	Parser parse.Config
}

func NewFxConfig(
	fc FileConfig,
	debug bool,
) (FxConfig, error) {
	out := FxConfig{
		DB: db.Config{
			Conn:  fc.DBConn,
			Debug: debug,
		},
		Parser: parse.Config{},
	}
	patterns, err := parsePatterns(fc.Patterns)
	if err != nil {
		return out, err
	}
	out.Parser.Patterns = patterns

	return out, nil
}

func parsePatterns(
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
			return nil, fmt.Errorf("wrong pattern: %q", pattern)
		}
		res = append(res, p)
	}

	return res, nil
}
