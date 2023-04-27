package schema

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type ParserConfig struct {
	Schemas []string
}

type Parser struct {
	db *DBConn

	schema Schema
}

func NewParser(db *DBConn) *Parser {
	return &Parser{
		db: db,
	}
}

func (p *Parser) LoadSchema(ctx context.Context, schemas []string) error {
	if err := p.LoadTables(ctx, schemas); err != nil {
		return err
	}
	p.schema.setTableNames()
	if err := p.LoadTablesColumns(ctx); err != nil {
		return err
	}
	if err := p.LoadConstraints(ctx); err != nil {
		return err
	}
	p.schema.setConstraintsNames()
	if err := p.LoadConstraintsColumns(ctx); err != nil {
		return err
	}
	return nil
}

func (p *Parser) LoadTables(ctx context.Context, schemas []string) error {
	q := NewQuery[Table](`
		-- list tables
		SELECT
			schemaname,
			tablename
		FROM
			pg_tables
		WHERE
			schemaname = ANY($1)`, schemas)

	tables, err := q.AllRet(ctx, p.db.Conn, func(s pgx.Rows) (Table, error) {
		t := Table{
			Columns:      make(map[string]*Column),
			ForeignKeys:  make(map[string]ForeignKey),
			ReferencedBy: make(map[string]*Constraint),
			Constraints:  make(map[string]*Constraint),
		}
		return t, s.Scan(&t.Name.Schema, &t.Name.Name)
	})
	if err != nil {
		return err
	}

	p.schema.Tables = make(map[string]*Table, len(tables))
	for idx := range tables {
		p.schema.Tables[tables[idx].Name.String()] = &tables[idx]
	}

	return nil
}

