package config

import (
	"fmt"
	"strings"

	"go.uber.org/fx"
)

type FileConfig struct {
	DBConn   string   `yaml:"dbconn"`
	Patterns []string `yaml:"parser"`
}

type DBConn string

type FxConfig struct {
	fx.Out

	DB     DBConn
	Parser Parser
}

type Parser struct {
	Patterns []ParserPattern
}

type ParserPattern struct {
	Schema string `yaml:"schema"`
	Tables string `yaml:"tables"`
}

func NewConfig(fc FileConfig) (FxConfig, error) {
	out := FxConfig{
		DB:     DBConn(fc.DBConn),
		Parser: Parser{},
	}
	patterns, err := parsePatterns(fc.Patterns)
	if err != nil {
		return out, err
	}
	out.Parser.Patterns = patterns

	return out, nil
}

func parsePatterns(patterns []string) ([]ParserPattern, error) {
	res := make([]ParserPattern, 0, len(patterns))
	for _, pattern := range patterns {
		// TODO unquote
		parts := strings.Split(pattern, ".")

		var p ParserPattern
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
