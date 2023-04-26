package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Feresey/diplom/migratest/app"
)

func main() {
	lc := zap.NewDevelopmentConfig()
	lc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := lc.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build logger: %v\n", err)
		os.Exit(1)
	}

	flags := app.ParseFlags()
	cnf, err := app.GetConfig(flags)
	if err != nil {
		logger.Fatal("unable to load config", zap.Error(err))
	}

	app := app.New(logger, cnf, flags)
	if err := app.Run(); err != nil {
		logger.Fatal("run app", zap.Error(err))
	}
}
