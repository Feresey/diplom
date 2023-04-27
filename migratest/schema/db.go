package schema

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Feresey/mtest/config"
)

type DBConn struct {
	*pgx.Conn
}

func NewDB(lc fx.Lifecycle, logger *zap.Logger, cfg config.DBConfig) (*DBConn, error) {
	var conn DBConn

	cnf, err := pgx.ParseConfig(string(cfg))
	if err != nil {
		return nil, err
	}
	// cnf.Tracer = &tracelog.TraceLog{
	// 	Logger:   pgxzap.NewLogger(logger),
	// 	LogLevel: tracelog.LogLevelInfo,
	// }

	lc.Append(fx.StartStopHook(
		func(ctx context.Context) error {
			c, err := pgx.ConnectConfig(ctx, cnf)
			conn.Conn = c
			return err
		},
		func(ctx context.Context) error {
			return conn.Conn.Close(ctx)
		},
	))

	return &conn, nil
}
