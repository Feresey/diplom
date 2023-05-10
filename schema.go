package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v2"

	"github.com/Feresey/mtest/parse"
	"github.com/Feresey/mtest/parse/queries"
	"github.com/Feresey/mtest/schema"
)

type schemaLoaderFlags struct {
	dumpPath *cli.StringFlag
}

func NewSchemaLoaderFlags() schemaLoaderFlags {
	return schemaLoaderFlags{
		dumpPath: &cli.StringFlag{
			Name:    "input",
			Aliases: []string{"i"},
			Usage:   "-i schema.json",
			Action: func(ctx *cli.Context, fpath string) error {
				fileInfo, err := os.Stat(fpath)
				if os.IsNotExist(err) {
					return fmt.Errorf("file %s does not exist", fpath)
				}
				if fileInfo.IsDir() {
					return fmt.Errorf("%q is a directory, expected file", fpath)
				}
				if fileInfo.Mode().IsRegular() && fileInfo.Mode().Perm()&(1<<2) != 0 {
					return nil
				} else {
					return fmt.Errorf("file %s exists but is not readable", fpath)
				}
			},
		},
	}
}

type schemaLoader struct {
	baseCommand

	conn *pgx.Conn
}

func NewSchemaLoader(base baseCommand) schemaLoader {
	return schemaLoader{
		baseCommand: base,
	}
}

func (p *schemaLoader) Init(ctx *cli.Context, flags flags, filename string) error {
	if filename == "" {
		conn, err := p.connectDB(ctx, flags.debug.Get(ctx))
		if err != nil {
			return cli.Exit(err, 3)
		}
		p.conn = conn
	}
	return nil
}

func (p *schemaLoader) Cleanup(ctx *cli.Context) error {
	if p.conn == nil {
		return nil
	}
	if err := p.conn.Close(ctx.Context); err != nil {
		return fmt.Errorf("close pgx conn: %w", err)
	}
	return nil
}

func (p *schemaLoader) GetSchema(ctx *cli.Context, filename string) (s *schema.Schema, err error) {
	if filename != "" {
		return p.getSchemaFromFile(filename)
	} else {
		return p.parseDB(ctx)
	}
}

func (p *schemaLoader) getSchemaFromFile(schemaInput string) (s *schema.Schema, err error) {
	var in io.Reader
	if schemaInput != "-" {
		in = os.Stdin
	} else {
		fileData, err := os.ReadFile(schemaInput)
		if err != nil {
			return nil, fmt.Errorf("read schema dump file: %w", err)
		}
		in = bytes.NewReader(fileData)
	}
	if err := json.NewDecoder(in).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode schema: %w", err)
	}
	p.log.Info("schema loaded")
	return s, nil
}

func (p *schemaLoader) parseDB(ctx *cli.Context) (s *schema.Schema, err error) {
	parser := parse.NewParser(p.conn, p.log)
	s, err = parser.LoadSchema(ctx.Context, p.cnf.Parser)
	if err != nil {
		var pErr queries.Error
		if errors.As(err, &pErr) {
			p.log.Error(pErr.Pretty())
		}
		return nil, fmt.Errorf("parse schema: %w", err)
	}
	p.log.Info("schema parsed")

	return s, nil
}
