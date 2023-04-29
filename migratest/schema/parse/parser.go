package parse

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/Feresey/mtest/schema"
	"github.com/Feresey/mtest/schema/db"
)

// TODO
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
// TODO как ограничивать внутри схемы таблицы, которые будут обрабатываться? Или на этом этапе это неважно?
func (p *Parser) LoadSchema(ctx context.Context, schemas []string) (*schema.Schema, error) {
	var s schema.Schema

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
func (p *Parser) LoadTables(ctx context.Context, s *schema.Schema, schemas []string) error {
	tables, err := QueryTables(ctx, p.db.Conn, schemas)
	if err != nil {
		p.log.Error("failed to query tables", zap.Error(err))
		return err
	}
	// TODO debug log

	s.Tables = make(map[string]*schema.Table, len(tables))
	s.TableNames = make([]string, 0, len(tables))
	for _, dbtable := range tables {
		table := &schema.Table{
			Name: schema.Identifier{
				Schema: dbtable.Schema,
				Name:   dbtable.Table,
			},
			Columns:      make(map[string]*schema.Column),
			Constraints:  make(map[string]*schema.Constraint),
			ForeignKeys:  make(map[string]schema.ForeignKey),
			ReferencedBy: make(map[string]*schema.Constraint),
		}

		s.Tables[table.Name.String()] = table
		s.TableNames = append(s.TableNames, table.Name.String())
	}

	return nil
}

// LoadTablesColumns загружает колонки таблиц, включая типы и аттрибуты
func (p *Parser) LoadTablesColumns(ctx context.Context, s *schema.Schema) error {
	columns, err := QueryTablesColumns(ctx, p.db.Conn, s.TableNames)
	if err != nil {
		p.log.Error("failed to query tables columns", zap.Error(err))
		return err
	}
	// TODO debug log

	for _, dbcolumn := range columns {
		tableName := schema.Identifier{
			Schema: dbcolumn.TableSchema,
			Name:   dbcolumn.TableName,
		}
		table, ok := s.Tables[tableName.String()]
		if !ok {
			// TODO element not found error
			return fmt.Errorf("table %q not found", tableName)
		}

		column := &schema.Column{
			Name:  dbcolumn.ColumnName,
			Table: table,
			// TODO types
			Type: nil,
			Attributes: schema.ColumnAttributes{
				IsDefault:   dbcolumn.ColumnDefault.Valid,
				Default:     dbcolumn.ColumnDefault.String,
				Nullable:    dbcolumn.IsNullable,
				ISGenerated: dbcolumn.IsGenerated.Bool,
				Generated:   dbcolumn.GenerationExpression.String,
			},
		}

		// TODO types
		table.Columns[column.Name] = column
		table.ColumnNames = append(table.ColumnNames, column.Name)
	}

	return nil
}

// перевод значений колонки pg_constraint.type
var pgConstraintType = map[string]schema.ConstraintType{
	"p": schema.ConstraintTypePK,
	"f": schema.ConstraintTypeFK,
	"c": schema.ConstraintTypeCheck,
	"u": schema.ConstraintTypeUnique,
	"t": schema.ConstraintTypeTrigger,
	"x": schema.ConstraintTypeExclusion,
}

// LoadConstraints загружает ограничения для всех найденных таблиц
func (p *Parser) LoadConstraints(ctx context.Context, s *schema.Schema) error {
	constraints, err := QueryTablesConstraints(ctx, p.db.Conn, s.TableNames)
	if err != nil {
		p.log.Error("failed to query tables constraints", zap.Error(err))
		return err
	}
	// TODO debug log

	s.Constraints = make(map[string]*schema.Constraint, len(constraints))
	s.ConstraintNames = make([]string, 0, len(constraints))
	for _, dbconstraint := range constraints {
		tableName := schema.Identifier{
			Schema: dbconstraint.TableSchema,
			Name:   dbconstraint.TableName,
		}

		table, ok := s.Tables[tableName.String()]
		if !ok {
			return fmt.Errorf("unable to find table %q", tableName)
		}
		typ, ok := pgConstraintType[dbconstraint.ConstraintType]
		if !ok {
			return fmt.Errorf("unsupported constraint type: %q", dbconstraint.ConstraintType)
		}

		c := &schema.Constraint{
			Name: schema.Identifier{
				Schema: dbconstraint.ConstraintSchema,
				Name:   dbconstraint.ConstraintName,
			},
			Table:            table,
			Type:             typ,
			NullsNotDistinct: dbconstraint.NullsNotDistinct,
			Definition:       dbconstraint.ConstraintDef,
			Columns:          make(map[string]*schema.Column),
		}

		s.Constraints[c.Name.String()] = c
		s.ConstraintNames = append(s.ConstraintNames, c.Name.String())

		// TODO это точно должно быть здесь?
		switch c.Type {
		case schema.ConstraintTypePK:
			table.PrimaryKey = c
		case schema.ConstraintTypeFK:
			// для FK запрос отдаёт таблицу, на которую этот FK ссылается
			table.ReferencedBy[c.Name.String()] = c
		}
	}

	return nil
}

func (p *Parser) LoadConstraintsColumns(ctx context.Context, s *schema.Schema) error {
	constraintsColumns, err := QueryConstraintsColumns(ctx, p.db.Conn, s.TableNames, s.ConstraintNames)
	if err != nil {
		p.log.Error("failed to query constraints columns", zap.Error(err))
		return err
	}
	// TODO debug log

	for _, dbc := range constraintsColumns {
		constraintName := schema.Identifier{
			Schema: dbc.ConstraintSchema,
			Name:   dbc.ConstraintName,
		}
		tableName := schema.Identifier{
			Schema: dbc.TableSchema,
			Name:   dbc.TableName,
		}

		constraint, ok := s.Constraints[constraintName.String()]
		if !ok {
			return fmt.Errorf("constraint %q not found", constraintName)
		}
		table, ok := s.Tables[tableName.String()]
		if !ok {
			return fmt.Errorf("table %q not found for constraint %q", tableName, constraintName)
		}
		column, ok := table.Columns[dbc.ColumnName]
		if !ok {
			return fmt.Errorf("column %q not found in table %q for constraint %q",
				dbc.ColumnName,
				tableName,
				constraintName,
			)
		}

		constraint.Columns[dbc.ColumnName] = column
	}

	return nil
}

func (p *Parser) LoadForeignConstraints(ctx context.Context, s *schema.Schema) error {
	fks, err := QueryForeignKeys(ctx, p.db.Conn, s.ConstraintNames)
	if err != nil {
		p.log.Error("failed to query foreign constraints", zap.Error(err))
		return err
	}
	// TODO debug log

	for _, dbfk := range fks {
		fkName := schema.Identifier{
			Schema: dbfk.ConstraintSchema,
			Name:   dbfk.ConstraintName,
		}
		uniqName := schema.Identifier{
			Schema: dbfk.UniqueConstraintSchema,
			Name:   dbfk.UniqueConstraintName,
		}
		fk, ok := s.Constraints[fkName.String()]
		if !ok {
			return fmt.Errorf("constraint %q not found", fkName)
		}
		uniq, ok := s.Constraints[uniqName.String()]
		if !ok {
			return fmt.Errorf("constraint %q not found", uniqName)
		}

		// таблица, которая ссылается
		fk.Table.ForeignKeys[fk.Name.String()] = schema.ForeignKey{
			Uniq:    uniq,
			Foreign: fk,
		}
		// таблица, на которую ссылаются
		uniq.Table.ReferencedBy[uniq.Name.String()] = uniq
	}

	return nil
}
