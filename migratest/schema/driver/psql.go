// Package driver implements an sqlboiler driver.
// It can be used by either building the main.go in the same project
// and using aS a binary or using the side effect import.
package driver

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zapadapter"
	"github.com/volatiletech/strmangle"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/Feresey/diplom/migratest/schema"
)

// PostgresDriver holds the database connection string and a handle
// to the database connection.
type PostgresDriver struct {
	logger  *zap.Logger
	Conn    *pgx.Conn
	version int
}

type In struct {
	fx.In
	LC     fx.Lifecycle
	Logger *zap.Logger
	Config Config
}

func NewPostgresDriver(in In) (*PostgresDriver, error) {
	logger := in.Logger.Named("driver")

	cnf, err := pgx.ParseConfig(in.Config.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cnf.Logger = zapadapter.NewLogger(logger.Named("pgx"))
	cnf.LogLevel = pgx.LogLevelWarn

	driver := &PostgresDriver{
		logger: logger,
	}

	in.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var err error
			driver.Conn, err = pgx.ConnectConfig(ctx, cnf)
			if err != nil {
				return fmt.Errorf("connect: %w", err)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return driver.Conn.Close(ctx)
		},
	})

	return driver, nil
}

func (p *PostgresDriver) GetSchemas(ctx context.Context) ([]string, error) {
	rows, err := p.Conn.Query(ctx, `SELECT schema_name FROM information_schema.schemata`)
	if err != nil {
		return nil, fmt.Errorf("create get schemas query: %w", err)
	}
	defer rows.Close()

	var res []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan schema name: %w", err)
		}
		res = append(res, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan rows error: %w", err)
	}
	return res, nil
}

func (p *PostgresDriver) ParseSchema(
	ctx context.Context,
	sc schema.SchemaSettings,
) (dbinfo []*schema.Table, err error) {
	err = p.getVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("get database version: %w", err)
	}

	tables, err := p.Tables(ctx, sc)
	if err != nil {
		return nil, fmt.Errorf("get schema info: %w", err)
	}

	return tables, err
}

// getVersion gets the version of underlying database
func (p *PostgresDriver) getVersion(ctx context.Context) error {
	var version string
	row := p.Conn.QueryRow(ctx, "SHOW server_version_num")
	if err := row.Scan(&version); err != nil {
		return err
	}

	v, err := strconv.Atoi(version)
	if err != nil {
		return err
	}
	p.version = v
	return nil
}

// tablesFromList takes a whitelist or blacklist and returns
// the table names.
func tablesFromList(list []string) []string {
	if len(list) == 0 {
		return nil
	}

	var tables []string
	for _, i := range list {
		splits := strings.Split(i, ".")

		if len(splits) == 1 {
			tables = append(tables, splits[0])
		}
	}

	return tables
}

func (p *PostgresDriver) filterTables(sc schema.SchemaSettings, query string, args ...interface{}) (string, []interface{}) {
	if wl := sc.Whitelist; len(wl) > 1 {
		tables := tablesFromList(wl)
		if len(tables) > 0 {
			query += fmt.Sprintf(" AND table_name IN (%s)", strmangle.Placeholders(true, len(tables), 2, 1))
			for _, w := range tables {
				args = append(args, w)
			}
		}
	} else if bl := sc.Blacklist; len(bl) > 0 {
		tables := tablesFromList(bl)
		if len(tables) > 0 {
			query += fmt.Sprintf(" AND table_name NOT IN (%s)", strmangle.Placeholders(true, len(tables), 2, 1))
			for _, b := range tables {
				args = append(args, b)
			}
		}
	}
	return query, args
}

// TableNames connects to the postgres database and
// retrieves all table names from the information_schema where the
// table schema is schema. It uses a whitelist and blacklist.
func (p *PostgresDriver) TableNames(ctx context.Context, sc schema.SchemaSettings) ([]string, error) {
	var names []string

	query := `
SELECT
	table_name
FROM
	information_schema.tables
WHERE
	table_schema = $1
	AND table_type = 'BASE TABLE'`

	args := []interface{}{sc.SchemaName}

	p.filterTables(sc, query)
	if wl := sc.Whitelist; len(wl) > 1 {
		tables := tablesFromList(wl)
		if len(tables) > 0 {
			query += fmt.Sprintf(" AND table_name IN (%s)", strmangle.Placeholders(true, len(tables), 2, 1))
			for _, w := range tables {
				args = append(args, w)
			}
		}
	} else if bl := sc.Blacklist; len(bl) > 0 {
		tables := tablesFromList(bl)
		if len(tables) > 0 {
			query += fmt.Sprintf(" AND table_name NOT IN (%s)", strmangle.Placeholders(true, len(tables), 2, 1))
			for _, b := range tables {
				args = append(args, b)
			}
		}
	}

	query += ` ORDER BY table_name`

	rows, err := p.Conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("create table names query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		names = append(names, name)
	}

	return names, rows.Err()
}

