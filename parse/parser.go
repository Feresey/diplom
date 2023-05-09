package parse

import (
	"context"
	"fmt"
	"sort"

	"go.uber.org/zap"

	"github.com/Feresey/mtest/db"
	"github.com/Feresey/mtest/parse/queries"
	"github.com/Feresey/mtest/schema"
)

type Config struct {
	Patterns []Pattern
}

type Pattern struct {
	Schema string
	Tables string
}

type Parser struct {
	conn queries.Executor
	log  *zap.Logger
	q    Queries
}

//go:generate mockery --name Queries --inpackage --testonly --with-expecter --quiet
type Queries interface {
	Tables(context.Context, queries.Executor, []queries.TablesPattern) ([]queries.Tables, error)
	Columns(context.Context, queries.Executor, []string) ([]queries.Column, error)
	Constraints(context.Context, queries.Executor, []string) ([]queries.Constraint, error)
	Types(context.Context, queries.Executor, []string) ([]queries.Type, error)
	Indexes(context.Context, queries.Executor, []string, []string) ([]queries.Index, error)
}

func NewParser(
	conn *db.Conn,
	log *zap.Logger,
) *Parser {
	return &Parser{
		log:  log.Named("parser"),
		conn: conn,
		q:    queries.Queries{},
	}
}

// TODO вернуть ошибку если данные ссылаются на не указанную схему
// TODO как ограничивать внутри схемы таблицы, которые будут обрабатываться?
// TODO Или на этом этапе это неважно?
func (p *Parser) LoadSchema(ctx context.Context, conf Config) (*schema.Schema, error) {
	s := &schema.Schema{
		Types:          make(map[string]*schema.DBType),
		ArrayTypes:     make(map[string]*schema.ArrayType),
		CompositeTypes: make(map[string]*schema.CompositeType),
		EnumTypes:      make(map[string]*schema.EnumType),
		RangeTypes:     make(map[string]*schema.RangeType),
		DomainTypes:    make(map[string]*schema.DomainType),
		Tables:         make(map[string]*schema.Table),
		Constraints:    make(map[string]*schema.Constraint),
		Indexes:        make(map[string]*schema.Index),
	}

	patterns := make([]queries.TablesPattern, 0, len(conf.Patterns))
	for _, p := range conf.Patterns {
		patterns = append(patterns, queries.TablesPattern(p))
	}

	if err := p.loadTables(ctx, s, patterns); err != nil {
		return nil, fmt.Errorf("load tables: %w", err)
	}
	if err := p.loadTablesColumns(ctx, s); err != nil {
		return nil, fmt.Errorf("load tables columns: %w", err)
	}
	if err := p.loadConstraints(ctx, s); err != nil {
		return nil, fmt.Errorf("load constraints: %w", err)
	}
	if err := p.loadIndexes(ctx, s); err != nil {
		return nil, fmt.Errorf("load indexes: %w", err)
	}

	if err := p.loadTypes(ctx, s); err != nil {
		return nil, fmt.Errorf("load types: %w", err)
	}
	return s, nil
}

// loadTables получает имена таблиц, найденных в схемах
func (p *Parser) loadTables(
	ctx context.Context,
	s *schema.Schema,
	patterns []queries.TablesPattern,
) error {
	tables, err := p.q.Tables(ctx, p.conn, patterns)
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
			Indexes:      make(map[string]*schema.Index),
			ReferencedBy: make(map[string]*schema.Constraint),
		}

		s.Tables[table.Name.String()] = table
		s.TableNames = append(s.TableNames, table.Name.String())
		if len(s.Tables) != len(s.TableNames) {
			return fmt.Errorf("duplicated tables: %q", table.Name)
		}
	}

	return nil
}

