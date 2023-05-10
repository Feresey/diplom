package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v2"

	"github.com/Feresey/mtest/parse"
	"github.com/Feresey/mtest/parse/queries"
	"github.com/Feresey/mtest/schema"
)

type parseFlags struct {
	flags
	outputPath *cli.StringFlag
}

func (pf *parseFlags) Set() []cli.Flag {
	return append(pf.flags.Set(),
		pf.outputPath,
	)
}

type parseCommand struct {
	pf parseFlags
	baseCommand

	conn *pgx.Conn
}

func NewParseCommand(f flags) *parseCommand {
	return &parseCommand{
		pf: parseFlags{
			flags: f,
			outputPath: &cli.StringFlag{
				Name:        "output",
				DefaultText: "dump",
				Required:    true,
				Aliases:     []string{"o"},
			},
		},
	}
}

func (p *parseCommand) Command() *cli.Command {
	return &cli.Command{
		Name:        "parse",
		Description: "parse schema",
		Flags:       p.pf.Set(),
		Before:      p.init,
		Action:      p.run,
		After:       p.cleanup,
	}
}

func (p *parseCommand) init(ctx *cli.Context) error {
	base, err := NewBase(ctx, p.pf.flags)
	if err != nil {
		return cli.Exit(err, 2)
	}
	conn, err := base.connectDB(ctx, p.pf.debug.Get(ctx))
	if err != nil {
		return cli.Exit(err, 3)
	}
	p.baseCommand = base
	p.conn = conn
	return nil
}

func (p *parseCommand) cleanup(ctx *cli.Context) error {
	if p.conn == nil {
		return nil
	}
	if err := p.conn.Close(ctx.Context); err != nil {
		return fmt.Errorf("close pgx conn: %w", err)
	}
	return nil
}

func (p *parseCommand) run(ctx *cli.Context) error {
	parser := parse.NewParser(p.conn, p.log)
	s, err := parser.LoadSchema(ctx.Context, p.cnf.Parser)
	if err != nil {
		var pErr queries.Error
		if errors.As(err, &pErr) {
			p.log.Error(pErr.Pretty())
		}
		return fmt.Errorf("parse schema: %w", err)
	}
	p.log.Info("schema parsed")

	g := schema.NewGraph(s)

	if _, err := g.TopologicalSort(); err != nil {
		return fmt.Errorf("try to determine tables order: %w", err)
	}

	outputPath := p.pf.outputPath.Get(ctx)

	return p.dump(g, outputPath)
}

func (p *parseCommand) dump(graph *schema.Graph, dumpPath string) error {
	slog := p.log.Sugar()

	if err := p.createDirIfNotExist(dumpPath); err != nil {
		return fmt.Errorf("create dump dir: %w", err)
	}

	schemaDumpPath := filepath.Join(dumpPath, "schema.sql")
	slog.Infof("dump sql to %q", schemaDumpPath)
	if err := p.dumpTemplate(schemaDumpPath, graph, schema.DumpSchemaTemplate); err != nil {
		return fmt.Errorf("failed to dump sql schema: %w", err)
	}

	graphDumpPath := filepath.Join(dumpPath, "graph.puml")
	slog.Infof("dump graph to %q", graphDumpPath)
	if err := p.dumpTemplate(graphDumpPath, graph, schema.DumpGrapthTemplate); err != nil {
		return fmt.Errorf("failed to dump grapth: %w", err)
	}

	jsonDumpPath := filepath.Join(dumpPath, "dump.json")
	slog.Infof("dump schema to %q", jsonDumpPath)
	if err := p.dumpToFile(graphDumpPath, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(graph.Schema)
	}); err != nil {
		return fmt.Errorf("failed to dump json schema: %w", err)
	}

	return nil
}

func (p *parseCommand) createDirIfNotExist(path string) error {
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Папка не существует, создаем ее
		err = os.MkdirAll(path, 0o755) //nolint:gomnd // dir mode
		if err != nil {
			return err
		}
	} else if !fileInfo.IsDir() {
		// Это не папка, возвращаем ошибку
		return &os.PathError{Op: "mkdir", Path: path, Err: os.ErrExist}
	}
	return nil
}

func (p *parseCommand) dumpTemplate(
	fileName string,
	g *schema.Graph, tpl schema.TemplateName,
) (err error) {
	return p.dumpToFile(fileName, func(w io.Writer) error {
		return g.Dump(w, tpl)
	})
}

func (p *parseCommand) dumpToFile(fileName string, f func(w io.Writer) error) (err error) {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("create output file for dump: %w", err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	return f(file)
}
