package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/Feresey/mtest/generate"
	"github.com/Feresey/mtest/schema"
)

type generateFlags struct {
	flags
	schema     schemaLoaderFlags
	outputPath *cli.StringFlag
}

type generateCommand struct {
	flags generateFlags
	baseCommand

	schemaLoader schemaLoader
}

func NewGenerateCommand(f flags) *generateCommand {
	return &generateCommand{
		flags: generateFlags{
			flags: f,
			outputPath: &cli.StringFlag{
				Name:     "output",
				Required: true,
				Usage:    "-o outdir",
				Aliases:  []string{"o"},
			},
			schema: NewSchemaLoaderFlags(),
		},
	}
}

func (p *generateCommand) Command() *cli.Command {
	return &cli.Command{
		Name:        "generate",
		Description: "generate records",
		Flags: append(
			p.flags.Set(),
			p.flags.outputPath,
			p.flags.schema.dumpPath,
		),
		Before: p.init,
		Action: p.GenerateRecords,
		Subcommands: []*cli.Command{
			p.DefaultsCommand(),
		},
	}
}

func (p *generateCommand) init(ctx *cli.Context) error {
	base, err := NewBase(ctx, p.flags.flags)
	if err != nil {
		return cli.Exit(err, 2)
	}
	p.baseCommand = base
	p.schemaLoader = NewSchemaLoader(base)

	return nil
}

func (p *generateCommand) GenerateRecords(ctx *cli.Context) error {
	s, err := p.schemaLoader.GetSchema(ctx, p.flags.schema.dumpPath.Get(ctx))
	if err != nil {
		return err
	}

	graph := schema.NewGraph(s)

	gen, err := generate.New(p.log, graph)
	if err != nil {
		return err
	}

	records, err := gen.GenerateRecords(nil, nil)
	if err != nil {
		return err
	}
	return dumpRecordsCSV(records)
}

func (p *generateCommand) DefaultsCommand() *cli.Command {
	defaultChecks := &cli.StringSliceFlag{
		Name:    "names",
		Aliases: []string{"n"},
		Usage:   `--names 'my_schema\.my_table.*,other_schema\..*,.*'`,
	}

	return &cli.Command{
		Name:        "default",
		Description: "generate default partial records",
		Flags: []cli.Flag{
			defaultChecks,
		},
		Action: func(ctx *cli.Context) error {
			s, err := p.schemaLoader.GetSchema(ctx, p.flags.schema.dumpPath.Get(ctx))
			if err != nil {
				return err
			}
			graph := schema.NewGraph(s)

			gen, err := generate.New(p.log, graph)
			if err != nil {
				return err
			}

			checks := gen.GetDefaultChecks()

			return dumpChecksCSV(checks)
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
