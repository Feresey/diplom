package schema

import (
	"context"
	"fmt"

	"github.com/Feresey/mtest/schema/db"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type ParserConfig struct {
	Schemas []string
}

type Parser struct {
	db  *db.DBConn
	log *zap.Logger
}

func NewParser(
	db *db.DBConn,
	log *zap.Logger,
) *Parser {
	return &Parser{
		log: log.Named("parser"),
		db:  db,
	}
}

// TODO вернуть ошибку если данные ссылаются на не указанную схему
func (p *Parser) LoadSchema(ctx context.Context, schemas []string) (*Schema, error) {
	var s Schema

	if err := p.LoadTables(ctx, &s, schemas); err != nil {
		return nil, err
	}
	if err := p.LoadTablesColumns(ctx, &s); err != nil {
		return nil, err
	}
	if err := p.LoadConstraints(ctx, &s); err != nil {
		return nil, err
	}
	if err := p.LoadConstraintsColumns(ctx, &s); err != nil {
		return nil, err
	}
	if err := p.LoadForeignConstraints(ctx, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// LoadTables получает имена таблиц, найденных в схемах
func (p *Parser) LoadTables(ctx context.Context, s *Schema, schemas []string) error {
	q := db.NewQuery[Table](`
-- list tables
SELECT
	schemaname,
	tablename
FROM
	pg_tables
WHERE
	schemaname = ANY($1)
`, schemas)

	tables, err := q.AllRet(ctx, p.db.Conn, func(rows pgx.Rows) (Table, error) {
		t := Table{
			Columns:      make(map[string]*Column),
			ForeignKeys:  make(map[string]ForeignKey),
			ReferencedBy: make(map[string]*Constraint),
			Constraints:  make(map[string]*Constraint),
		}
		return t, rows.Scan(&t.Name.Schema, &t.Name.Name)
	})
	if err != nil {
		return err
	}

	s.Tables = make(map[string]*Table, len(tables))
	s.TableNames = make([]string, 0, len(tables))
	for idx, table := range tables {
		s.Tables[table.Name.String()] = &tables[idx]
		s.TableNames = append(s.TableNames, table.Name.String())
	}

	return nil
}

// LoadTablesColumns загружает колонки таблиц, включая типы и аттрибуты
func (p *Parser) LoadTablesColumns(ctx context.Context, s *Schema) error {
	type tableColumn struct {
		Table Identifier
		Column
	}
	q := db.NewQuery[tableColumn](`
-- tables columns
SELECT
	table_name,
	table_schema,
	column_name,
	data_type,
	data_type = 'ARRAY' AS is_array,
	data_type = 'USER-DEFINED' AS is_user_type,
	udt_schema,
	udt_name,
	character_maximum_length,
	domain_schema,
	domain_name,
	is_nullable = 'YES' AS is_nullable,
	column_default,
	is_generated = 'ALWAYS' AS is_generated,
	generation_expression
FROM
	information_schema.columns
WHERE
	table_schema || '.' || table_name = ANY($1)`,
		s.TableNames)

	columns, err := q.All(ctx, p.db.Conn,
		func(rows pgx.Rows, c *tableColumn) error {
			return rows.Scan(
				&c.Table.Name,
				&c.Table.Schema,

				&c.Name,
				&c.Type.TypeName,
				&c.Type.IsArray,
				&c.Type.IsUserType,
				&c.Type.UDTSchema,
				&c.Type.UDT,
				&c.Type.CharMaxLength,
				&c.Type.DomainSchema,
				&c.Type.Domain,
				&c.Attributes.Nullable,
				&c.Attributes.Default,
				&c.Attributes.ISGenerated,
				&c.Attributes.Generated,
			)
		})
	if err != nil {
		return err
	}
	p.log.Debug("", zap.Reflect("columns", columns))

	for idx, column := range columns {
		table, ok := s.Tables[column.Table.String()]
		if !ok {
			return fmt.Errorf("table %q not found", column.Table)
		}
		col := &columns[idx].Column
		col.Table = table
		table.Columns[column.Name] = col
		table.ColumnNames = append(table.ColumnNames, col.Name)
	}

	return nil
}

// LoadConstraints загружает ограничения для всех найденных таблиц
func (p *Parser) LoadConstraints(ctx context.Context, s *Schema) error {
	type constraint struct {
		Table            Identifier
		Constraint       Identifier
		ConstraintType   string
		NullsNotDistinct bool
		ConstraintDef    string
	}
	q := db.NewQuery[constraint](`
-- tables constraints
SELECT
	nr.nspname::TEXT AS table_schema,
	r.relname::TEXT  AS table_name,
	nr.nspname::TEXT AS constraint_schema,
	c.conname::TEXT  AS constraint_name,
	c.contype::TEXT  AS constraint_type,
	COALESCE(
		(SELECT NOT pg_index.indnullsnotdistinct
			FROM pg_index
			WHERE pg_index.indexrelid = c.conindid
		), False
	) AS nulls_not_distinct,
	pg_get_constraintdef(c.oid) as constraint_def
FROM
	pg_constraint c
	JOIN pg_class r ON c.conrelid = r.oid
	JOIN pg_namespace nr ON nr.oid = r.relnamespace AND c.conrelid = r.oid
WHERE
	nr.nspname || '.' || r.relname = ANY($1)`,
		s.TableNames,
	)

	constraints, err := q.All(ctx, p.db.Conn,
		func(rows pgx.Rows, c *constraint) error {
			return rows.Scan(
				&c.Table.Schema,
				&c.Table.Name,
				&c.Constraint.Schema,
				&c.Constraint.Name,
				&c.ConstraintType,
				&c.NullsNotDistinct,
				&c.ConstraintDef,
			)
		})
	if err != nil {
		return err
	}
	p.log.Debug("query constraints", zap.Reflect("constraints", constraints))

	s.Constraints = make(map[string]*Constraint, len(constraints))
	for _, constraint := range constraints {
		table, ok := s.Tables[constraint.Table.String()]
		if !ok {
			return fmt.Errorf("unable to find table %q", constraint.Table)
		}
		c := Constraint{
			Name:             constraint.Constraint,
			Table:            table,
			Columns:          make(map[string]*Column),
			NullsNotDistinct: constraint.NullsNotDistinct,
			Definition:       constraint.ConstraintDef,
		}
		if err := c.SetType(constraint.ConstraintType); err != nil {
			return err
		}

		table.Constraints[c.Name.String()] = &c
		s.Constraints[c.Name.String()] = &c
		s.ConstraintNames = append(s.ConstraintNames, c.Name.String())

		switch c.Type {
		case ConstraintTypePK:
			table.PrimaryKey = &c
		case ConstraintTypeFK:
			// для FK запрос отдаёт таблицу, на которую этот FK ссылается
			table.ReferencedBy[c.Name.String()] = &c
		}
	}

	return nil
}

func (p *Parser) LoadConstraintsColumns(ctx context.Context, s *Schema) error {
	type constraintColumn struct {
		Table      Identifier
		Column     string
		Constraint Identifier
	}
	q := db.NewQuery[constraintColumn](`
-- constraints columns
SELECT
	ccu.table_schema,
	ccu.table_name,
	ccu.column_name,
	ccu.constraint_schema,
	ccu.constraint_name
FROM
	information_schema.constraint_column_usage ccu
WHERE
	ccu.table_schema || '.' || ccu.table_name = ANY($1)
	AND ccu.constraint_schema || '.' || ccu.constraint_name = ANY($2)`,
		s.TableNames, s.ConstraintNames)

	constraintsColumns, err := q.All(ctx, p.db.Conn,
		func(rows pgx.Rows, c *constraintColumn) error {
			return rows.Scan(
				&c.Table.Schema,
				&c.Table.Name,
				&c.Column,
				&c.Constraint.Schema,
				&c.Constraint.Name,
			)
		})
	if err != nil {
		return err
	}
	p.log.Debug("", zap.Reflect("constraints_columns", constraintsColumns))

	for _, constraintColumn := range constraintsColumns {
		constraint, ok := s.Constraints[constraintColumn.Constraint.String()]
		if !ok {
			return fmt.Errorf("constraint %q not found", constraintColumn.Constraint)
		}

		table, ok := s.Tables[constraintColumn.Table.String()]
		if !ok {
			return fmt.Errorf("table %q not found for constraint %q", constraintColumn.Table, constraintColumn.Constraint)
		}
		column, ok := table.Columns[constraintColumn.Column]
		if !ok {
			return fmt.Errorf("column %q not found in table %q for constraint %q",
				constraintColumn.Column,
				constraintColumn.Table,
				constraintColumn.Constraint,
			)
		}
		constraint.Columns[constraintColumn.Column] = column
	}

	return nil
}

func (p *Parser) LoadForeignConstraints(ctx context.Context, s *Schema) error {
	type foreignConstraint struct {
		Foreign Identifier
		Unique  Identifier
	}

	q := db.NewQuery[foreignConstraint](`
-- foreign keys
SELECT
	constraint_schema,
	constraint_name,
	unique_constraint_schema,
	unique_constraint_name,
	match_option,
	update_rule,
	delete_rule
FROM
	information_schema.referential_constraints
WHERE
	constraint_schema || '.' || constraint_name = ANY($1)`,
		s.ConstraintNames,
	)

	foreignKeys, err := q.All(ctx, p.db.Conn,
		func(rows pgx.Rows, fc *foreignConstraint) error {
			return rows.Scan(
				&fc.Foreign.Schema,
				&fc.Foreign.Name,
				&fc.Unique.Schema,
				&fc.Unique.Name,
				nil,
				nil,
				nil,
			)
		})
	if err != nil {
		return err
	}
	p.log.Debug("", zap.Reflect("fk", foreignKeys))

	for _, keys := range foreignKeys {
		fk, ok := s.Constraints[keys.Foreign.String()]
		if !ok {
			return fmt.Errorf("constraint %q not found", keys.Foreign)
		}

		uniq, ok := s.Constraints[keys.Unique.String()]
		if !ok {
			return fmt.Errorf("constraint %q not found", keys.Foreign)
		}

		// таблица, в которой есть FK
		fk.Table.ForeignKeys[fk.Name.String()] = ForeignKey{
			Uniq:    uniq,
			Foreign: fk,
		}
		// таблица, на которую ссылаются
		uniq.Table.ReferencedBy[uniq.Name.String()] = uniq
	}

	return nil
}
