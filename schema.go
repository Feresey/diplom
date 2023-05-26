package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/Feresey/mtest/db"
	"github.com/Feresey/mtest/parse"
	"github.com/Feresey/mtest/schema"
)

type SchemaLoaderFlags struct {
	dumpPath *cli.StringFlag
}

func NewSchemaLoaderFlags() SchemaLoaderFlags {
	return SchemaLoaderFlags{
		dumpPath: &cli.StringFlag{
			Name:    "input",
			Aliases: []string{"i"},
			Usage:   "-i schema.json",
			Action: func(ctx *cli.Context, fpath string) error {
				fileInfo, err := os.Stat(fpath)
				if os.IsNotExist(err) {
					return xerrors.Errorf("dump file %q does not exist", fpath)
				}
				if fileInfo.IsDir() {
					return xerrors.Errorf("%q is a directory, expected file", fpath)
				}
				if fileInfo.Mode().IsRegular() && fileInfo.Mode().Perm()&(1<<2) != 0 {
					return nil
				}
				return xerrors.Errorf("file %s exists but is not readable", fpath)
			},
		},
	}
}

type SchemaLoader struct {
	BaseCommand

	conn *pgx.Conn
}

func NewSchemaLoader(
	ctx *cli.Context,
	base BaseCommand,
	flags flags,
	sflags SchemaLoaderFlags,
) (SchemaLoader, error) {
	s := SchemaLoader{
		BaseCommand: base,
	}

	err := s.Init(ctx, flags, sflags)
	return s, err
}

const stdinFileName = "-"

func (p *SchemaLoader) Init(ctx *cli.Context, flags flags, sflags SchemaLoaderFlags) error {
	if sflags.dumpPath.Get(ctx) == "" {
		conn, err := p.connectDB(ctx, flags.debug.Get(ctx))
		if err != nil {
			return cli.Exit(err, 3)
		}
		p.conn = conn
	}
	return nil
}

func (p *SchemaLoader) Cleanup(ctx *cli.Context) error {
	if p.conn == nil {
		return nil
	}
	if err := p.conn.Close(ctx.Context); err != nil {
		return xerrors.Errorf("close pgx conn: %w", err)
	}
	return nil
}

func (p *SchemaLoader) GetSchema(
	ctx *cli.Context,
	sflags SchemaLoaderFlags,
) (s *schema.Schema, err error) {
	if filename := sflags.dumpPath.Get(ctx); filename != "" {
		return p.getSchemaFromFile(filename)
	}
	p.log.Info("schema dump path is not specified")
	return p.parseDB(ctx)
}

func (p *SchemaLoader) getSchemaFromFile(filename string) (s *schema.Schema, err error) {
	p.log.Debug("load schema from file", zap.String("filename", filename))
	defer p.log.Info("schema loaded", zap.Error(err), zap.String("filename", filename))
	var in io.Reader
	if filename == stdinFileName {
		in = os.Stdin
	} else {
		fileData, err := os.ReadFile(filename)
		if err != nil {
			return nil, xerrors.Errorf("read schema dump file: %w", err)
		}
		in = bytes.NewReader(fileData)
	}
	if err := json.NewDecoder(in).Decode(&s); err != nil {
		return nil, xerrors.Errorf("decode schema: %w", err)
	}
	return s, nil
}

func (p *SchemaLoader) parseDB(ctx *cli.Context) (s *schema.Schema, err error) {
	if p.conn == nil {
		p.log.Fatal("connection is nil")
	}
	p.log.Debug("parse schema")
	defer p.log.Info("schema parsed", zap.Error(err))

	parser := parse.NewParser(p.conn, p.log)
	s, err = parser.LoadSchema(ctx.Context, p.cnf.Parser)
	if err != nil {
		var pErr db.Error
		if errors.As(err, &pErr) {
			p.log.Error(pErr.Pretty())
		}
		return nil, xerrors.Errorf("parse schema: %w", err)
	}

	return s, nil
}
