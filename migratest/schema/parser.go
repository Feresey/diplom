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
	c  ParserConfig
	db DBConn

	schema *Schema
}

func NewParser(c ParserConfig, db DBConn) *Parser {
	return &Parser{
		c:  c,
		db: db,
	}
}

func (p *Parser) LoadSchema(ctx context.Context) error {
	if err := p.LoadTables(ctx); err != nil {
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

func (p *Parser) LoadTables(ctx context.Context) error {
	q := NewQuery[Table](`
		SELECT
			schemaname,
			tablename
		FROM
			pg_tables
		WHERE
			schemaname = ANY($1)`, p.c.Schemas)

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
			table_schema = ANY($1)
			AND table_name = ANY($2)
		`, p.c.Schemas, p.schema.TableNames)

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
		table          string
		constraint     string
		constraintType string
	}
	q := NewQuery[constraint](`
		SELECT
			r.relname AS table_name,
			c.conname AS constraint_name,
			c.contype AS constraint_type
		FROM
			pg_constraint c
			JOIN pg_class r ON c.conrelid = r.oid
			JOIN pg_namespace nr ON no.oid = r.relnamespace AND c.conrelid = r.oid
		WHERE
			nr.nspname = ANY($1)
			AND r.relname = ANY($2)`,
		p.c.Schemas, p.schema.TableNames,
	)

	constraints, err := q.All(ctx, p.db.Conn,
		func(s pgx.Rows, c *constraint) error {
			return s.Scan(
				&c.table,
				&c.constraint,
				&c.constraintType,
			)
		})
	if err != nil {
		return err
	}

	p.schema.Constraints = make(map[string]*Constraint, len(constraints))
	for _, constraint := range constraints {
		table, ok := p.schema.Tables[constraint.table]
		if !ok {
			return fmt.Errorf("unable to find table %q", constraint.table)
		}
		c := Constraint{
			Identifier:   constraint.constraint,
			Table:        constraint.table,
			TableColumns: make(map[string]*Column),
		}
		if err := c.SetType(constraint.constraintType); err != nil {
			return err
		}

		table.Constraints[c.Identifier] = &c
		p.schema.Constraints[c.Identifier] = &c

		switch c.Type {
		case ConstraintTypePK:
			table.PrimaryKey = &c
		case ConstraintTypeFK:
			// для FK запрос отдаёт таблицу, на которую этот FK ссылается
			table.ReferencedBy[c.Identifier] = &c
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
			SELECT
				table_name,
				column_name,
				constraint_name
			FROM
				information_schema.constraint_column_usage
			WHERE
				table_schema = ANY($1)
				AND table_name = ANY($2)
				AND constraint_name = ANY($3)`,
		p.c.Schemas, p.schema.TableNames, p.schema.ConstraintNames)

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
		constraint.TableColumns[constraintColumn.column] = column
	}

	return nil
}

func (p *Parser) LoadForeignConstraints(ctx context.Context) error {
	type tableColumn struct {
		schema string
		name   string
		column string
	}

	type foreignConstraint struct {
		name    string
		schema  string
		local   tableColumn
		foreign tableColumn
	}

	q := NewQuery[foreignConstraint](`
		SELECT
			tc.table_name,
			kcu.column_name,
			ccu.table_schema AS foreign_table_schema,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name
		FROM
			information_schema.constraints AS tc
		JOIN
			information_schema.key_column_usage AS kcu
			ON
				tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
		JOIN
			information_schema.constraint_column_usage AS ccu
			ON
				ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
		WHERE
			tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
			AND tc.table_name = ANY($2)`,
		p.c.Schema, p.schema.TableNames,
	)

	foreignKeys, err := q.All(ctx, p.db.Conn,
		func(s pgx.Rows, fc *foreignConstraint) error {
			return s.Scan(
				&c.table,
				&c.column,
				&c.constraint,
			)
		})
	if err != nil {
		return err
	}

	for _, constraintColumn := range foreignKeys {
		constraint, ok := p.schema.Constraints[constraintColumn.constraint]
		if !ok {
			return fmt.Errorf("constraint %q not found", constraintColumn.constraint)
		}
		constraint.Columns[constraintColumn.table] = constraintColumn.column
	}

	return nil
}
