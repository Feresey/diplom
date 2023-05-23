package parse

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/Feresey/mtest/parse/query"
	"github.com/Feresey/mtest/schema"
)

type Config struct {
	Patterns []Pattern
}

type Pattern struct {
	Schema string
	Tables string
}

//go:generate mockery --name Queries --inpackage --testonly --with-expecter --quiet
type Queries interface {
	Tables(ctx context.Context, exec query.Executor, patterns []query.TablesPattern) ([]query.Table, error)
	Columns(ctx context.Context, exec query.Executor, tables []int) ([]query.Column, error)
	Constraints(ctx context.Context, exec query.Executor, tables []int) ([]query.Constraint, error)
	Types(ctx context.Context, exec query.Executor, types []int) ([]query.Type, error)
	Indexes(ctx context.Context, exec query.Executor, tables []int, constraints []int) ([]query.Index, error)
	Enums(ctx context.Context, exec query.Executor, enums []int) ([]query.Enum, error)
}

type Parser struct {
	conn query.Executor
	log  *zap.Logger
	q    Queries

	schema parseSchema
}

func NewParser(
	conn query.Executor,
	log *zap.Logger,
) *Parser {
	return &Parser{
		log:  log.Named("parser"),
		conn: conn,
		q:    query.Queries{},
		schema: parseSchema{
			typesLoadOrder: nil,
			typesByOID:     make(map[int]schema.DBType),
			types:          make(map[int]query.Type),

			enumList: nil,
			enums:    make(map[int]query.Enum),

			tables: make(map[int]parseTable),

			constraints:      make(map[int]query.Constraint),
			constraintsByOID: make(map[int]schema.Constraint),
			indexes:          make(map[int]query.Index),
		},
	}
}

// TODO вернуть ошибку если данные ссылаются на не указанную схему.
func (p *Parser) LoadSchema(ctx context.Context, conf Config) (*schema.Schema, error) {
	patterns := make([]query.TablesPattern, 0, len(conf.Patterns))
	for _, p := range conf.Patterns {
		patterns = append(patterns, query.TablesPattern(p))
	}

	if err := p.loadTables(ctx, patterns); err != nil {
		return nil, fmt.Errorf("load tables: %w", err)
	}
	tableOIDs := maps.Keys(p.schema.tables)
	if err := p.loadTablesColumns(ctx, tableOIDs); err != nil {
		return nil, fmt.Errorf("load tables columns: %w", err)
	}
	if err := p.loadConstraints(ctx, tableOIDs); err != nil {
		return nil, fmt.Errorf("load constraints: %w", err)
	}
	if err := p.loadIndexes(ctx, tableOIDs); err != nil {
		return nil, fmt.Errorf("load indexes: %w", err)
	}
	if err := p.loadTypes(ctx); err != nil {
		return nil, fmt.Errorf("load types: %w", err)
	}
	if err := p.loadEnums(ctx); err != nil {
		return nil, fmt.Errorf("load enums: %w", err)
	}
	return p.schema.convertToSchema()
}

// loadTables получает имена таблиц, найденных в схемах.
func (p *Parser) loadTables(
	ctx context.Context,
	patterns []query.TablesPattern,
) error {
	tables, err := p.q.Tables(ctx, p.conn, patterns)
	if err != nil {
		p.log.Error("failed to query tables", zap.Error(err))
		return err
	}
	p.log.Debug("loaded tables", zap.Reflect("tables", tables))

	// realloc
	p.schema.tables = make(map[int]parseTable, len(tables))
	for _, dbtable := range tables {
		table := parseTable{
			table:   dbtable,
			columns: make(map[int]query.Column),
		}

		p.schema.tables[dbtable.OID] = table
	}

	return nil
}