func (p *Parser) LoadTablesColumns(ctx context.Context) error {
	type tableColumn struct {
		Table Identifier
		Column
	}
	q := NewQuery[tableColumn](`
		-- tables columns
		SELECT
			table_name,
			table_schema,
			column_name,
			data_type,
			udt_schema,
			udt_name,
			character_maximum_length,
			domain_schema,
			domain_name,
			is_nullable = 'YES',
			column_default,
			is_generated = 'ALWAYS',
			generation_expression
		FROM
			information_schema.columns
		WHERE
			table_schema || '.' || table_name = ANY($1)`,
		p.schema.TableNames)

	columns, err := q.All(ctx, p.db.Conn,
		func(s pgx.Rows, c *tableColumn) error {
			return s.Scan(
				&c.Table.Name,
				&c.Table.Schema,

				&c.Name,
				&c.Type.Type,
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

	for idx, column := range columns {
		table, ok := p.schema.Tables[column.Table.String()]
		if !ok {
			return fmt.Errorf("table %q not found", column.Table)
		}
		table.Columns[column.Name] = &columns[idx].Column
	}

	return nil
}

func (p *Parser) LoadConstraints(ctx context.Context) error {
	type constraint struct {
		table          Identifier
		constraint     string
		constraintType string
	}
	q := NewQuery[constraint](`
		-- tables constraints
		SELECT
			nr.nspname::TEXT AS table_schema,
			r.relname::TEXT AS table_name,
			c.conname::TEXT AS constraint_name,
			c.contype::TEXT AS constraint_type
		FROM
			pg_constraint c
			JOIN pg_class r ON c.conrelid = r.oid
			JOIN pg_namespace nr ON nr.oid = r.relnamespace AND c.conrelid = r.oid
		WHERE
			nr.nspname || '.' || r.relname = ANY($1)`,
		p.schema.TableNames,
	)

	constraints, err := q.All(ctx, p.db.Conn,
		func(s pgx.Rows, c *constraint) error {
			return s.Scan(
				&c.table.Schema,
				&c.table.Name,
				&c.constraint,
				&c.constraintType,
			)
		})
	if err != nil {
		return err
	}

	p.schema.Constraints = make(map[string]*Constraint, len(constraints))
	for _, constraint := range constraints {
		table, ok := p.schema.Tables[constraint.table.String()]
		if !ok {
			return fmt.Errorf("unable to find table %q", constraint.table)
		}
		c := Constraint{
			Name: Identifier{
				Schema: constraint.table.Schema,
				Name:   constraint.constraint,
			},
			Table:   table,
			Columns: make(map[string]*Column),
		}
		if err := c.SetType(constraint.constraintType); err != nil {
			return err
		}

		table.Constraints[c.Name.String()] = &c
		p.schema.Constraints[c.Name.String()] = &c

		switch c.Type {
		case ConstraintTypePK:
			table.PrimaryKey = &c
		case ConstraintTypeFK:
			// FIXME
			// для FK запрос отдаёт таблицу, на которую этот FK ссылается
			table.ReferencedBy[c.Name.String()] = &c
		}
	}

	return nil
}

func (p *Parser) LoadConstraintsColumns(ctx context.Context) error {
	type constraintColumn struct {
		table      string
		column     string
		constraint string
	}
	q := NewQuery[constraintColumn](`
		-- constraints columns
		SELECT
			table_name,
			column_name,
			constraint_name
		FROM
			information_schema.constraint_column_usage
		WHERE
			table_schema || '.' || table_name = ANY($1)
			AND constraint_name = ANY($2)`,
		p.schema.TableNames, p.schema.ConstraintNames)

	constraintsColumns, err := q.All(ctx, p.db.Conn,
		func(s pgx.Rows, c *constraintColumn) error {
			return s.Scan(
				&c.table,
				&c.column,
				&c.constraint,
			)
		})
	if err != nil {
		return err
	}

	for _, constraintColumn := range constraintsColumns {
		constraint, ok := p.schema.Constraints[constraintColumn.constraint]
		if !ok {
			return fmt.Errorf("constraint %q not found", constraintColumn.constraint)
		}

		table, ok := p.schema.Tables[constraintColumn.table]
		if !ok {
			return fmt.Errorf("table %q not found for constraint %q", constraintColumn.table, constraintColumn.constraint)
		}
		column, ok := table.Columns[constraintColumn.column]
		if !ok {
			return fmt.Errorf("column %q not found in table %q for constraint %q",
				constraintColumn.column,
				constraintColumn.table,
				constraintColumn.constraint,
			)
		}
		constraint.Columns[constraintColumn.column] = column
	}

	return nil
}

func (p *Parser) LoadForeignConstraints(ctx context.Context) error {
	type foreignConstraint struct {
		foreign Identifier
		unique  Identifier
	}

	q := NewQuery[foreignConstraint](`
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
			AND constraint_schema || '.' || constraint_name = ANY($1)`,
		p.schema.ConstraintNames,
	)

	foreignKeys, err := q.All(ctx, p.db.Conn,
		func(s pgx.Rows, fc *foreignConstraint) error {
			return s.Scan(
				&fc.foreign.Schema,
				&fc.foreign.Name,
				&fc.unique.Schema,
				&fc.unique.Name,
				nil,
				nil,
				nil,
			)
		})
	if err != nil {
		return err
	}

	for _, keys := range foreignKeys {
		fk, ok := p.schema.Constraints[keys.foreign.String()]
		if !ok {
			return fmt.Errorf("constraint %q not found", keys.foreign)
		}

		uniq, ok := p.schema.Constraints[keys.unique.String()]
		if !ok {
			return fmt.Errorf("constraint %q not found", keys.foreign)
		}

		// FIXME
		// таблица, в которой есть FK
		uniq.Table.ForeignKeys[fk.Name.String()] = ForeignKey{
			Uniq:    uniq,
			Foreign: fk,
		}
		// таблица, на которую ссылаются
		fk.Table.ReferencedBy[uniq.Name.String()] = uniq
	}

	return nil
}
