package app

import (
	"context"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v4/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/file"

	"github.com/Feresey/diplom/migratest/schema/driver"
)

type MigratorConfig struct {
	Path string `json:"path"`
}

type Migrator struct {
	logger *zap.Logger

	*migrate.Migrate
	targetInstance database.Driver
	sourceInstance source.Driver
}

func NewMigrator(
	lc fx.Lifecycle,
	config MigratorConfig,
	logger *zap.Logger,
	driver *driver.PostgresDriver,
) *Migrator {
	mm := &Migrator{
		logger: logger.Named("migrator"),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) (err error) {
			db := stdlib.OpenDB(*driver.Conn.Config())
			mm.targetInstance, err = postgres.WithInstance(db, new(postgres.Config))
			if err != nil {
				return fmt.Errorf("create target instance: %w", err)
			}

			mm.sourceInstance, err = new(file.File).Open(config.Path)
			if err != nil {
				return fmt.Errorf("create source instance: %w", err)
			}

			migrator, err := migrate.NewWithInstance("files", mm.sourceInstance, "postgres", mm.targetInstance)
			if err != nil {
				return fmt.Errorf("create migrator instance: %w", err)
			}

			mm.Migrate = migrator
			return nil
		},
		OnStop: mm.Shutdown,
	})

	return mm
}

func (m *Migrator) Shutdown(_ context.Context) (err error) {
	err = multierr.Append(err, m.targetInstance.Close())
	err = multierr.Append(err, m.sourceInstance.Close())
	return err
}
