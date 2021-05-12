package app

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"go.uber.org/fx"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v4/stdlib"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/file"

	"github.com/Feresey/diplom/migratest/schema"
	"github.com/Feresey/diplom/migratest/schema/driver"
)

// MigrationConfig для версии миграции
type MigrationConfig struct {
	// Patterns это маска для имени схем. Тут имеется ввиду именно PostgreSQL SCHEMA.
	Patterns schema.SchemaPatterns `json:"patterns,omitempty"`
	// ConcreteConfigs - настройки для конкретных схем. Я хз нужно это или нет.
	ConcreteConfigs []schema.SchemaSettings `json:"concrete_config,omitempty"`
}

func (mc *MigrationConfig) Copy() MigrationConfig {
	res := *mc

	res.Patterns.Blacklist = make([]string, len(mc.Patterns.Blacklist))
	res.Patterns.Whitelist = make([]string, len(mc.Patterns.Whitelist))
	copy(res.Patterns.Blacklist, mc.Patterns.Blacklist)
	copy(res.Patterns.Whitelist, mc.Patterns.Whitelist)

	res.ConcreteConfigs = make([]schema.SchemaSettings, len(mc.ConcreteConfigs))
	for idx, val := range mc.ConcreteConfigs {
		cnf := res.ConcreteConfigs[idx]

		cnf.SchemaName = val.SchemaName
		cnf.Blacklist = make([]string, len(val.Blacklist))
		cnf.Whitelist = make([]string, len(val.Whitelist))
		copy(cnf.Blacklist, val.Blacklist)
		copy(cnf.Whitelist, val.Whitelist)

		res.ConcreteConfigs[idx] = cnf
	}
	return res
}

func NewDefaultMigrationConfig() MigrationConfig {
	return MigrationConfig{
		Patterns: schema.SchemaPatterns{
			Whitelist: []string{"*"},
			Blacklist: []string{"pg_*", "information_schema"},
		},
		ConcreteConfigs: []schema.SchemaSettings{
			{SchemaName: "public"},
		},
	}
}

// MigrationSettings - конфиг для одной версии миграции.
// Для последующих версий конфиг расширяется за счёт предыдущих, если не указано обратное.
type MigrationSettings struct {
	// SkipOlderConfigs - нивелирует влияние прошлых конфигов на текущий.
	SkipOlderConfigs bool `json:"skip_older_configs,omitempty"`
	// Migration - настройки для взаимодействия со схемами внутри базы.
	Migration MigrationConfig `json:"migration,omitempty"`
	// Generator - конфиг для генерации данных.
	Generator GeneratorConfig `json:"generator,omitempty"`
}

func (ms MigrationSettings) Copy() MigrationSettings {
	res := ms
	res.Migration = res.Migration.Copy()
	res.Generator = res.Generator.Copy()

	return res
}

func (ms *MigrationSettings) MergeWith(rc MigrationSettings) {
	if rc.SkipOlderConfigs {
		*ms = rc.Copy()
		return
	}
	ms.Generator.MergeWith(rc.Generator)
	// не ну тут хороший вопрос надо ли ваще дополнять
	ms.Migration = rc.Migration
}

func MergeMigrationSettings(ms []MigrationSettings) MigrationSettings {
	if len(ms) == 0 {
		return MigrationSettings{}
	}

	res := ms[0]
	for _, cnf := range ms {
		res.MergeWith(cnf)
	}
	return res
}

type MigratorConfig struct {
	Path string `json:"path,omitempty"`
	// Settings настройки для конкретных миграций.
	//  ключ - номер миграции.
	//  значение - конфиг для конкретной миграции.
	Settings map[int]MigrationSettings `json:"settings,omitempty"`
}

func NewDefaultMigratorConfig() MigratorConfig {
	return MigratorConfig{
		Path: "./migtions/",
		Settings: map[int]MigrationSettings{
			0: {
				SkipOlderConfigs: false,
				Migration:        NewDefaultMigrationConfig(),
			},
		},
	}
}

func (mc *MigratorConfig) GetVersionSettingsIndexes() []int {
	res := make([]int, len(mc.Settings), 0)
	for idx := range mc.Settings {
		res = append(res, idx)
	}
	sort.Ints(res)
	return res
}

func (mc *MigratorConfig) GetVersionConfig(indexes []int, version int) MigrationSettings {
	if len(indexes) == 0 {
		// TODO use default settings
		return MigrationSettings{
			Migration:        NewDefaultMigrationConfig(),
			SkipOlderConfigs: false,
		}
	}
	res := mc.Settings[indexes[0]].Copy()
	for _, idx := range indexes[1:] {
		if idx > version {
			break
		}
		res.MergeWith(mc.Settings[idx])
	}
	return res
}

type Migrator struct {
	logger *zap.Logger

	m              *migrate.Migrate
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

			mm.m = migrator
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

func (m *Migrator) GetVersion() (int, error) {
	version, dirty, err := m.m.Version()
	if err != nil {
		if !errors.Is(err, migrate.ErrNilVersion) {
			return 0, fmt.Errorf("get database version: %w", err)
		}
		return 0, nil
	}
	if dirty {
		err = fmt.Errorf("database is dirty: %w", migrate.ErrDirty{Version: int(version)})
	}
	return int(version), err
}

// Up повышает версию миграций на 1 шаг.
func (m *Migrator) Up() (noChange bool, err error) {
	err = m.m.Steps(1)
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			// TODO do something
			m.logger.Info("max migrations reached")
			return true, nil
		}
		return false, fmt.Errorf("migrate up: %w", err)
	}
	return false, nil
}

// Up понижает версию миграций на 1 шаг.
func (m *Migrator) Down() error {
	err := m.m.Steps(-1)
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			// TODO do something
			m.logger.Error("min migrations reached")
			return nil
		}
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}
