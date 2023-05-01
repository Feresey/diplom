package parse

import (
	"context"
	"fmt"
	"sort"

	"go.uber.org/zap"

	"github.com/Feresey/mtest/config"
	"github.com/Feresey/mtest/schema"
	"github.com/Feresey/mtest/schema/db"
	"github.com/Feresey/mtest/schema/parse/queries"
)

type Parser struct {
	conn *db.DBConn
	log  *zap.Logger
}

func NewParser(
	conn *db.DBConn,
	log *zap.Logger,
) *Parser {
	return &Parser{
		log:  log.Named("parser"),
		conn: conn,
	}
}

// TODO вернуть ошибку если данные ссылаются на не указанную схему
// TODO как ограничивать внутри схемы таблицы, которые будут обрабатываться? Или на этом этапе это неважно?
func (p *Parser) LoadSchema(ctx context.Context, conf config.Parser) (*schema.Schema, error) {
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

	if err := p.LoadTables(ctx, s, conf); err != nil {
		return nil, fmt.Errorf("load tables: %w", err)
	}
	if err := p.LoadTablesColumns(ctx, s); err != nil {
		return nil, fmt.Errorf("load tables columns: %w", err)
	}
	if err := p.LoadConstraints(ctx, s); err != nil {
		return nil, fmt.Errorf("load constraints: %w", err)
	}

	constraintNames := mapKeys(s.Constraints)
	if err := p.LoadConstraintsColumns(ctx, s, constraintNames); err != nil {
		return nil, fmt.Errorf("load constraints columns: %w", err)
	}
	if err := p.LoadForeignConstraints(ctx, s, constraintNames); err != nil {
		return nil, fmt.Errorf("load foreign constraints: %w", err)
	}

	if err := p.LoadTypes(ctx, s); err != nil {
		return nil, fmt.Errorf("load types: %w", err)
	}
	if err := p.LoadEnums(ctx, s); err != nil {
		return nil, fmt.Errorf("load enums: %w", err)
	}
	// TODO load other types
	return s, nil
}

// LoadTables получает имена таблиц, найденных в схемах
func (p *Parser) LoadTables(ctx context.Context, s *schema.Schema, c config.Parser) error {
	tables, err := queries.QueryTables(ctx, p.conn.Conn, c)
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
		if len(s.Tables) != len(s.TableNames) {
			return fmt.Errorf("duplicated tables")
		}
	}

	return nil
}

// LoadTablesColumns загружает колонки таблиц, включая типы и аттрибуты
func (p *Parser) LoadTablesColumns(ctx context.Context, s *schema.Schema) error {
	columns, err := queries.QueryColumns(ctx, p.conn.Conn, s.TableNames)
	if err != nil {
		p.log.Error("failed to query tables columns", zap.Error(err))
		return err
	}
	p.log.Debug("columns loaded", zap.Reflect("columns", columns))

	for idx := range columns {
		dbcolumn := &columns[idx]
		tableName := schema.Identifier{
			Schema: dbcolumn.SchemaName,
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

		typ, ok := s.Types[typeName.String()]
		if !ok {
			typ = &schema.DBType{
				TypeName: typeName,
			}
			p.log.Debug("add type", zap.Stringer("type", typ.TypeName))
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
					NotNullable:      dbcolumn.IsNullable,
					CharMaxLength:    dbcolumn.CharacterMaxLength.Int,
					HasCharMaxLength: dbcolumn.CharacterMaxLength.Valid,
					ArrayDims:        dbcolumn.ArrayDims,
				},
			},
		}

		table.Columns[column.Name] = column
		table.ColumnNames = append(table.ColumnNames, column.Name)
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
	constraints, err := queries.QueryTableConstraints(ctx, p.conn.Conn, s.TableNames)
	if err != nil {
		p.log.Error("failed to query tables constraints", zap.Error(err))
		return err
	}
	p.log.Debug("loaded constraints", zap.Reflect("constraints", constraints))

	// realloc
	s.Constraints = make(map[string]*schema.Constraint, len(constraints))
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

		switch c.Type {
		case schema.ConstraintTypePK:
			// PRIMARY KEY всегда один
			table.PrimaryKey = c
		case schema.ConstraintTypeFK:
			table.ReferencedBy[c.Name.String()] = c
		}
	}

	return nil
}

