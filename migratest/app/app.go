package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Feresey/diplom/migratest/schema/driver"
	"github.com/golang-migrate/migrate/v4"
)

type App struct {
	app   *fx.App
	c     *AppConfig
	flags *Flags

	logger *zap.Logger
	driver *driver.PostgresDriver
	m      *Migrator
}

func New(logger *zap.Logger, conf *AppConfig, flags *Flags) *App {
	app := &App{
		c:     conf,
		flags: flags,
	}
	app.app = fx.New(
		fx.Supply(
			*conf,
			logger,
		),
		fx.Provide(
			NewMigrator,
			driver.NewPostgresDriver,
		),
		fx.Populate(&app.driver, &app.logger, &app.m),
	)

	return app
}

func (app *App) Run() error {
	startCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.app.Start(startCtx); err != nil {
		return fmt.Errorf("start application failed: %w", err)
	}

	runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.run(runCtx); err != nil {
		return err
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.app.Stop(stopCtx); err != nil {
		return fmt.Errorf("stop application failed: %w", err)
	}

	return nil
}

func (app *App) run(ctx context.Context) error {
	// TODO one by one
	if err := app.m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("apply migrations: %w", err)
		}
		// _ = app.m.Down()
		// goto up
	}
	// TODO multiple schemas
	info, err := app.driver.ParseSchema(ctx, app.c.Schema.ConcreteConfig[0])
	if err != nil {
		return fmt.Errorf("parse schema: %w", err)
	}

	file, err := os.Create(app.flags.OutFile)
	if err != nil {
		return err
	}
	defer file.Close()

	grapth := BuildGrapth(info)
	enc := json.NewEncoder(file)
	enc.SetIndent("", "\t")

	if err := enc.Encode(grapth); err != nil {
		return fmt.Errorf("show schema: %w", err)
	}

	return nil
}
