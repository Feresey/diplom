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
	"go.uber.org/zap"

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

func NewSchemaLoader(
	ctx *cli.Context,
	base baseCommand,
	flags flags,
	sflags schemaLoaderFlags,
) (schemaLoader, error) {
	s := schemaLoader{
		baseCommand: base,
	}

	err := s.Init(ctx, flags, sflags)
	return s, err
}

func (p *schemaLoader) Init(ctx *cli.Context, flags flags, sflags schemaLoaderFlags) error {
	if sflags.dumpPath.Get(ctx) == "" {
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

func (p *schemaLoader) GetSchema(
	ctx *cli.Context,
	sflags schemaLoaderFlags,
) (s *schema.Schema, err error) {
	if filename := sflags.dumpPath.Get(ctx); filename != "" {
		return p.getSchemaFromFile(filename)
	} else {
		return p.parseDB(ctx)
	}
}

func (p *schemaLoader) getSchemaFromFile(filename string) (s *schema.Schema, err error) {
	p.log.Debug("load schema from file", zap.String("filename", filename))
	defer p.log.Info("schema loaded", zap.Error(err))
	var in io.Reader
	if filename == "-" {
		in = os.Stdin
	} else {
		fileData, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("read schema dump file: %w", err)
		}
		in = bytes.NewReader(fileData)
	}
	if err := json.NewDecoder(in).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode schema: %w", err)
	}
	return s, nil
}

func (p *schemaLoader) parseDB(ctx *cli.Context) (s *schema.Schema, err error) {
	if p.conn == nil {
		p.log.Fatal("connection is nil")
	}
	p.log.Debug("parse schema")
	defer p.log.Info("schema parsed", zap.Error(err))

	parser := parse.NewParser(p.conn, p.log)
	s, err = parser.LoadSchema(ctx.Context, p.cnf.Parser)
	if err != nil {
		var pErr queries.Error
		if errors.As(err, &pErr) {
			p.log.Error(pErr.Pretty())
		}
		return nil, fmt.Errorf("parse schema: %w", err)
	}

	return s, nil
}
