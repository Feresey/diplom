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
	s := &schema.Schema{
		Types:          make(map[string]*schema.DBType),
		ArrayTypes:     make(map[string]*schema.ArrayType),
		CompositeTypes: make(map[string]*schema.CompositeType),
		EnumTypes:      make(map[string]*schema.EnumType),
		RangeTypes:     make(map[string]*schema.RangeType),
		DomainTypes:    make(map[string]*schema.DomainType),
		Tables:         make(map[string]*schema.Table),
		Constraints:    make(map[string]*schema.Constraint),
	}

	if err := p.LoadTables(ctx, s, schemas); err != nil {
		return nil, err
	}
	if err := p.LoadTablesColumns(ctx, s); err != nil {
		return nil, err
	}
	if err := p.LoadConstraints(ctx, s); err != nil {
		return nil, err
	}
	if err := p.LoadConstraintsColumns(ctx, s); err != nil {
		return nil, err
	}
	if err := p.LoadForeignConstraints(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// LoadTables получает имена таблиц, найденных в схемах
func (p *Parser) LoadTables(ctx context.Context, s *schema.Schema, schemas []string) error {
	tables, err := QueryTables(ctx, p.db.Conn, schemas)
	if err != nil {
		p.log.Error("failed to query tables", zap.Error(err))
		return err
	}
	p.log.Debug("loaded tables", zap.Reflect("tables", tables))

	// realloc
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
			ForeignKeys:  make(map[string]*schema.ForeignKey),
			ReferencedBy: make(map[string]*schema.Constraint),
		}

		s.Tables[table.Name.String()] = table
		s.TableNames = append(s.TableNames, table.Name.String())
		// FIXME
		if len(s.Tables) != len(s.TableNames) {
			return fmt.Errorf("duplicated tables")
		}
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
	p.log.Debug("columns loaded", zap.Reflect("columns", columns))

	for _, dbcolumn := range columns {
		tableName := schema.Identifier{
			Schema: dbcolumn.TableSchema,
			Name:   dbcolumn.TableName,
		}
		typeName := schema.Identifier{
			Schema: dbcolumn.TypeSchema,
			Name:   dbcolumn.TypeName,
		}
		table, ok := s.Tables[tableName.String()]
		if !ok {
			// TODO element not found error
			return fmt.Errorf("table %q not found", tableName)
		}
		typ, err := p.addType(s, typeName, dbcolumn)
		if err != nil {
			return err
		}

		column := &schema.Column{
			Name:  dbcolumn.ColumnName,
			Table: table,
			Type:  typ,
			Attributes: schema.ColumnAttributes{
				HasDefault:       dbcolumn.HasDefault || dbcolumn.IsGenerated,
				Default:          dbcolumn.DefaultExpr.String,
				Nullable:         dbcolumn.IsNullable,
				HasCharMaxLength: dbcolumn.TypeMaxLength.Valid,
				CharMaxLength:    dbcolumn.TypeMaxLength.Int,
				ArrayDims:        dbcolumn.ArrayDims,
			},
		}

		table.Columns[column.Name] = column
		table.ColumnNames = append(table.ColumnNames, column.Name)
		// FIXME
		if len(table.Columns) != len(table.ColumnNames) {
			return fmt.Errorf("duplicated columns")
		}
	}

	return nil
}

// перевод значений колонки pg_type.typtype
var pgTypType = map[string]schema.DataType{
	"b": schema.DataTypeBase,
	"c": schema.DataTypeComposite,
	"d": schema.DataTypeDomain,
	"e": schema.DataTypeEnum,
	"r": schema.DataTypeRange,
	// TODO что с ним делать???
	"m": schema.DataTypeRange,
}

