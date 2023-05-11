package parse

import (
	"context"
	"fmt"
	"sort"

	"go.uber.org/zap"

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
	Columns(context.Context, queries.Executor, []int) ([]queries.Column, error)
	Constraints(context.Context, queries.Executor, []int) ([]queries.Constraint, error)
	Types(context.Context, queries.Executor, []int) ([]queries.Type, error)
	Indexes(context.Context, queries.Executor, []int, []int) ([]queries.Index, error)
	Enums(context.Context, queries.Executor, []int) ([]queries.Enum, error)
}

func NewParser(
	conn queries.Executor,
	log *zap.Logger,
) *Parser {
	return &Parser{
		log:  log.Named("parser"),
		conn: conn,
		q:    queries.Queries{},
	}
}

// TODO вернуть ошибку если данные ссылаются на не указанную схему.
func (p *Parser) LoadSchema(ctx context.Context, conf Config) (*schema.Schema, error) {
	s := &schema.Schema{
		Types:          make(map[int]*schema.DBType),
		ArrayTypes:     make(map[int]*schema.ArrayType),
		CompositeTypes: make(map[int]*schema.CompositeType),
		EnumTypes:      make(map[int]*schema.EnumType),
		RangeTypes:     make(map[int]*schema.RangeType),
		DomainTypes:    make(map[int]*schema.DomainType),
		Tables:         make(map[int]*schema.Table),
		Constraints:    make(map[int]*schema.Constraint),
		Indexes:        make(map[int]*schema.Index),
	}

	patterns := make([]queries.TablesPattern, 0, len(conf.Patterns))
	for _, p := range conf.Patterns {
		patterns = append(patterns, queries.TablesPattern(p))
	}

	if err := p.loadTables(ctx, s, patterns); err != nil {
		return nil, fmt.Errorf("load tables: %w", err)
	}
	tableOIDs := mapKeys(s.Tables)
	if err := p.loadTablesColumns(ctx, s, tableOIDs); err != nil {
		return nil, fmt.Errorf("load tables columns: %w", err)
	}
	if err := p.loadConstraints(ctx, s, tableOIDs); err != nil {
		return nil, fmt.Errorf("load constraints: %w", err)
	}
	if err := p.loadIndexes(ctx, s, tableOIDs); err != nil {
		return nil, fmt.Errorf("load indexes: %w", err)
	}
	if err := p.loadTypes(ctx, s); err != nil {
		return nil, fmt.Errorf("load types: %w", err)
	}
	if err := p.loadEnums(ctx, s); err != nil {
		return nil, fmt.Errorf("load types: %w", err)
	}
	return s, nil
}

// loadTables получает имена таблиц, найденных в схемах.
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
	s.Tables = make(map[int]*schema.Table, len(tables))
	for _, dbtable := range tables {
		table := &schema.Table{
			Name: schema.Identifier{
				OID:    dbtable.OID,
				Schema: dbtable.Schema,
				Name:   dbtable.Table,
			},
			Columns:      make(map[int]*schema.Column),
			Constraints:  make(map[int]*schema.Constraint),
			ForeignKeys:  make(map[int]*schema.ForeignKey),
			Indexes:      make(map[int]*schema.Index),
			ReferencedBy: make(map[int]*schema.Constraint),
		}

		s.Tables[table.OID()] = table
	}

	return nil
}

