package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/Feresey/mtest/config"
	"github.com/Feresey/mtest/schema"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger() *zap.Logger {
	lc := zap.NewDevelopmentConfig()
	lc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	log, err := lc.Build()
	if err != nil {
		println("failed to build zap logger: ", err)
		os.Exit(1)
	}

	return log
}

func newConfig() config.Config {
	return config.Config{
		DBConn: "postgresql://postgres:postgres@localhost:5432",
	}
}

func main() {
	log := newLogger()
	config := newConfig()

	var parser *schema.Parser

	app := fx.New(
		fx.Supply(
			log,
			config,
		),
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			l := &fxevent.ZapLogger{Logger: logger}
			l.UseErrorLevel(zap.ErrorLevel)
			l.UseLogLevel(zap.DebugLevel)
			return l
		}),
		fx.Provide(
			schema.NewDB,
			schema.NewParser,
		),
		fx.StartTimeout(5*time.Second),
		fx.StopTimeout(time.Second),
		fx.Populate(
			&parser,
		),
		fx.Invoke(func(sh fx.Shutdowner, p *schema.Parser) {
		}),
	)

	err := app.Start(context.Background())
	if err != nil {
		log.Fatal("start app", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-app.Done()
		cancel()
	}()

	s, err := parser.LoadSchema(ctx, []string{"test"})
	if err != nil {
		log.Error("load schema", zap.Error(err))
		var qErr schema.Error
		if errors.As(err, &qErr) {
			log.Sugar().Errorf("error:\n%s", qErr.Pretty())
		}
	}

	s.Dump(os.Stdout)


	err = app.Stop(context.Background())
	if err != nil {
		log.Fatal("stop app", zap.Error(err))
	}
}
