package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

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
		Flags:       gf.Set(),
		Action: func(ctx *cli.Context) error {
			log, err := newLogger(gf.Debug.Get(ctx))
			if err != nil {
				return err
			}
			cnf, err := ReadConfig(gf.Config.Get(ctx))
			if err != nil {
				return fmt.Errorf("read config: %w", err)
			}
			conf, err := NewFxConfig(cnf, gf.Debug.Get(ctx))
			if err != nil {
				return err
			}

			var (
				parser       *parse.Parser
				parserConfig parse.Config
			)

			app := NewApp(ctx, log, conf, gf.flags, fx.Populate(&log, &parser, &parserConfig))
			return RunApp(ctx.Context, app, log, func(runCtx context.Context) error {
				var graph *schema.Graph
				dump := gf.schemaDumpPath.Get(ctx)
				if dump == "" {
					g, err := parseSchema(runCtx, log, parser, conf.Parser)
					if err != nil {
						return err
					}
					graph = g
					// } else {
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

				records, err := gen.GenerateRecords(nil, nil)
				if err != nil {
					return err
				}
				return dumpRecordsCSV(records)
			})
		},
		Subcommands: []*cli.Command{
			generateDefaultCommand(gf),
		},
	}
}

func generateDefaultCommand(gf generateFlags) *cli.Command {
	return &cli.Command{
		Name:        "default",
		Description: "generate default partial records",
		Flags:       gf.Set(),
		Action: func(ctx *cli.Context) error {
			log, err := newLogger(gf.Debug.Get(ctx))
			if err != nil {
				return err
			}
			cnf, err := ReadConfig(gf.Config.Get(ctx))
			if err != nil {
				return fmt.Errorf("read config: %w", err)
			}
			conf, err := NewFxConfig(cnf, gf.Debug.Get(ctx))
			if err != nil {
				return err
			}

			var (
				parser       *parse.Parser
				parserConfig parse.Config
			)

			app := NewApp(ctx, log, conf, gf.flags, fx.Populate(&log, &parser, &parserConfig))
			return RunApp(ctx.Context, app, log, func(runCtx context.Context) error {
				var graph *schema.Graph
				dump := gf.schemaDumpPath.Get(ctx)
				if dump == "" {
					g, err := parseSchema(runCtx, log, parser, conf.Parser)
					if err != nil {
						return err
					}
					graph = g
					// } else {
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
		var lastColumns string

		fmt.Println("\n\nDUMP TABLE ", tableName)
		cw := csv.NewWriter(os.Stdout)
		for _, record := range tableChecks {
			if currColumns := strings.Join(record.Columns, ","); currColumns != lastColumns {
				lastColumns = currColumns
				if err := cw.Write(record.Columns); err != nil {
					return err
				}
			}
			if err := cw.Write(record.Values); err != nil {
				return err
			}
		}
		cw.Flush()
		if err := cw.Error(); err != nil {
			return err
		}
		fmt.Println("DUMP TABLE FINISHED", tableName)
	}

	return nil
}

func dumpRecordsCSV(records map[string]*generate.Records) error {
	for tableName, tableRecords := range records {
		fmt.Println("\n\nDUMP TABLE ", tableName)
		cw := csv.NewWriter(os.Stdout)
		if err := cw.Write(tableRecords.Columns); err != nil {
			return err
		}
		if err := cw.WriteAll(tableRecords.Values); err != nil {
			return err
		}
		fmt.Println("DUMP TABLE FINISHED", tableName)
	}

	return nil
}
