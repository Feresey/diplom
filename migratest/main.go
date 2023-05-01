package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Feresey/mtest/db"
	"github.com/Feresey/mtest/parse"
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

type flags struct {
	Config       *cli.StringFlag
	Debug        *cli.BoolFlag
	StartTimeout *cli.DurationFlag
	StopTimeout  *cli.DurationFlag
}

func (f *flags) Set() []cli.Flag {
	return []cli.Flag{
		f.Config,
		f.Debug,
		f.StartTimeout,
		f.StopTimeout,
	}
}

func main() {
	f := flags{
		Config: &cli.StringFlag{
			Name:      "config",
			Value:     "mtest.yml",
			Usage:     "config file path",
			TakesFile: true,
			Aliases:   []string{"c"},
		},
		Debug: &cli.BoolFlag{
			Name:   "debug",
			Value:  false,
			Usage:  "show debug information",
			Hidden: true,
		},
		StartTimeout: &cli.DurationFlag{
			Name:  "start-timeout",
			Value: 5 * time.Second,
		},
		StopTimeout: &cli.DurationFlag{
			Name:  "stop-timeout",
			Value: 5 * time.Second,
		},
	}

	app := &cli.App{
		Name:        "mtest",
		Description: "migration test tool",
		Flags:       f.Set(),
		Commands: []*cli.Command{
			DumpCommand(f),
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func NewApp(
	ctx *cli.Context,
	log *zap.Logger,
	cnf FxConfig,
	f flags,
	populate fx.Option,
) *fx.App {
	return fx.New(
		fx.Supply(
			log,
			cnf,
		),
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			l := &fxevent.ZapLogger{Logger: logger}
			l.UseErrorLevel(zap.DebugLevel)
			l.UseLogLevel(zap.DebugLevel)
			return l
		}),
		fx.Provide(
			db.NewDB,
			parse.NewParser,
		),
		fx.StartTimeout(f.StartTimeout.Get(ctx)),
		fx.StopTimeout(f.StopTimeout.Get(ctx)),
		fx.ErrorHook(ErrorHandler{
			log:   log,
			debug: f.Debug.Get(ctx),
		}),

		populate,
	)
}

func RunApp(app *fx.App, log *zap.Logger, run func(ctx context.Context) error) (runErr error) {
	runCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-app.Done()
		cancel()
	}()

	err := app.Start(runCtx)
	if err != nil {
		return fmt.Errorf("start app: %w", err)
	}
	log.Debug("app started")

	defer func() {
		stopCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		err = app.Stop(stopCtx)
		if err != nil {
			runErr = errors.Join(runErr, err)
		}
		log.Debug("app stopped gracefullty")
	}()

	return run(runCtx)
}
