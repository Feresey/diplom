package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Conn  string
	debug bool
}

func (c *Config) SetDebug(debug bool) { c.debug = debug }

func NewDB(
	ctx context.Context,
	logger *zap.Logger,
	cfg Config,
) (*pgx.Conn, error) {
	cnf, err := pgx.ParseConfig(cfg.Conn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.debug {
		cnf.Tracer = &tracelog.TraceLog{
			Logger:   tracelog.LoggerFunc(queryMessageLog(logger)),
			LogLevel: tracelog.LogLevelInfo,
		}
	}

	c, err := pgx.ConnectConfig(ctx, cnf)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return c, nil
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