func (p *Parser) LoadConstraintsColumns(ctx context.Context, s *schema.Schema, constraintNames []string) error {
	constraintsColumns, err := queries.QueryConstraintColumns(ctx, p.conn.Conn, s.TableNames, constraintNames)
	if err != nil {
		p.log.Error("failed to query constraints columns", zap.Error(err))
		return err
	}
	p.log.Debug("load constraints columns", zap.Reflect("constraints_columns", constraintsColumns))

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

func (p *Parser) LoadForeignConstraints(ctx context.Context, s *schema.Schema, constraintNames []string) error {
	fks, err := queries.QueryForeignKeys(ctx, p.conn.Conn, constraintNames)
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
	types, err := queries.QueryTypes(ctx, p.conn.Conn, typeNames)
	if err != nil {
		return nil, err
	}
	p.log.Debug("load types",
		zap.Strings("type_names", typeNames),
		zap.Reflect("types", types),
	)

	for idx := range types {
		dbtype := &types[idx]
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

// перевод значений колонки pg_type.typtype
var pgTypType = map[string]schema.DataType{
	"b": schema.DataTypeBase,
	"c": schema.DataTypeComposite,
	"d": schema.DataTypeDomain,
	"e": schema.DataTypeEnum,
	"r": schema.DataTypeRange,
	"m": schema.DataTypeMultiRange,
	"p": schema.DataTypePseudo,
}

func (p *Parser) fillType(
	s *schema.Schema,
	typ *schema.DBType,
	dbtype *queries.Type,
) (moreTypes []string, err error) {
	typType, ok := pgTypType[dbtype.TypeType]
	if !ok {
		return nil, fmt.Errorf("type value is undefined: %q", dbtype.TypeType)
	}
	if dbtype.IsArray {
		typType = schema.DataTypeArray
	}
	typ.Type = typType

	p.log.Debug("fill type",
		zap.Stringer("type", typ.TypeName),
		zap.Stringer("typ", typType),
	)

	switch typ.Type {
	default:
		return nil, fmt.Errorf("data type is undefined: %s", typ.Type)
	case schema.DataTypeBase:
	// TODO тут вроде ничего не надо делать, базовый же тип
	case schema.DataTypeMultiRange:
	// TODO что мне делать с multirange типом? где он вообще может использоваться?
	case schema.DataTypePseudo:
	// TODO что мне делать с pseudo типом? где он вообще может использоваться?
	case schema.DataTypeDomain:
		moreTypes, typ.DomainType, err = p.makeDomainType(s, typ.TypeName, dbtype)
	case schema.DataTypeArray:
		moreTypes, typ.ArrayType, err = p.makeArrayType(s, typ.TypeName, dbtype)
	case schema.DataTypeEnum:
		typ.EnumType, err = p.makeEnumType(s, typ.TypeName)
	case schema.DataTypeRange:
		moreTypes, typ.RangeType, err = p.makeRangeType(s, typ.TypeName, dbtype)
	case schema.DataTypeComposite:
		typ.CompositeType, err = p.makeCompositeType(s, typ.TypeName)
	}
	return moreTypes, err
}

func (p *Parser) makeDomainType(
	s *schema.Schema,
	domainTypeName schema.Identifier,
	dbtype *queries.Type,
) (needLoad []string, domain *schema.DomainType, err error) {
	domain, ok := s.DomainTypes[domainTypeName.String()]
	if ok {
		return nil, nil, fmt.Errorf("duplicate domain type: %q", domainTypeName)
	}
	defer func() {
		s.DomainTypes[domainTypeName.String()] = domain
	}()

	// Тип элемента, на котором основан домен
	elemTypeName := schema.Identifier{
		Schema: dbtype.DomainSchema.String,
		Name:   dbtype.DomainType.String,
	}
	elemType, ok := s.Types[elemTypeName.String()]
	if !ok {
		elemType = &schema.DBType{
			TypeName: elemTypeName,
		}
		s.Types[elemTypeName.String()] = elemType
		needLoad = append(needLoad, elemTypeName.String())
	}

	domain = &schema.DomainType{
		TypeName: domainTypeName,
		Attributes: schema.DomainAttributes{
			NotNullable:      !dbtype.DomainIsNotNullable,
			HasCharMaxLength: dbtype.DomainCharacterMaxSize.Valid,
			CharMaxLength:    dbtype.DomainCharacterMaxSize.Int,
			ArrayDims:        dbtype.DomainArrayDims,
		},
		ElemType: elemType,
	}

	return needLoad, domain, nil
}

//nolint:dupl // fp
func (p *Parser) makeArrayType(
	s *schema.Schema,
	arrayTypeName schema.Identifier,
	dbtype *queries.Type,
) (needLoad []string, array *schema.ArrayType, err error) {
	array, ok := s.ArrayTypes[arrayTypeName.String()]
	if ok {
		return nil, nil, fmt.Errorf("duplicate domain type: %q", arrayTypeName)
	}
	defer func() {
		s.ArrayTypes[arrayTypeName.String()] = array
	}()

	// Тип элемента массива
	elemTypeName := schema.Identifier{
		Schema: dbtype.ElemTypeSchema.String,
		Name:   dbtype.ElemTypeName.String,
	}
	elemType, ok := s.Types[elemTypeName.String()]
	if !ok {
		elemType = &schema.DBType{
			TypeName: elemTypeName,
		}
		s.Types[elemTypeName.String()] = elemType
		needLoad = append(needLoad, elemTypeName.String())
	}

	array = &schema.ArrayType{
		TypeName: arrayTypeName,
		ElemType: elemType,
	}

	return needLoad, array, nil
}

func (p *Parser) makeEnumType(
	s *schema.Schema,
	enumTypeName schema.Identifier,
) (enum *schema.EnumType, err error) {
	enum, ok := s.EnumTypes[enumTypeName.String()]
	if ok {
		return nil, fmt.Errorf("duplicate domain type: %q", enumTypeName)
	}
	defer func() {
		s.EnumTypes[enumTypeName.String()] = enum
	}()

	return &schema.EnumType{
		TypeName: enumTypeName,
	}, nil
}

//nolint:dupl // fp
func (p *Parser) makeRangeType(
	s *schema.Schema,
	rangeTypeName schema.Identifier,
	dbtype *queries.Type,
) (needLoad []string, rng *schema.RangeType, err error) {
	rng, ok := s.RangeTypes[rangeTypeName.String()]
	if ok {
		return nil, nil, fmt.Errorf("duplicate domain type: %q", rangeTypeName)
	}
	defer func() {
		s.RangeTypes[rangeTypeName.String()] = rng
	}()

	elemTypeName := schema.Identifier{
		Schema: dbtype.RangeElementTypeSchema.String,
		Name:   dbtype.RangeElementTypeName.String,
	}
	elemType, ok := s.Types[elemTypeName.String()]
	if !ok {
		elemType = &schema.DBType{
			TypeName: elemTypeName,
		}
		s.Types[elemTypeName.String()] = elemType
		needLoad = append(needLoad, elemTypeName.String())
	}

	rng = &schema.RangeType{
		TypeName: rangeTypeName,
		ElemType: elemType,
	}

	return needLoad, rng, nil
}

func (p *Parser) makeCompositeType(
	s *schema.Schema,
	compositeTypeName schema.Identifier,
) (composite *schema.CompositeType, err error) {
	composite, ok := s.CompositeTypes[compositeTypeName.String()]
	if ok {
		return nil, fmt.Errorf("duplicate domain type: %q", compositeTypeName)
	}
	defer func() {
		s.CompositeTypes[compositeTypeName.String()] = composite
	}()

	return &schema.CompositeType{
		TypeName: compositeTypeName,
		// TODO а что мне делать с композитами?
		Attributes: make(map[string]*schema.CompositeAttribute),
	}, nil
}

func (p *Parser) LoadEnums(ctx context.Context, s *schema.Schema) error {
	enums, err := queries.QueryEnums(ctx, p.conn.Conn, mapKeys(s.EnumTypes))
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
