package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/xerrors"

	"github.com/Feresey/mtest/db"
	"github.com/jackc/pgx/v5"
	"github.com/urfave/cli/v2"
)

func newLogger(debug bool) (*zap.Logger, error) {
	lc := zap.NewDevelopmentConfig()
	lc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	lc.DisableStacktrace = true
	if debug {
		lc.Level.SetLevel(zap.DebugLevel)
	} else {
		lc.Level.SetLevel(zap.InfoLevel)
	}
	return lc.Build()
}

type flags struct {
	configPath *cli.StringFlag
	debug      *cli.BoolFlag
}

func (f *flags) Set() []cli.Flag {
	return []cli.Flag{
		f.configPath,
		f.debug,
	}
}

func main() {
	f := flags{
		configPath: &cli.StringFlag{
			Name:      "config",
			Value:     "mtest.yml",
			Usage:     "config file path",
			TakesFile: true,
			Aliases:   []string{"c"},
		},
		debug: &cli.BoolFlag{
			Name:   "debug",
			Value:  false,
			Usage:  "show debug information",
			Hidden: true,
		},
	}

	app := &cli.App{
		Name:        "mtest",
		Description: "migration test tool",
		Flags:       f.Set(),
		Commands: []*cli.Command{
			NewParseCommand(f).Command(),
			NewGenerateCommand(f).Command(),
		},
		ExitErrHandler: func(ctx *cli.Context, err error) {
			if err == nil {
				return
			}
			if f.debug.Get(ctx) {
				fmt.Printf("%+v\n", err)
			} else {
				fmt.Printf("%v\n", err)
			}
			os.Exit(1)
		},
		EnableBashCompletion: true,
	}
	if err := app.Run(os.Args); err != nil {
		println(err.Error())
		os.Exit(2)
	}
}

type BaseCommand struct {
	log *zap.Logger
	cnf *AppConfig
}

func NewBase(ctx *cli.Context, f flags) (BaseCommand, error) {
	var empty BaseCommand
	log, err := newLogger(f.debug.Get(ctx))
	if err != nil {
		return empty, xerrors.Errorf("create logger: %w", err)
	}
	zap.ReplaceGlobals(log)
	cnf, err := ReadConfig(f.configPath.Get(ctx))
	if err != nil {
		return empty, xerrors.Errorf("get config: %w", err)
	}
	log.Debug("config readed")

	return BaseCommand{
		log: log,
		cnf: cnf,
	}, nil
}

func (b *BaseCommand) connectDB(ctx *cli.Context, debug bool) (*pgx.Conn, error) {
	if debug {
		b.cnf.DB.SetDebug(true)
	}
	conn, err := db.NewDB(ctx.Context, b.log, b.cnf.DB)
	if err != nil {
		return nil, xerrors.Errorf("create database connection: %w", err)
	}
	b.log.Debug("connected to database")

	return conn, nil
}
