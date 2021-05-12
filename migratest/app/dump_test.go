package app

import (
	"context"
	"testing"

	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"

	"github.com/Feresey/diplom/migratest/schema/driver"
)

func TestDump(t *testing.T) {
	cnf := driver.Config{
		Credentials: driver.UserInfo{
			Username: "postgres",
			Password: "pass",
		},
		Host:   "localhost",
		Port:   5432,
		DBName: "test",
	}

	var d *Dumper

	app := fxtest.New(
		t,
		fx.Supply(
			zap.NewExample(),
			cnf,
			MigrationConfig{},
		),
		fx.Provide(
			NewDumper,
			driver.NewPostgresDriver,
		),
		fx.Populate(&d),
	)

	app.RequireStart()
	defer app.RequireStop()

	ctx := context.Background()

	err := d.InsertData(ctx, nil)
	require.NoError(t, err)
}
