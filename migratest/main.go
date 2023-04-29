package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Feresey/mtest/config"
	"github.com/Feresey/mtest/schema"
	"github.com/Feresey/mtest/schema/db"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

func newConfig() config.Config {
	// TODO load from file
	return config.Config{
		DBConn: "postgresql://postgres:postgres@localhost:5432",
	}
}

type ErrorHandler struct {
	log *zap.Logger
}

func (e ErrorHandler) HandleError(err error) {
	e.log.Error(err.Error())
	// vis, verr := fx.VisualizeError(err)
	// e.log.Info(vis, zap.Error(verr))
}

func NewApp(populate fx.Option) *fx.App {
	flags := config.NewFlags()
	flag.Parse()
	log := newLogger(flags)
	config := newConfig()

	return fx.New(
		fx.Supply(
			log,
			config,
			flags,
		),
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			l := &fxevent.ZapLogger{Logger: logger}
			l.UseErrorLevel(zap.DebugLevel)
			l.UseLogLevel(zap.DebugLevel)
			return l
		}),
		fx.Provide(
			db.NewDB,
			schema.NewParser,
		),
		fx.StartTimeout(5*time.Second),
		fx.StopTimeout(time.Second),
		fx.ErrorHook(ErrorHandler{
			log: log,
		}),

		populate,
	)
}

func main() {
	var (
		parser *schema.Parser
		log    *zap.Logger
	)

	app := NewApp(fx.Populate(&parser, &log))

	runCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-app.Done()
		cancel()
	}()

	err := app.Start(runCtx)
	if err != nil {
		log.Fatal("failed to start app")
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
	if err := launch(runCtx, log, parser); err != nil {
		if err != nil {
			log.Error("launch app", zap.Error(err))
		}
	}
}

func launch(ctx context.Context, log *zap.Logger, parser *schema.Parser) error {
	s, err := parser.LoadSchema(ctx, []string{"test"})
	if err != nil {
		prettyErr, ok := err.(interface{ Pretty() string })
		if ok {
			log.Error(prettyErr.Pretty())
			return nil
		}

		// TODO экзит коды
		return fmt.Errorf("error loading schema: %w", err)
	}

	if err := s.Dump(os.Stdout); err != nil {
		return fmt.Errorf("failed to dump schema: %w", err)
	}

	return nil
}
