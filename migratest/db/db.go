package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Feresey/mtest/config"
)

type DBConn struct {
	*pgx.Conn
}

func NewDB(
	lc fx.Lifecycle,
	logger *zap.Logger,
	cfg config.DBConn,
	flags *config.Flags,
) (*DBConn, error) {
	var conn DBConn

	cnf, err := pgx.ParseConfig(string(cfg))
	if err != nil {
		return nil, err
	}

	if flags.Debug {
		cnf.Tracer = &tracelog.TraceLog{
			Logger:   tracelog.LoggerFunc(queryMessageLog(logger)),
			LogLevel: tracelog.LogLevelInfo,
		}
	}

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

func queryMessageLog(log *zap.Logger) func(
	ctx context.Context,
	level tracelog.LogLevel,
	msg string,
	data map[string]any,
) {
	return func(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
		if msg == "Prepare" {
			return
		}
		var rawSQL *string
		fields := make([]zapcore.Field, 0, len(data))
		for k, v := range data {
			f := zap.Any(k, v)
			if f.Key == "sql" && f.Type == zapcore.StringType {
				rawSQL = &f.String
				continue
			}
			fields = append(fields, f)
		}

		var lvl zapcore.Level
		switch level {
		default:
			fallthrough
		case tracelog.LogLevelNone, tracelog.LogLevelTrace, tracelog.LogLevelDebug:
			lvl = zapcore.DebugLevel
		case tracelog.LogLevelInfo:
			lvl = zapcore.InfoLevel
		case tracelog.LogLevelWarn:
			lvl = zapcore.WarnLevel
		case tracelog.LogLevelError:
			lvl = zapcore.ErrorLevel
		}
		if rawSQL != nil {
			msg = msg + "\n" + *rawSQL
		}
		ce := log.Check(lvl, msg)
		ce.Write(fields...)
	}
}
