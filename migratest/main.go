package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/Feresey/mtest/config"
	"github.com/Feresey/mtest/schema"
	"github.com/Feresey/mtest/schema/db"
	"github.com/Feresey/mtest/schema/parse"
)

func newLogger(flags *config.Flags) *zap.Logger {
	lc := zap.NewDevelopmentConfig()
	lc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	lc.DisableStacktrace = true
	if flags.Debug {
		lc.Level.SetLevel(zap.DebugLevel)
	} else {
		lc.Level.SetLevel(zap.InfoLevel)
	}
	log, err := lc.Build()
	if err != nil {
		println("failed to build zap logger: ", err)
		os.Exit(1)
	}

	return log
}

func readConfig(confPath string) (config.FileConfig, error) {
	var fc config.FileConfig
	file, err := os.ReadFile(confPath)
	if err != nil {
		return fc, err
	}

	err = yaml.Unmarshal(file, &fc)
	return fc, err
}

type ErrorHandler struct {
	log   *zap.Logger
	debug bool
}

func (e ErrorHandler) HandleError(err error) {
	e.log.Error(err.Error())
	if e.debug {
		vis, verr := fx.VisualizeError(err)
		e.log.Info(vis, zap.Error(verr))
	}
}

func NewApp(populate fx.Option) *fx.App {
	flags := config.NewFlags()
	flag.Parse()

	log := newLogger(flags)
	cnf, err := readConfig(flags.Config)
	if err != nil {
		println("Failed to read config", err.Error())
		os.Exit(2)
	}

	return fx.New(
		fx.Supply(
			log,
			cnf,
			flags,
		),
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			l := &fxevent.ZapLogger{Logger: logger}
			l.UseErrorLevel(zap.DebugLevel)
			l.UseLogLevel(zap.DebugLevel)
			return l
		}),
		fx.Provide(
			config.NewConfig,
			db.NewDB,
			parse.NewParser,
		),
		fx.StartTimeout(flags.StartTimeout),
		fx.StopTimeout(flags.StopTimeout),
		fx.ErrorHook(ErrorHandler{
			log:   log,
			debug: flags.Debug,
		}),

		populate,
	)
}

func main() {
	var (
		log          *zap.Logger
		parser       *parse.Parser
		parserConfig config.Parser
	)

	app := NewApp(fx.Populate(&log, &parser, &parserConfig))

	runCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-app.Done()
		cancel()
	}()

	err := app.Start(runCtx)
	if err != nil {
		println("failed to start app", err.Error())
		os.Exit(1)
	}
	log.Debug("app started")

	defer func() {
		stopCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		err = app.Stop(stopCtx)
		if err != nil {
			log.Fatal("stop app", zap.Error(err))
		}
		log.Debug("app stopped gracefullty")
	}()

	// TODO добавить команды (schema, generate, ...)
	if err := launch(runCtx, log, parser, parserConfig); err != nil {
		if err != nil {
			log.Error("launch app", zap.Error(err))
		}
	}
}

func launch(
	ctx context.Context,
	log *zap.Logger,
	parser *parse.Parser,
	pc config.Parser,
) error {
	s, err := parser.LoadSchema(ctx, pc)
	if err != nil {
		prettyErr, ok := err.(interface{ Pretty() string })
		if ok {
			log.Error(prettyErr.Pretty())
			// TODO экзит коды
			return nil
		}

		return fmt.Errorf("error loading schema: %w", err)
	}

	log.Info("dump schema")
	if err := s.Dump(os.Stdout, schema.DumpSchemaTemplate); err != nil {
		return fmt.Errorf("failed to dump schema: %w", err)
	}

	log.Info("dump types")
	if err := s.Dump(os.Stdout, schema.DumpTypesTemplate); err != nil {
		return fmt.Errorf("failed to dump types: %w", err)
	}

	return nil
}
