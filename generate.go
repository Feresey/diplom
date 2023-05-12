package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/Feresey/mtest/generate"
)

type generateFlags struct {
	flags
	schema     SchemaLoaderFlags
	outputPath *cli.StringFlag
}

func (f generateFlags) Set() []cli.Flag {
	return append(
		f.flags.Set(),
		f.outputPath,
		f.schema.dumpPath,
	)
}

type GenerateCommand struct {
	flags generateFlags
	BaseCommand

	schemaLoader SchemaLoader
}

func NewGenerateCommand(f flags) *GenerateCommand {
	return &GenerateCommand{
		flags: generateFlags{
			flags: f,
			outputPath: &cli.StringFlag{
				Name: "output",
				Action: func(ctx *cli.Context, dirname string) error {
					// Проверяем, существует ли директория
					info, err := os.Stat(dirname)
					if err != nil {
						return fmt.Errorf("output path does not exist: %q", dirname)
					}

					// Проверяем, является ли это директорией
					if !info.IsDir() {
						return fmt.Errorf("output path is not a directory: %q", dirname)
					}
					return nil
				},
				Usage:   "-o outdir",
				Aliases: []string{"o"},
			},
			schema: NewSchemaLoaderFlags(),
		},
	}
}

func (p *GenerateCommand) Command() *cli.Command {
	return &cli.Command{
		Name:        "generate",
		Description: "generate records",
		Flags:       p.flags.Set(),
		Before:      p.Init,
		Action:      p.GenerateRecords,
		Subcommands: []*cli.Command{
			p.DefaultsCommand(),
		},
	}
}

func (p *GenerateCommand) Init(ctx *cli.Context) error {
	base, err := NewBase(ctx, p.flags.flags)
	if err != nil {
		return cli.Exit(err, 2)
	}
	p.BaseCommand = base
	loader, err := NewSchemaLoader(ctx, base, p.flags.flags, p.flags.schema)
	if err != nil {
		return err
	}
	p.schemaLoader = loader

	return nil
}

func (p *GenerateCommand) GenerateRecords(ctx *cli.Context) error {
	s, err := p.schemaLoader.GetSchema(ctx, p.flags.schema)
	if err != nil {
		return err
	}

	gen, err := generate.New(p.log, s.Tables)
	if err != nil {
		return err
	}

	records, err := gen.GenerateRecords(nil, nil)
	if err != nil {
		return fmt.Errorf("generate records: %w", err)
	}
	return p.DumpRecords(records, p.flags.outputPath.Get(ctx))
}

func (p *GenerateCommand) DefaultsCommand() *cli.Command {
	var tableRegs []*regexp.Regexp
	tablesPatterns := &cli.StringSliceFlag{
		Name:    "pattern",
		Aliases: []string{"p"},
		Usage:   `-p 'my_schema\.my_table.*' -p 'other_schema\..*' -p '.*'`,
		Action: func(ctx *cli.Context, names []string) error {
			for _, name := range names {
				re, err := regexp.Compile(name)
				if err != nil {
					return err
				}
				tableRegs = append(tableRegs, re)
			}
			return nil
		},
	}
	tablesNames := &cli.StringSliceFlag{
		Name:    "name",
		Aliases: []string{"n"},
		Usage:   `-n my_schema.my_table -n other_schema.other_table`,
	}
	return &cli.Command{
		Name:        "default",
		Description: "generate default partial records",
		Flags: append(
			p.flags.Set(),
			tablesNames,
			tablesPatterns,
		),
		Action: func(ctx *cli.Context) error {
			s, err := p.schemaLoader.GetSchema(ctx, p.flags.schema)
			if err != nil {
				return err
			}

			tables := mapset.NewThreadUnsafeSet[int]()

			for _, re := range tableRegs {
				for _, table := range s.Tables {
					if re.MatchString(table.String()) {
						tables.Add(table.OID())
					}
				}
			}
			for _, name := range tablesNames.Get(ctx) {
				for _, table := range s.Tables {
					if strings.EqualFold(name, table.String()) {
						tables.Add(table.OID())
					}
				}
			}
			p.log.Debug("got table oids", zap.Ints("tables", tables.ToSlice()))

			gen, err := generate.New(p.log, s.Tables)
			if err != nil {
				return fmt.Errorf("create generator: %w", err)
			}

			checks, err := gen.GetDefaultChecks(tables.ToSlice())
			if err != nil {
				return fmt.Errorf("generate default checks: %w", err)
			}

			return p.DumpPartial(checks, p.flags.outputPath.Get(ctx))
		},
	}
}

func (p *GenerateCommand) DumpRecords(records map[int]generate.Records, dumpdir string) error {
	for _, tableRecords := range records {
		err := dumpToFile(p.log, dumpdir, tableRecords.Table.String(), tableRecords, p.dumpRecordsCSV)
		if err != nil {
			err = fmt.Errorf("dump partial records for table %q: %w", tableRecords.Table, err)
			p.log.Error(err.Error())
			return err
		}
	}

	return nil
}

func (p *GenerateCommand) DumpPartial(checks []generate.PartialRecords, dumpdir string) error {
	for _, partial := range checks {
		err := dumpToFile(p.log, dumpdir, partial.Table.String(), partial, p.dumpPartialCSV)
		if err != nil {
			err = fmt.Errorf("dump partial records for table %q: %w", partial.Table, err)
			p.log.Error(err.Error())
			return err
		}
	}
	return nil
}

func (p *GenerateCommand) dumpPartialCSV(w io.Writer, precords generate.PartialRecords) error {
	var conv CSVConverter
	return csv.NewWriter(w).WriteAll(conv.ConvertPartialRecords(precords))
}

func (p *GenerateCommand) dumpRecordsCSV(w io.Writer, records generate.Records) error {
	var conv CSVConverter
	return csv.NewWriter(w).WriteAll(conv.ConvertRecords(records))
}

func dumpToFile[T any](
	log *zap.Logger,
	dumpdir string,
	tableName string,
	data T,
	dumpFunc func(w io.Writer, data T) error,
) (err error) {
	log = log.WithOptions(zap.AddCallerSkip(1))
	dumpfile := filepath.Join(dumpdir, tableName+".csv")
	if dumpdir == "" {
		dumpfile = "stdout"
	}
	defer func() {
		log.Info("dumped partial records for table",
			zap.String("table", tableName),
			zap.String("dumpfile", dumpfile),
			zap.Error(err),
		)
	}()

	var out io.Writer
	if dumpdir == "" {
		out = os.Stdout
	} else {
		file, err := os.Create(dumpfile)
		if err != nil {
			err := fmt.Errorf("create output file for default checks: %w", err)
			log.Error(err.Error())
			return cli.Exit("", 5)
		}
		defer func() {
			err = errors.Join(err, file.Close())
		}()
		out = file
	}

	return dumpFunc(out, data)
}
