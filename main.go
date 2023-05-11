package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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
		EnableBashCompletion: true,
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

type baseCommand struct {
	log *zap.Logger
	cnf *AppConfig
}

func NewBase(ctx *cli.Context, f flags) (baseCommand, error) {
	var empty baseCommand
	log, err := newLogger(f.debug.Get(ctx))
	if err != nil {
		return empty, fmt.Errorf("create logger: %w", err)
	}
	zap.ReplaceGlobals(log)
	cnf, err := ReadConfig(f.configPath.Get(ctx))
	if err != nil {
		return empty, fmt.Errorf("get config: %w", err)
	}
	log.Debug("config readed")

	return baseCommand{
		log: log,
		cnf: cnf,
	}, nil
}

func (b *baseCommand) connectDB(ctx *cli.Context, debug bool) (*pgx.Conn, error) {
	if debug {
		b.cnf.DB.SetDebug(true)
	}
	conn, err := db.NewDB(ctx.Context, b.log, b.cnf.DB)
	if err != nil {
		return nil, fmt.Errorf("create database connection: %w", err)
	}
	b.log.Debug("connected to database")

	return conn, nil
}
