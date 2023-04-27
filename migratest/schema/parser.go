package schema

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type ParserConfig struct {
	Schema string
}

type Parser struct {
	c  ParserConfig
	sb squirrel.SelectBuilder
	db DBConn

	schema *Schema
}

func NewParser(c ParserConfig, db DBConn) *Parser {
	return &Parser{
		c:  c,
		db: db,
		sb: squirrel.SelectBuilder{}.PlaceholderFormat(squirrel.Dollar),
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
	q := p.sb.
		Columns("tablename").
		From("pg_tables").
		Where("schemaname = ?", p.c.Schema)

	tables, err := QueryAll(ctx, p.db.Conn, q, func(s pgx.Rows, t *Table) error {
		t.Columns = make(map[string]*Column)
		t.Constraints = make(map[string]*Constraint)
		return s.Scan(&t.Name)
	})
	if err != nil {
		return err
	}

	p.schema.Tables = make(map[string]*Table, len(tables))
	for idx := range tables {
		p.schema.Tables[tables[idx].Name] = &tables[idx]
	}

	return nil
}

func (p *Parser) LoadTablesColumns(ctx context.Context) error {
	q := p.sb.
		Columns(
			"table_name",
			"column_name",
			"data_type",
			"udt_name",
			"character_maximum_length",
			"domain_name",
			"is_nullable = 'YES'",
			"column_default",
			"is_generated = 'ALWAYS'",
			"generation_expression",
		).
		From("information_schema.columns").
		Where("table_schema = ? AND table_name = ANY(?)", p.c.Schema, p.schema.TableNames)

	columns, err := QueryAll(ctx, p.db.Conn, q,
		func(s pgx.Rows,
			c *struct {
				Table string
				Column
			},
		) error {
			return s.Scan(
				&c.Table,
				&c.Name,
				&c.Type.PrettyType,
				&c.Type.UDT,
				&c.Type.CharMaxLength,
				&c.Type.Domain,
				&c.Nullable,
				&c.Default,
				&c.ISGenerated,
				&c.Generated,
			)
		})
	if err != nil {
		return err
	}

	for idx, column := range columns {
		table, ok := p.schema.Tables[column.Table]
		if !ok {
			return fmt.Errorf("table %q not found", column.Table)
		}
		table.Columns[column.Name] = &columns[idx].Column
	}

	return nil
}

func (p *Parser) LoadConstraints(ctx context.Context) error {
	q := p.sb.
		Columns(
			"r.relname AS table_name",
			"c.conname AS constraint_name",
			"c.contype AS constraint_type",
		).
		From("pg_constraint c").
		Join("pg_class r ON c.conrelid = r.oid").
		Join("pg_namespace nr ON no.oid = r.relnamespace AND c.conrelid = r.oid").
		Where("nr.nspname = ? AND r.relname = ANY(?)", p.c.Schema, p.schema.TableNames)

	constraints, err := QueryAll(ctx, p.db.Conn, q,
		func(s pgx.Rows,
			c *struct {
				table          string
				constraint     string
				constraintType string
			},
		) error {
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
			Name:    constraint.constraint,
			Columns: make(map[string]*Column),
		}
		if err := c.SetType(constraint.constraintType); err != nil {
			return err
		}

		table.Constraints[c.Name] = &c
		p.schema.Constraints[c.Name] = &c

		switch c.Type {
		case ConstraintTypePK:
			table.PrimaryKey = &c
		case ConstraintTypeFK:
			// для FK запрос отдаёт таблицу, на которую этот FK ссылается
			table.ReferencedBy[c.Name] = &c
		}
	}

	return nil
}

func (p *Parser) LoadConstraintsColumns(ctx context.Context) error {
	q := p.sb.
		Columns(
			"table_name",
			"column_name",
			"constraint_name",
		).
		From("information_schema.constraint_column_usage").
		Where(
			"table_schema = ? AND table_name = ANY(?) AND constraint_name = ANY(?)",
			p.c.Schema, p.schema.TableNames, p.schema.ConstraintNames,
		)

	constraintsColumns, err := QueryAll(ctx, p.db.Conn, q,
		func(s pgx.Rows,
			c *struct {
				table      string
				column     string
				constraint string
			},
		) error {
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
		constraint.Columns[constraintColumn.table] = constraintColumn.column
	}

	return nil
}

func (p *Parser) LoadForeignConstraints(ctx context.Context) error {
	q := p.sb.
		Columns(
			"table_name",
			"column_name",
			"constraint_name",
		).
		From("information_schema.constraint_column_usage").
		Where(
			"table_schema = ? AND table_name = ANY(?) AND constraint_name = ANY(?)",
			p.c.Schema, p.schema.TableNames, p.schema.ConstraintNames,
		)

	constraintsColumns, err := QueryAll(ctx, p.db.Conn, q,
		func(s pgx.Rows,
			c *struct {
				table      string
				column     string
				constraint string
			},
		) error {
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
		constraint.Columns[constraintColumn.table] = constraintColumn.column
	}

	return nil
}