// loadTablesColumns загружает колонки таблиц, включая типы и аттрибуты.
func (p *Parser) loadTablesColumns(
	ctx context.Context,
	s *schema.Schema,
	tableOIDs []int,
) error {
	columns, err := p.q.Columns(ctx, p.conn, tableOIDs)
	if err != nil {
		p.log.Error("failed to query tables columns", zap.Error(err))
		return err
	}
	p.log.Debug("columns loaded", zap.Int("n", len(columns)))

	for idx := range columns {
		dbcolumn := &columns[idx]
		table, ok := s.Tables[dbcolumn.TableOID]
		if !ok {
			err := fmt.Errorf("table with oid %d not found", dbcolumn.TableOID)
			p.log.Error("failed to get table for column", zap.Error(err))
			return err
		}

		typ, ok := s.Types[dbcolumn.TypeOID]
		if !ok {
			typ = &schema.DBType{}
			p.log.Debug("add type", zap.Int("type", typ.OID()))
			s.Types[dbcolumn.TypeOID] = typ
		}

		column := &schema.Column{
			ColNum: dbcolumn.ColumnNum,
			Name:   dbcolumn.ColumnName,
			Table:  table,
			Type:   typ,
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

		table.Columns[column.ColNum] = column
	}

	return nil
}

// Перевод значений колонки pg_constraint.type.
var pgConstraintType = map[string]schema.ConstraintType{
	"p": schema.ConstraintTypePK,
	"f": schema.ConstraintTypeFK,
	"c": schema.ConstraintTypeCheck,
	"u": schema.ConstraintTypeUnique,
	"t": schema.ConstraintTypeTrigger,
	"x": schema.ConstraintTypeExclusion,
}

// loadConstraints загружает ограничения для всех найденных таблиц.
func (p *Parser) loadConstraints(
	ctx context.Context,
	s *schema.Schema,
	tableOIDs []int,
) error {
	constraints, err := p.q.Constraints(ctx, p.conn, tableOIDs)
	if err != nil {
		p.log.Error("failed to query tables constraints", zap.Error(err))
		return err
	}
	p.log.Debug("loaded constraints", zap.Int("n", len(constraints)))

	// realloc
	s.Constraints = make(map[int]*schema.Constraint, len(constraints))
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

	_, ok := s.Constraints[constraintName.OID]
	if ok {
		return fmt.Errorf("duplicate constraint %q", constraintName)
	}

	table, ok := s.Tables[dbconstraint.TableOID]
	if !ok {
		return fmt.Errorf("unable to find table with oid %d", dbconstraint.TableOID)
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

	cols, err := p.checkTableColumns(dbconstraint.Colnums, table)
	if err != nil {
		return fmt.Errorf("check table columns for constraint %q: %w", c.Name, err)
	}
	c.Columns = cols

	s.Constraints[constraintName.OID] = c
	table.Constraints[c.OID()] = c

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

		table.ForeignKeys[c.OID()] = fk
	}

	return nil
}

func (p *Parser) checkTableColumns(
	colnums []int,
	table *schema.Table,
) (map[int]*schema.Column, error) {
	cols := make(map[int]*schema.Column, len(colnums))
	for _, colnum := range colnums {
		tcol, ok := table.Columns[colnum]
		if !ok {
			return nil, fmt.Errorf("column %d not found in table %q", colnum, table)
		}

		cols[colnum] = tcol
	}
	return cols, nil
}

func (p *Parser) makeForeignConstraint(
	s *schema.Schema,
	c *schema.Constraint,
	dbconstraint *queries.Constraint,
) (*schema.ForeignKey, error) {
	fkoid := int(dbconstraint.ForeignTableOID.Int32)
	ftable, ok := s.Tables[fkoid]
	if !ok {
		return nil, fmt.Errorf("foreign table with oid %d not found for constraint %q", fkoid, c.Name)
	}
	ftable.ReferencedBy[c.OID()] = c

	fk := &schema.ForeignKey{
		Foreign:          c,
		Reference:        ftable,
		ReferenceColumns: make(map[int]*schema.Column, len(dbconstraint.ForeignColnums)),
	}

	cols, err := p.checkTableColumns(dbconstraint.ForeignColnums, ftable)
	if err != nil {
		return nil, fmt.Errorf("check foreign table columns for constraint %q: %w", c.Name, err)
	}
	fk.ReferenceColumns = cols

	return fk, nil
}

func (p *Parser) loadIndexes(
	ctx context.Context,
	s *schema.Schema,
	tableOIDs []int,
) error {
	indexes, err := p.q.Indexes(ctx, p.conn, tableOIDs, mapKeys(s.Constraints))
	if err != nil {
		p.log.Error("failed to query tables indexes", zap.Error(err))
		return err
	}
	p.log.Debug("loaded indexes", zap.Int("n", len(indexes)))

	// realloc
	s.Indexes = make(map[int]*schema.Index, len(indexes))
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

	table, ok := s.Tables[dbindex.TableOID]
	if !ok {
		return fmt.Errorf("table with oid %d not found for index %q", dbindex.TableOID, indexName)
	}
	index, ok := s.Indexes[indexName.OID]
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
		s.Indexes[indexName.OID] = index
		table.Indexes[indexName.OID] = index
	}

	if dbindex.ConstraintOID.Valid {
		s.Constraints[int(dbindex.ConstraintOID.Int32)].Index = index
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
	typeNames []int,
) (moreTypes []int, err error) {
	types, err := p.q.Types(ctx, p.conn, typeNames)
	if err != nil {
		return nil, err
	}
	p.log.Debug("types loaded",
		zap.Ints("type_names", typeNames),
	)

	for idx := range types {
		dbtype := &types[idx]
		typeName := schema.Identifier{
			OID:    dbtype.TypeOID,
			Schema: dbtype.TypeSchema,
			Name:   dbtype.TypeName,
		}

		typ, ok := s.Types[typeName.OID]
		if !ok {
			p.log.Error("internal error, type not found", zap.Int("oid", typeName.OID))
			return nil, fmt.Errorf("type %d not found", typeName.OID)
		}
		typ.TypeName = typeName

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

// Перевод значений колонки pg_type.typtype.
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
) (moreTypes []int, err error) {
	switch typ.Type {
	default:
		return nil, fmt.Errorf("data type is undefined: %s", typ.Type)
	case schema.DataTypeBase:
	case schema.DataTypeMultiRange:
	case schema.DataTypePseudo:
	case schema.DataTypeDomain:
		typ.DomainType = &schema.DomainType{
			TypeName: typ.TypeName,
			Attributes: schema.DomainAttributes{
				NotNullable:      !dbtype.DomainIsNotNullable,
				HasCharMaxLength: dbtype.DomainCharacterMaxSize.Valid,
				CharMaxLength:    int(dbtype.DomainCharacterMaxSize.Int32),
				ArrayDims:        dbtype.DomainArrayDims,
				IsNumeric:        dbtype.DomainIsNumeric,
				NumericPrecision: int(dbtype.DomainNumericPrecision.Int32),
				NumericScale:     int(dbtype.DomainNumericScale.Int32),
			},
		}
		moreTypes = fillElemType(s.Types, s.DomainTypes, typ.DomainType, int(dbtype.DomainTypeOID.Int32))
	case schema.DataTypeArray:
		typ.ArrayType = &schema.ArrayType{TypeName: typ.TypeName}
		moreTypes = fillElemType(s.Types, s.ArrayTypes, typ.ArrayType, int(dbtype.ElemTypeOID.Int32))
	case schema.DataTypeEnum:
		typ.EnumType = &schema.EnumType{TypeName: typ.TypeName}
		s.EnumTypes[typ.OID()] = typ.EnumType
	case schema.DataTypeRange:
		typ.RangeType = &schema.RangeType{TypeName: typ.TypeName}
		moreTypes = fillElemType(s.Types, s.RangeTypes, typ.RangeType, int(dbtype.RangeElementTypeOID.Int32))
	case schema.DataTypeComposite:
		typ.CompositeType = &schema.CompositeType{TypeName: typ.TypeName}
		s.CompositeTypes[typ.OID()] = typ.CompositeType
	}
	return moreTypes, err
}

func (p *Parser) loadEnums(ctx context.Context, s *schema.Schema) error {
	enumOIDs := mapKeys(s.EnumTypes)

	enums, err := p.q.Enums(ctx, p.conn, enumOIDs)
	if err != nil {
		return fmt.Errorf("error loading enums: %w", err)
	}

	for _, dbenum := range enums {
		enum, ok := s.EnumTypes[dbenum.TypeOID]
		if !ok {
			return fmt.Errorf("enum with oid %q not found", dbenum.TypeOID)
		}
		enum.Values = dbenum.Values
	}

	return nil
}

func fillElemType[T schema.Elementer](
	types map[int]*schema.DBType,
	elemMap map[int]T,
	base T,
	elemTypeOID int,
) (needLoad []int) {
	// Тип элемента, на котором основан тип
	elem, ok := types[elemTypeOID]
	if !ok {
		elem = &schema.DBType{}
		types[elemTypeOID] = elem
		needLoad = append(needLoad, elemTypeOID)
	}
	base.SetElemType(elem)
	elemMap[base.OID()] = base
	return needLoad
}

func mapKeys[K ~string | ~int, V any](m map[K]V) (keys []K) {
	keys = make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
