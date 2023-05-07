package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Feresey/mtest/parse"
	"github.com/Feresey/mtest/parse/queries"
	"github.com/Feresey/mtest/schema"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func DumpCommand(f flags) *cli.Command {
	dumpPath := &cli.StringFlag{
		Name:     "dump",
		Value:    "dump",
		Required: true,
		Aliases:  []string{"d"},
	}

	return &cli.Command{
		Name:        "parse",
		Description: "parse schema",
		Flags: append(f.Set(),
			dumpPath,
		),
		Action: func(ctx *cli.Context) error {
			log, err := newLogger(f.Debug.Get(ctx))
			if err != nil {
				return err
			}
			cnf, err := ReadConfig(f.Config.Get(ctx))
			if err != nil {
				return fmt.Errorf("read config: %w", err)
			}
			conf, err := NewFxConfig(cnf, f.Debug.Get(ctx))
			if err != nil {
				return err
			}

			if err := createDirIfNotExist(dumpPath.Get(ctx)); err != nil {
				return fmt.Errorf("create dump dir: %w", err)
			}

			var (
				parser       *parse.Parser
				parserConfig parse.Config
			)

			app := NewApp(ctx, log, conf, f, fx.Populate(&log, &parser, &parserConfig))
			return RunApp(app, log, func(runCtx context.Context) error {
				return parseSchema(runCtx, log, dumpPath.Get(ctx), parser, conf.Parser)
			})
		},
	}
}

func createDirIfNotExist(path string) error {
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

func parseSchema(
	ctx context.Context,
	log *zap.Logger,
	dumpPath string,
	parser *parse.Parser,
	pc parse.Config,
) error {
	s, err := parser.LoadSchema(ctx, pc)
	if err != nil {
		log.Error(err.Error())
		var pErr queries.Error
		if errors.As(err, &pErr) {
			log.Error(pErr.Pretty())
			// TODO экзит коды
			return nil
		}

		return fmt.Errorf("error loading schema: %w", err)
	}
	g := schema.NewGraph(s)

	slog := log.Sugar()

	schemaDumpPath := filepath.Join(dumpPath, "schema.sql")
	slog.Infof("dump schema to %q", schemaDumpPath)
	if err := dumpToFile(schemaDumpPath, g, schema.DumpSchemaTemplate); err != nil {
		return fmt.Errorf("failed to dump schema: %w", err)
	}

	typesDumpPath := filepath.Join(dumpPath, "types.txt")
	slog.Infof("dump types to %q", typesDumpPath)
	if err := dumpToFile(typesDumpPath, g, schema.DumpTypesTemplate); err != nil {
		return fmt.Errorf("failed to dump types: %w", err)
	}

	graphDumpPath := filepath.Join(dumpPath, "graph.puml")
	slog.Infof("dump graph to %q", graphDumpPath)
	if err := dumpToFile(graphDumpPath, g, schema.DumpGrapthTemplate); err != nil {
		return fmt.Errorf("failed to dump grapth: %w", err)
	}

	return nil
}

func dumpToFile(fileName string, g *schema.Graph, tpl schema.TemplateName) (err error) {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("create output file for dump: %w", err)
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	return g.Dump(file, tpl)
}