// loadTablesColumns загружает колонки таблиц, включая типы и аттрибуты
func (p *Parser) loadTablesColumns(ctx context.Context, s *schema.Schema) error {
	columns, err := p.q.Columns(ctx, p.conn, s.TableNames)
	if err != nil {
		p.log.Error("failed to query tables columns", zap.Error(err))
		return err
	}
	p.log.Debug("columns loaded", zap.Int("n", len(columns)))

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
				HasDefault:  dbcolumn.HasDefault,
				IsGenerated: dbcolumn.IsGenerated,
				Default:     dbcolumn.DefaultExpr.String,
				DomainAttributes: schema.DomainAttributes{
					NotNullable:      dbcolumn.IsNullable,
					HasCharMaxLength: dbcolumn.CharacterMaxLength.Valid,
					CharMaxLength:    int(dbcolumn.CharacterMaxLength.Int32),
					ArrayDims:        dbcolumn.ArrayDims,
					IsNumeric:        dbcolumn.IsNumeric,
					NumericPrecision: int(dbcolumn.NumericPriecision.Int32),
					NumericScale:     int(dbcolumn.NumericScale.Int32),
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

// loadConstraints загружает ограничения для всех найденных таблиц
func (p *Parser) loadConstraints(ctx context.Context, s *schema.Schema) error {
	constraints, err := p.q.Constraints(ctx, p.conn, s.TableNames)
	if err != nil {
		p.log.Error("failed to query tables constraints", zap.Error(err))
		return err
	}
	p.log.Debug("loaded constraints", zap.Int("n", len(constraints)))

	// realloc
	s.Constraints = make(map[string]*schema.Constraint, len(constraints))
	for idx := range constraints {
		if err := p.makeConstraint(s, &constraints[idx]); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) makeConstraint(
	s *schema.Schema,
	dbconstraint *queries.Constraint,
) error {
	constraintName := schema.Identifier{
		OID:    dbconstraint.ConstraintOID,
		Schema: dbconstraint.SchemaName,
		Name:   dbconstraint.ConstraintName,
	}

	_, ok := s.Constraints[constraintName.String()]
	if ok {
		return fmt.Errorf("duplicate constraint %q", constraintName)
	}

	tableName := schema.Identifier{
		OID:    dbconstraint.TableOID,
		Schema: dbconstraint.SchemaName,
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
		Name:       constraintName,
		Table:      table,
		Type:       typ,
		Definition: dbconstraint.ConstraintDef,
	}

	cols, err := p.checkTableColumns(dbconstraint.Columns, table)
	if err != nil {
		return fmt.Errorf("check table columns for constraint %q: %w", c.Name, err)
	}
	c.Columns = cols

	s.Constraints[constraintName.String()] = c
	table.Constraints[c.Name.String()] = c

	switch c.Type {
	case schema.ConstraintTypePK:
		// PRIMARY KEY либо один либо нет его
		table.PrimaryKey = c
		return nil
	case schema.ConstraintTypeFK:
		fk, err := p.makeForeignConstraint(s, c, dbconstraint)
		if err != nil {
			return err
		}

		table.ForeignKeys[c.Name.String()] = fk
	}

	return nil
}

func (p *Parser) checkTableColumns(
	columns []string,
	table *schema.Table,
) (map[string]*schema.Column, error) {
	cols := make(map[string]*schema.Column, len(columns))
	for _, column := range columns {
		tcol, ok := table.Columns[column]
		if !ok {
			return nil, fmt.Errorf("column %q not found in table %q", column, table.Name)
		}

		cols[column] = tcol
	}
	return cols, nil
}

func (p *Parser) makeForeignConstraint(
	s *schema.Schema,
	c *schema.Constraint,
	dbconstraint *queries.Constraint,
) (*schema.ForeignKey, error) {
	foreignTableName := schema.Identifier{
		OID:    int(dbconstraint.ForeignTableOID.Int32),
		Schema: dbconstraint.ForeignSchemaName.String,
		Name:   dbconstraint.ForeignTableName.String,
	}
	ftable, ok := s.Tables[foreignTableName.String()]
	if !ok {
		return nil, fmt.Errorf("foreign table %q not found for constraint %q", foreignTableName, c.Name)
	}
	ftable.ReferencedBy[c.Name.String()] = c

	fk := &schema.ForeignKey{
		Foreign:          c,
		Reference:        ftable,
		ReferenceColumns: make(map[string]*schema.Column, len(dbconstraint.ForeignColumns)),
	}

	columns := make([]string, 0, len(dbconstraint.ForeignColumns))
	for _, column := range dbconstraint.ForeignColumns {
		if !column.Valid {
			return nil, fmt.Errorf("null foreign column for constraint %q", c.Name)
		}
		columns = append(columns, column.String)
	}

	cols, err := p.checkTableColumns(columns, ftable)
	if err != nil {
		return nil, fmt.Errorf("check foreign table columns for constraint %q: %w", c.Name, err)
	}
	fk.ReferenceColumns = cols

	return fk, nil
}

func (p *Parser) loadIndexes(ctx context.Context, s *schema.Schema) error {
	indexes, err := p.q.Indexes(ctx, p.conn, s.TableNames, mapKeys(s.Constraints))
	if err != nil {
		p.log.Error("failed to query tables indexes", zap.Error(err))
		return err
	}
	p.log.Debug("loaded indexes", zap.Int("n", len(indexes)))

	// realloc
	s.Indexes = make(map[string]*schema.Index, len(indexes))
	for idx := range indexes {
		if err := p.makeIndex(s, &indexes[idx]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) makeIndex(s *schema.Schema, dbindex *queries.Index) error {
	indexName := schema.Identifier{
		OID:    dbindex.IndexOID,
		Schema: dbindex.IndexSchema,
		Name:   dbindex.IndexName,
	}
	tableName := schema.Identifier{
		OID:    dbindex.TableOID,
		Schema: dbindex.TableSchema,
		Name:   dbindex.TableName,
	}

	table, ok := s.Tables[tableName.String()]
	if !ok {
		return fmt.Errorf("table %q not found for index %q", tableName, indexName)
	}
	index, ok := s.Indexes[indexName.String()]
	if !ok {
		index = &schema.Index{
			Name:       indexName,
			Table:      table,
			Definition: dbindex.IndexDefinition,

			IsUnique:           dbindex.IsUnique,
			IsPrimary:          dbindex.IsPrimary,
			IsNullsNotDistinct: dbindex.IsNullsNotDistinct,
		}
		columns, err := p.checkTableColumns(dbindex.Columns, table)
		if err != nil {
			return fmt.Errorf("check table columns for index %q: %w", indexName, err)
		}
		index.Columns = columns
		s.Indexes[indexName.String()] = index
		table.Indexes[indexName.String()] = index
	}

	constraintName := schema.Identifier{
		OID:    int(dbindex.ConstraintOID.Int32),
		Schema: dbindex.ConstraintSchema.String,
		Name:   dbindex.ConstraintName.String,
	}
	var constraint *schema.Constraint
	if dbindex.ConstraintOID.Valid {
		constraint, ok = s.Constraints[constraintName.String()]
		if !ok {
			return fmt.Errorf("constraint %q not found fot index %q for table %q",
				constraintName,
				indexName,
				tableName,
			)
		}
	}
	if constraint != nil {
		constraint.Index = index
	}

	return nil
}

func (p *Parser) loadTypes(ctx context.Context, s *schema.Schema) error {
	var err error
	loadTypes := mapKeys(s.Types)

	for len(loadTypes) != 0 {
		loadTypes, err = p.loadTypesByNames(ctx, s, loadTypes)
		if err != nil {
			return fmt.Errorf("error loading types: %w", err)
		}
	}

	return nil
}

func (p *Parser) loadTypesByNames(
	ctx context.Context,
	s *schema.Schema,
	typeNames []string,
) (moreTypes []string, err error) {
	types, err := p.q.Types(ctx, p.conn, typeNames)
	if err != nil {
		return nil, err
	}
	p.log.Debug("types loaded",
		zap.Strings("type_names", typeNames),
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
	switch typ.Type {
	default:
		return nil, fmt.Errorf("data type is undefined: %s", typ.Type)
	case schema.DataTypeBase:
	case schema.DataTypeMultiRange:
	// TODO что мне делать с multirange типом? где он вообще может использоваться?
	case schema.DataTypePseudo:
	// TODO что мне делать с pseudo типом? где он вообще может использоваться?
	case schema.DataTypeDomain:
		moreTypes, typ.DomainType, err = p.makeDomainType(s, typ.TypeName, dbtype)
	case schema.DataTypeArray:
		moreTypes, typ.ArrayType, err = p.makeArrayType(s, typ.TypeName, dbtype)
	case schema.DataTypeEnum:
		typ.EnumType, err = p.makeEnumType(s, typ.TypeName, dbtype)
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
			CharMaxLength:    int(dbtype.DomainCharacterMaxSize.Int32),
			ArrayDims:        dbtype.DomainArrayDims,
			IsNumeric:        dbtype.DomainIsNumeric,
			NumericPrecision: int(dbtype.DomainNumericPrecision.Int32),
			NumericScale:     int(dbtype.DomainNumericScale.Int32),
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
	dbtype *queries.Type,
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
		Values:   dbtype.EnumValues,
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

func mapKeys[V any](m map[string]V) (keys []string) {
	keys = make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
