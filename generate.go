package main

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/Feresey/mtest/generate"
	"github.com/Feresey/mtest/schema"
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
						return xerrors.Errorf("output path does not exist: %q", dirname)
					}

					// Проверяем, является ли это директорией
					if !info.IsDir() {
						return xerrors.Errorf("output path is not a directory: %q", dirname)
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

	gen, err := generate.New(p.log, s)
	if err != nil {
		return err
	}

	_ = gen

	// TODO load partial
	// TODO load domains
	// records, warnings := gen.GenerateRecords(nil, nil)
	// if len(warnings) > 0 {
	// 	p.log.Warn("generate records", zap.Errors("warnings", warnings))
	// }
	// for tableOID, records := range records {
	// 	table, ok := s.Tables[tableOID]
	// 	if !ok {
	// 		err := xerrors.Errorf("internal error: table with oid %d not found for generated records", tableOID)
	// 		p.log.Error(err.Error())
	// 		return err
	// 	}
	// 	err := p.DumpRecords(&table, records, p.flags.outputPath.Get(ctx))
	// 	if err != nil {
	// 		err = xerrors.Errorf("dump generated records: %w", err)
	// 		p.log.Error(err.Error())
	// 		return err
	// 	}
	// }

	return nil
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

			tables := make(map[string]schema.Table)

			for _, re := range tableRegs {
				for _, table := range s.Tables {
					if re.MatchString(table.String()) {
						tables[table.String()] = table
					}
				}
			}
			for _, name := range tablesNames.Get(ctx) {
				for _, table := range s.Tables {
					if strings.EqualFold(name, table.String()) {
						tables[table.String()] = table
					}
				}
			}
			p.log.Debug("got table oids",
				zap.Strings("tables", mapset.NewThreadUnsafeSetFromMapKeys(tables).ToSlice()))

			gen, err := generate.New(p.log, s)
			if err != nil {
				return xerrors.Errorf("create generator: %w", err)
			}

			for _, table := range tables {
				records := gen.GetDefaultChecks(table)
				err := p.DumpRecords(table, records, p.flags.outputPath.Get(ctx))
				if err != nil {
					err = xerrors.Errorf("dump default checks: %w", err)
					p.log.Error(err.Error())
					return err
				}
			}

			return nil
		},
	}
}

func (p *GenerateCommand) DumpRecords(
	table schema.Table,
	records generate.Records,
	dumpdir string,
) error {
	err := dumpToFile(
		p.log,
		dumpdir, table.String(),
		records,
		func(w io.Writer, records generate.Records) error {
			var conv CSVConverter
			return csv.NewWriter(w).WriteAll(conv.ConvertRecords(table, records))
		})
	if err != nil {
		err = xerrors.Errorf("dump partial records for table %q: %w", table, err)
		p.log.Error(err.Error())
		return err
	}

	return nil
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
			err := xerrors.Errorf("create output file for default checks: %w", err)
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