//go:embed columns.sql
var allColumnsQuery string

// Columns takes a table name and attempts to retrieve the table information
// from the database information_schema.columns. It retrieves the column names
// and column types and returns those aS a []Column after TranslateColumnType()
// converts the SQL types to Go types, for example: "varchar" to "string"
func (p *PostgresDriver) Columns(ctx context.Context, sc schema.SchemaSettings, tableName string) (res []*schema.Column, err error) {
	rows, err := p.Conn.Query(ctx, allColumnsQuery, sc.SchemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("create all columns query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			column   schema.Column
			identity bool
		)
		if err := rows.Scan(
			&column.Name, &column.DBType, &column.FullDBType,
			&column.UDTName, &column.ArrType,
			&column.DomainName, &column.Default,
			&column.Comment, &column.Nullable,
			&identity, &column.Unique,
		); err != nil {
			return nil, fmt.Errorf("scan columns for table %s: %w", tableName, err)
		}
		if identity {
			column.Default = new(string)
			*column.Default = "IDENTITY"
		}

		res = append(res, &column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan rows: %w", err)
	}

	return res, nil
}

// PrimaryKeyInfo looks up the primary key for a table.
func (p *PostgresDriver) PrimaryKeyInfo(ctx context.Context, sc schema.SchemaSettings, tableName string) (*schema.PrimaryKey, error) {
	pkey := &schema.PrimaryKey{}
	var err error

	pkNameQuery := `
SELECT
	tc.constraint_name
FROM
	information_schema.table_constraints AS tc
WHERE
	tc.table_name = $1
	AND tc.constraint_type = 'PRIMARY KEY'
	AND tc.table_schema = $2`

	row := p.Conn.QueryRow(ctx, pkNameQuery, tableName, sc.SchemaName)
	if err = row.Scan(&pkey.Name); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get PK name: %w", err)
	}

	pkColumnsQuery := `
SELECT
	k.column_name
FROM
	information_schema.key_column_usage AS k
WHERE
	constraint_name = $1
	AND table_name = $2
	AND table_schema = $3
ORDER BY
	k.ordinal_position`

	rows, err := p.Conn.Query(ctx, pkColumnsQuery, pkey.Name, tableName, sc.SchemaName)
	if err != nil {
		return nil, fmt.Errorf("create pk columns name query: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string

		err = rows.Scan(&column)
		if err != nil {
			return nil, fmt.Errorf("scan line: %w", err)
		}

		columns = append(columns, column)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("scan rows: %w", err)
	}

	pkey.Columns = columns

	return pkey, nil
}

// ForeignKeyInfo retrieves the foreign keys for a given table name.
func (p *PostgresDriver) ForeignKeyInfo(ctx context.Context, sc schema.SchemaSettings, tableName string) (fkeys []*schema.ForeignKey, err error) {
	whereConditions := []string{"pgn.nspname = $2", "pgc.relname = $1", "pgcon.contype = 'f'"}
	if p.version >= 120000 {
		whereConditions = append(whereConditions, "pgasrc.attgenerated = ''", "pgadst.attgenerated = ''")
	}

	query := fmt.Sprintf(`
	SELECT
		pgcon.conname,
		pgc.relname AS source_table,
		pgasrc.attname AS source_column,
		dstlookupname.relname AS dest_table,
		pgadst.attname AS dest_column
	FROM pg_namespace pgn
		INNER JOIN pg_class pgc ON pgn.oid = pgc.relnamespace AND pgc.relkind = 'r'
		INNER JOIN pg_constraint pgcon ON pgn.oid = pgcon.connamespace AND pgc.oid = pgcon.conrelid
		INNER JOIN pg_class dstlookupname ON pgcon.confrelid = dstlookupname.oid
		INNER JOIN pg_attribute pgasrc ON pgc.oid = pgasrc.attrelid AND pgasrc.attnum = ANY(pgcon.conkey)
		INNER JOIN pg_attribute pgadst ON pgcon.confrelid = pgadst.attrelid AND pgadst.attnum = ANY(pgcon.confkey)
	where %s
	ORDER BY pgcon.conname, source_table, source_column, dest_table, dest_column`,
		strings.Join(whereConditions, " and "),
	)

	rows, err := p.Conn.Query(ctx, query, tableName, sc.SchemaName)
	if err != nil {
		return nil, fmt.Errorf("create FK query: %w", err)
	}

	for rows.Next() {
		var fkey schema.ForeignKey
		var sourceTable string

		fkey.Table = tableName
		err = rows.Scan(&fkey.Name, &sourceTable, &fkey.Column, &fkey.ForeignTable, &fkey.ForeignColumn)
		if err != nil {
			return nil, fmt.Errorf("scan line: %w", err)
		}

		fkeys = append(fkeys, &fkey)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("scan rows: %w", err)
	}

	return fkeys, nil
}