// loadTablesColumns загружает колонки таблиц, включая типы и аттрибуты.
func (p *Parser) loadTablesColumns(
	ctx context.Context,
	tableOIDs []int,
) error {
	columns, err := p.q.Columns(ctx, p.conn, tableOIDs)
	if err != nil {
		p.log.Error("failed to query tables columns", zap.Error(err))
		return err
	}
	p.log.Debug("columns loaded", zap.Int("n", len(columns)))

	for _, col := range columns {
		table, ok := p.schema.tables[col.TableOID]
		if !ok {
			err := fmt.Errorf("table with oid %d not found", col.TableOID)
			p.log.Error("failed to get table for column", zap.Error(err))
			return err
		}
		table.columns[col.ColumnNum] = col
	}
	return nil
}

// loadConstraints загружает ограничения для всех найденных таблиц.
func (p *Parser) loadConstraints(
	ctx context.Context,
	tableOIDs []int,
) error {
	cons, err := p.q.Constraints(ctx, p.conn, tableOIDs)
	if err != nil {
		p.log.Error("failed to query tables constraints", zap.Error(err))
		return err
	}
	p.log.Debug("loaded constraints", zap.Int("n", len(cons)))

	// realloc
	p.schema.constraints = make(map[int]query.Constraint, len(cons))
	for _, c := range cons {
		p.schema.constraints[c.ConstraintOID] = c
	}

	return nil
}

func (p *Parser) loadIndexes(
	ctx context.Context,
	tableOIDs []int,
) error {
	indexes, err := p.q.Indexes(ctx, p.conn, tableOIDs, maps.Keys(p.schema.constraints))
	if err != nil {
		p.log.Error("failed to query tables indexes", zap.Error(err))
		return err
	}
	p.log.Debug("loaded indexes", zap.Int("n", len(indexes)))

	for _, idx := range indexes {
		p.schema.indexes[idx.IndexOID] = idx
	}
	return nil
}

func (p *Parser) loadTypes(ctx context.Context) error {
	typeSet := make(map[int]struct{})
	for _, table := range p.schema.tables {
		for _, col := range table.columns {
			typeSet[col.TypeOID] = struct{}{}
		}
	}

	loadTypes := make([]int, 0, len(typeSet))
	for typ := range typeSet {
		loadTypes = append(loadTypes, typ)
	}

	for len(loadTypes) != 1 {
		p.schema.typesLoadOrder = append(p.schema.typesLoadOrder, loadTypes...)

		var err error
		loadTypes, err = p.loadTypesByOIDs(ctx, loadTypes)
		if err != nil {
			return fmt.Errorf("error loading types: %w", err)
		}
	}

	return nil
}

func (p *Parser) loadTypesByOIDs(
	ctx context.Context,
	typeOIDs []int,
) (moreTypes []int, err error) {
	types, err := p.q.Types(ctx, p.conn, typeOIDs)
	if err != nil {
		return nil, err
	}
	p.log.Debug("types loaded",
		zap.Ints("type_oids", typeOIDs),
	)

	for _, typ := range types {
		p.schema.types[typ.TypeOID] = typ

		typType, ok := pgTypType[typ.TypeType]
		if ok {
			if typType == schema.DataTypeEnum {
				p.schema.enumList = append(p.schema.enumList, typ.TypeOID)
			}
		}

		elemTypeOID := int(typ.ElemTypeOID.Int32)
		if _, ok := p.schema.types[elemTypeOID]; !ok {
			moreTypes = append(moreTypes, elemTypeOID)
		}

		rangeElemTypeOID := int(typ.RangeElementTypeOID.Int32)
		if _, ok := p.schema.types[rangeElemTypeOID]; !ok {
			moreTypes = append(moreTypes, rangeElemTypeOID)
		}

		domainTypeOID := int(typ.DomainTypeOID.Int32)
		if _, ok := p.schema.types[domainTypeOID]; !ok {
			moreTypes = append(moreTypes, domainTypeOID)
		}
	}

	return moreTypes, nil
}

func (p *Parser) loadEnums(ctx context.Context) error {
	enums, err := p.q.Enums(ctx, p.conn, p.schema.enumList)
	if err != nil {
		return fmt.Errorf("error loading enums: %w", err)
	}

	for _, e := range enums {
		p.schema.enums[e.TypeOID] = e
	}

	return nil
}
