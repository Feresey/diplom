package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/Feresey/mtest/generate"
	"github.com/Feresey/mtest/parse"
	"github.com/Feresey/mtest/schema"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
)

type generateFlags struct {
	flags
	schemaDumpPath *cli.StringFlag
	outputPath     *cli.StringFlag
}

func (f *generateFlags) Set() []cli.Flag {
	return append(f.flags.Set(),
		f.schemaDumpPath,
		f.outputPath,
	)
}

func GenerateCommand(f flags) *cli.Command {
	gf := generateFlags{
		flags: f,
		schemaDumpPath: &cli.StringFlag{
			Name:    "input",
			Aliases: []string{"i"},
		},
		outputPath: &cli.StringFlag{
			Name:        "output",
			DefaultText: "stdout",
			Usage:       "-o outdir",
			Aliases:     []string{"o"},
		},
	}

	return &cli.Command{
		Name:        "generate",
		Description: "generate records",
		Flags:       f.Set(),
		// TODO
		Action: nil,
		Subcommands: []*cli.Command{
			generateDefaultCommand(gf),
		},
	}
}

func generateDefaultCommand(f generateFlags) *cli.Command {
	return &cli.Command{
		Name:        "default",
		Description: "generate default partial records",
		Flags:       f.Set(),
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

			var (
				parser       *parse.Parser
				parserConfig parse.Config
			)

			app := NewApp(ctx, log, conf, f.flags, fx.Populate(&log, &parser, &parserConfig))
			return RunApp(ctx.Context, app, log, func(runCtx context.Context) error {
				var graph *schema.Graph
				dump := f.schemaDumpPath.Get(ctx)
				if dump == "" {
					g, err := parseSchema(runCtx, log, parser, conf.Parser)
					if err != nil {
						return err
					}
					graph = g
				} else {
					// TODO load parsed schema
				}

				// var out io.Writer = os.Stdout
				// if output := f.outputPath.Get(ctx); output != "" {
				// 	fileOut, err := os.Create(output)
				// 	if err != nil {
				// 		return err
				// 	}
				// 	defer fileOut.Close()
				// 	out = fileOut
				// }

				gen, err := generate.New(log, graph)
				if err != nil {
					return err
				}

				checks := gen.GetDefaultChecks()

				return dumpChecksCSV(checks)
			})
		},
	}
}

func dumpChecksCSV(checks map[string]generate.PartialRecords) error {
	for tableName, tableChecks := range checks {
		fmt.Println("\n\nDUMP TABLE ", tableName)
		cw := csv.NewWriter(os.Stdout)
		for _, record := range tableChecks {
			if err := cw.Write(record.Columns); err != nil {
				return err
			}
			if err := cw.Write(record.Values); err != nil {
				return err
			}
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
			return err
		}
		fmt.Println("\n\nDUMP TABLE FINISHED", tableName)
	}

	return nil
}
