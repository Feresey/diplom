package parse

import (
	"context"
	"fmt"
	"sort"

	"go.uber.org/zap"

	"github.com/Feresey/mtest/schema"
	"github.com/Feresey/mtest/schema/db"
	"github.com/Feresey/mtest/schema/parse/queries"
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

	if err := p.LoadEnums(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// LoadTables получает имена таблиц, найденных в схемах
func (p *Parser) LoadTables(ctx context.Context, s *schema.Schema, schemas []string) error {
	tables, err := queries.QueryTables(ctx, p.db.Conn, schemas)
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

// LoadTablesColumns загружает колонки таблиц, включая типы и аттрибуты
func (p *Parser) LoadTablesColumns(ctx context.Context, s *schema.Schema) error {
	columns, err := queries.QueryColumns(ctx, p.db.Conn, s.TableNames)
	if err != nil {
		p.log.Error("failed to query tables columns", zap.Error(err))
		return err
	}
	p.log.Debug("columns loaded", zap.Reflect("columns", columns))

	for _, dbcolumn := range columns {
		tableName := schema.Identifier{
			Schema: dbcolumn.SchemaName,
			Name:   dbcolumn.TableName,
		}
		typeName := schema.Identifier{
			Schema: dbcolumn.SchemaName,
			Name:   dbcolumn.TypeName,
		}
		table, ok := s.Tables[tableName.String()]
		if !ok {
			// TODO element not found error
			return fmt.Errorf("table %q not found", tableName)
		}

		typ, ok := s.Types[typeName.String()]
		if !ok {
			typeType, ok := pgTypType[dbcolumn.TypeName]
			if !ok {
				return fmt.Errorf("type value is undefined: %q", dbcolumn.TypeType)
			}

			typ = &schema.DBType{
				TypeName: typeName,
				Type:     typeType,
			}
			s.Types[typeName.String()] = typ
		}

		column := &schema.Column{
			Name:  dbcolumn.ColumnName,
			Table: table,
			Type:  typ,
			Attributes: schema.ColumnAttributes{
				HasDefault: dbcolumn.HasDefault || dbcolumn.IsGenerated,
				Default:    dbcolumn.DefaultExpr.String,
				DomainAttributes: schema.DomainAttributes{
					Nullable:         dbcolumn.IsNullable,
					HasCharMaxLength: dbcolumn.CharacterMaxLength.Valid,
					CharMaxLength:    dbcolumn.CharacterMaxLength.Int,
					ArrayDims:        dbcolumn.ArrayDims,
				},
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
	constraints, err := queries.QueryTableConstraints(ctx, p.db.Conn, s.TableNames)
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
	constraintsColumns, err := queries.QueryConstraintColumns(ctx, p.db.Conn, s.TableNames, s.ConstraintNames)
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
	fks, err := queries.QueryForeignKeys(ctx, p.db.Conn, s.ConstraintNames)
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

func (p *Parser) LoadTypes(ctx context.Context, s *schema.Schema) error {
	var err error
	loadTypes := mapKeys(s.Types)

	for len(loadTypes) != 0 {
		loadTypes, err = p.loadTypes(ctx, s, loadTypes)
		if err != nil {
			return fmt.Errorf("error loading types: %w", err)
		}
	}

	return nil
}

func (p *Parser) loadTypes(
	ctx context.Context,
	s *schema.Schema,
	typeNames []string,
) (moreTypes []string, err error) {
	types, err := queries.QueryTypes(ctx, p.db.Conn, typeNames)
	if err != nil {
		return nil, err
	}

	for _, dbtype := range types {
		typeName := schema.Identifier{
			Schema: dbtype.SchemaName,
			Name:   dbtype.TypeName,
		}

		typ, ok := s.Types[typeName.String()]
		if !ok {
			return nil, fmt.Errorf("internal error, type %q not found", typeName)
		}

		more, err := p.fillType(s, typ, dbtype)
		if err != nil {
			return nil, fmt.Errorf("fill type %q: %w", typeName, err)
		}
		moreTypes = append(moreTypes, more...)
	}

	return moreTypes, nil
}

func (p *Parser) fillType(
	s *schema.Schema,
	typ *schema.DBType,
	dbtype queries.Type,
) (moreTypes []string, err error) {
	typType, ok := pgTypType[dbtype.TypeType]
	if !ok {
		return nil, fmt.Errorf("type value is undefined: %q", dbtype.TypeType)
	}
	if dbtype.IsArray {
		typType = schema.DataTypeArray
	}
	typ.Type = typType

	switch typType {
	default:
	case schema.DataTypeBase:
	case schema.DataTypeDomain:
		domainTypeName := schema.Identifier{
			Schema: dbtype.SchemaName,
			Name:   dbtype.DomainType.String,
		}
		domain, ok := s.DomainTypes[domainTypeName.String()]
		if ok {
			return nil, fmt.Errorf("duplicate domain type: %q", domainTypeName)
		}

		elemTypeName := schema.Identifier{
			Schema: dbtype.DomainSchema.String,
			Name:   dbtype.DomainType.String,
		}
		elemType, ok := s.Types[elemTypeName.String()]
		if !ok {
			moreTypes = append(moreTypes, elemTypeName.String())
			elemType = &schema.DBType{
				TypeName: elemTypeName,
			}
			s.Types[elemTypeName.String()] = elemType
		}

		domain = &schema.DomainType{
			TypeName: domainTypeName,
			Attributes: schema.DomainAttributes{
				Nullable:         !dbtype.DomainIsNotNullable,
				HasCharMaxLength: dbtype.DomainCharacterMaxSize.Valid,
				CharMaxLength:    dbtype.DomainCharacterMaxSize.Int,
				ArrayDims:        domain.Attributes.ArrayDims,
			},
			// TODO тип элемента домена это другой тип? Или он просто имеет аттрибуты как колонка таблицы?
			ElemType: elemType,
		}
		s.DomainTypes[domainTypeName.String()] = domain
		typ.DomainType = domain
	case schema.DataTypeArray:
		arrTypeName := schema.Identifier{
			Schema: dbtype.ElemTypeSchema.String,
			Name:   dbtype.ElemTypeSchema.String,
		}
		elemType, ok := s.Types[arrTypeName.String()]
		if !ok {
			elemType = &schema.DBType{
				TypeName: arrTypeName,
			}
			s.Types[arrTypeName.String()] = elemType
		}
		arr := &schema.ArrayType{
			TypeName: arrTypeName,
			ElemType: elemType,
		}
		s.ArrayTypes[arrTypeName.String()] = arr
		typ.ArrayType = arr
	case schema.DataTypeEnum:
		enumName := schema.Identifier{
			Schema: dbtype.SchemaName,
			Name:   dbtype.TypeName,
		}
		enum, ok := s.EnumTypes[enumName.String()]
		if !ok {
			enum = &schema.EnumType{
				TypeName: enumName,
				// Values: ,
			}
			s.EnumTypes[enumName.String()] = enum
		}
		enumType, ok := s.Types[enumName.String()]
		if !ok {
			enumType = &schema.DBType{
				TypeName: enumName,
				Type:     schema.DataTypeEnum,
			}
			s.Types[enumName.String()] = enumType
		}
		enumType.EnumType = enum
		typ.EnumType = enum
	case schema.DataTypeRange:
		rangeName := schema.Identifier{
			Schema: dbtype.SchemaName,
			Name:   dbtype.TypeName,
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
		// TODO то есть если тип - композит, то он всегда ссылается на себя?
		compositeName := typ.TypeName
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

	return moreTypes, nil
}

func (p *Parser) LoadEnums(ctx context.Context, s *schema.Schema) error {
	enums, err := queries.QueryEnums(ctx, p.db.Conn, mapKeys(s.EnumTypes))
	if err != nil {
		p.log.Error("failed to query enums", zap.Error(err))
		return err
	}

	for _, dbenum := range enums {
		enumName := schema.Identifier{
			Schema: dbenum.SchemaName,
			Name:   dbenum.EnumName,
		}
		enum, ok := s.EnumTypes[enumName.String()]
		if !ok {
			return fmt.Errorf("enum %q not found", enumName.String())
		}
		enum.Values = dbenum.EnumValues
	}

	return nil
}

func mapKeys[V any](m map[string]V) (keys []string) {
	keys = make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