func (p *Parser) addType(
	s *schema.Schema,
	typeName schema.Identifier,
	dbcolumn queryTablesColumns,
) (*schema.DBType, error) {
	typ, ok := s.Types[typeName.String()]
	if ok {
		return typ, nil
	}

	typType, ok := pgTypType[dbcolumn.TypeType]
	if !ok {
		return nil, fmt.Errorf("type value is undefined: %q", dbcolumn.TypeType)
	}
	if dbcolumn.ArrayDims != 0 {
		typType = schema.DataTypeArray
	}

	typ = &schema.DBType{
		TypeName: typeName,
		Type:     typType,
	}

	switch typType {
	default:
	case schema.DataTypeBase:
	case schema.DataTypeDomain:
		domainTypeName := schema.Identifier{
			Schema: dbcolumn.TypeSchema,
			Name:   dbcolumn.TypeName,
		}
		domain, ok := s.DomainTypes[domainTypeName.String()]
		if !ok {
			domain = &schema.DomainType{
				TypeName: domainTypeName,
				// TODO тип элемента домена это другой тип? Или он просто имеет аттрибуты как колонка таблицы?
				// ElemType: ,
			}
			s.DomainTypes[domainTypeName.String()] = domain
		}
		typ.DomainType = domain
	case schema.DataTypeArray:
		arrTypeName := schema.Identifier{
			Schema: dbcolumn.TypeSchema,
			Name:   dbcolumn.TypeName,
		}
		arr, ok := s.ArrayTypes[arrTypeName.String()]
		if !ok {
			typ.ArrayType = arr
			arr = &schema.ArrayType{
				TypeName: arrTypeName,
				// ElemType: ,
			}
			s.ArrayTypes[arrTypeName.String()] = arr
		}
		typ.ArrayType = arr
	case schema.DataTypeEnum:
		enumName := schema.Identifier{
			Schema: dbcolumn.TypeSchema,
			Name:   dbcolumn.TypeName,
		}
		enum, ok := s.EnumTypes[enumName.String()]
		if !ok {
			enum = &schema.EnumType{
				TypeName: enumName,
				// Values: ,
			}
			s.EnumTypes[enumName.String()] = enum
		}
		typ.EnumType = enum
	case schema.DataTypeRange:
		rangeName := schema.Identifier{
			Schema: dbcolumn.TypeSchema,
			Name:   dbcolumn.TypeName,
		}
		rng, ok := s.RangeTypes[rangeName.String()]
		if !ok {
			rng = &schema.RangeType{
				TypeName: rangeName,
				// ElemType: ,
			}
			s.RangeTypes[rangeName.String()] = rng
		}
		typ.RangeType = rng
	case schema.DataTypeComposite:
		compositeName := schema.Identifier{
			Schema: dbcolumn.CompositeSchema.String,
			// TODO тут разве не должен быть TypeName?
			Name: dbcolumn.CompositeType.String,
		}
		composite, ok := s.CompositeTypes[compositeName.String()]
		if !ok {
			composite = &schema.CompositeType{
				TypeName:   compositeName,
				Attributes: make(map[string]*schema.CompositeAttribute),
			}
			s.CompositeTypes[compositeName.String()] = composite
		}
		typ.CompositeType = composite
	}

	s.Types[typeName.String()] = typ

	return typ, nil
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
	p.log.Debug("loaded constraints", zap.Reflect("constraints", constraints))

	// realloc
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
		table.Constraints[c.Name.String()] = c
		s.ConstraintNames = append(s.ConstraintNames, c.Name.String())
		// FIXME
		if len(s.Constraints) != len(s.ConstraintNames) {
			return fmt.Errorf("duplicated constraints")
		}

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
			Schema: dbc.SchemaName,
			Name:   dbc.ConstraintName,
		}
		tableName := schema.Identifier{
			Schema: dbc.SchemaName,
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
	p.log.Debug("loaded foreign constraints", zap.Reflect("constraints", fks))

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
		fk.Table.ForeignKeys[fk.Name.String()] = &schema.ForeignKey{
			Uniq:    uniq,
			Foreign: fk,
		}
		// таблица, на которую ссылаются
		uniq.Table.ReferencedBy[uniq.Name.String()] = uniq
	}

	return nil
}
