package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Feresey/diplom/migratest/schema"
	"github.com/Feresey/diplom/migratest/schema/driver"
	"github.com/gobwas/glob"
	"github.com/jackc/pgx/v4"
)

type App struct {
	app   *fx.App
	c     *AppConfig
	flags *Flags

	logger *zap.Logger
	driver *driver.PostgresDriver
	m      *Migrator
	dumper *Dumper
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
			NewDumper,
		),
		fx.Populate(
			&app.driver,
			&app.logger,
			&app.m,
			&app.dumper,
		),
	)

	return app
}

func (app *App) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	if err := app.app.Start(ctx); err != nil {
		return fmt.Errorf("start application failed: %w", err)
	}

	if err := app.run(ctx); err != nil {
		return err
	}

	if err := app.app.Stop(ctx); err != nil {
		return fmt.Errorf("stop application failed: %w", err)
	}

	return nil
}

// GenerateData по конфигу для текущей версии миграции получает схему из базы и генерирует данные для таблиц.
func (app *App) GenerateData(
	ctx context.Context,
	cnf MigrationSettings,
) (map[string]TableRecords, error) {
	schemas, err := app.driver.GetSchemas(ctx)
	if err != nil {
		return nil, fmt.Errorf("get schemas: %w", err)
	}
	app.logger.Debug("database schemas", zap.Strings("schemas", schemas))

	var allTables []*schema.Table

	schemaSettings := app.GetSchemaConfigs(cnf, schemas)
	for _, ss := range schemaSettings {
		tables, err := app.driver.ParseSchema(ctx, ss)
		if err != nil {
			return nil, fmt.Errorf("parse schema %q: %w", ss.SchemaName, err)
		}
		allTables = append(allTables, tables...)
	}

	gen := NewGenerator(app.logger, allTables, cnf.Generator)

	data, err := gen.GenerateTablesData()
	if err != nil {
		return nil, fmt.Errorf("generate data: %w", err)
	}

	return data, nil
}

func (app *App) run(ctx context.Context) error {
	configIndexes := app.c.Migrations.GetVersionSettingsIndexes()

	/*
		1. get database version
		2. migrate up if database is clean, else skip
		3. get config for migrations:
		   - get schemas
		   - get tables
		   - generate data
		3. dump schema
		4. restore schema + insert data
		5. migrate up
		6. migrate down
		7. check data??
		8. migrate up -- same version as at the second step

		next iteration
	*/

	for {
		isLastMigration, err := app.m.Up()
		if err != nil {
			return err
		}

		if isLastMigration {
			// TODO something
			return nil
		}

		version, err := app.m.GetVersion()
		if err != nil {
			return err
		}
		app.logger.Info("current database version", zap.Int("version", version))

		cnf := app.c.Migrations.GetVersionConfig(configIndexes, version)
		data, err := app.GenerateData(ctx, cnf)
		if err != nil {
			return err
		}

		if err := app.dumper.InsertData(ctx, cnf.Migration.Patterns, app.InsertData(data)); err != nil {
			return fmt.Errorf("create dump: %w", err)
		}

		app.logger.Info("upgrade migration", zap.Int("next_version", version+1))
		// migrate up with data
		if _, err := app.m.Up(); err != nil {
			return err
		}

		app.logger.Info("downgrade with data", zap.Int("prev_version", version))
		// migrate down with data
		if err := app.m.Down(); err != nil {
			return err
		}

		// TODO check data

		app.logger.Info("upgrade migration back", zap.Int("next_version", version+1))
		// turn back to current version
		if _, err := app.m.Up(); err != nil {
			return err
		}

		version, err = app.m.GetVersion()
		if err != nil {
			return err
		}
		app.logger.Info("current database version", zap.Int("version", version))
	}
}

func (app *App) InsertData(data map[string]TableRecords) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		for _, tableRecords := range data {
			// copy to
			cpt := pgx.CopyFromSlice(
				len(tableRecords.Records),
				func(idx int) ([]interface{}, error) {
					return tableRecords.Records[idx], nil
				})
			_, err := app.driver.Conn.CopyFrom(
				ctx,
				pgx.Identifier{tableRecords.SchemaName, tableRecords.TableName},
				tableRecords.Columns,
				cpt)
			if err != nil {
				return fmt.Errorf("copy to table %q: %w", tableRecords.TableName, err)
			}
		}
		return nil
	}
}

// GetSchemaConfigs достаёт из базы имена схем и сопоставляет их шаблонам и конкретным конфигам
// TODO split
func (app *App) GetSchemaConfigs(cnf MigrationSettings, schemas []string) []schema.Settings {
	var allow, block []glob.Glob

	for _, pattern := range cnf.Migration.Patterns.Whitelist {
		glb, err := glob.Compile(pattern)
		if err != nil {
			app.logger.Warn("incorrect schema name glob pattern", zap.String("pattern", pattern), zap.Error(err))
			continue
		}
		allow = append(allow, glb)
	}

	for _, pattern := range cnf.Migration.Patterns.Blacklist {
		glb, err := glob.Compile(pattern)
		if err != nil {
			app.logger.Warn("incorrect schema name glob pattern", zap.String("pattern", pattern), zap.Error(err))
			continue
		}
		block = append(block, glb)
	}

	var resultSchemas []string

SCHEMAS_LOOP:
	for _, schemaName := range schemas {
		for _, bp := range block {
			if !bp.Match(schemaName) {
				continue SCHEMAS_LOOP
			}
		}
		for _, al := range allow {
			if al.Match(schemaName) {
				resultSchemas = append(resultSchemas, schemaName)
			}
		}
	}

	concreteConfigs := make(map[string]schema.Settings)

	for _, concrete := range cnf.Migration.ConcreteConfigs {
		concreteConfigs[concrete.SchemaName] = concrete
	}

	resultConfigs := make([]schema.Settings, len(concreteConfigs), 0)

	for _, schemaName := range resultSchemas {
		concrete, ok := concreteConfigs[schemaName]
		if ok {
			resultConfigs = append(resultConfigs, concrete)
		} else {
			resultConfigs = append(resultConfigs, schema.Settings{
				SchemaName: schemaName,
			})
		}
	}

	return resultConfigs
}
