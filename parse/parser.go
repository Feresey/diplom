package parse

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/xerrors"

	"github.com/Feresey/mtest/parse/query"
	"github.com/Feresey/mtest/schema"
	mapset "github.com/deckarep/golang-set/v2"
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
			typesByOID: make(map[int]schema.DBType),
			types:      make(map[int]query.Type),

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
		return nil, xerrors.Errorf("load tables: %w", err)
	}
	tableOIDs := maps.Keys(p.schema.tables)
	slices.Sort(tableOIDs)

	if err := p.loadTablesColumns(ctx, tableOIDs); err != nil {
		return nil, xerrors.Errorf("load tables columns: %w", err)
	}
	if err := p.loadConstraints(ctx, tableOIDs); err != nil {
		return nil, xerrors.Errorf("load constraints: %w", err)
	}
	if err := p.loadIndexes(ctx, tableOIDs); err != nil {
		return nil, xerrors.Errorf("load indexes: %w", err)
	}
	if err := p.loadTypes(ctx); err != nil {
		return nil, xerrors.Errorf("load types: %w", err)
	}
	if err := p.loadEnums(ctx); err != nil {
		return nil, xerrors.Errorf("load enums: %w", err)
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
	p.log.Debug("loaded columns", zap.Int("n", len(columns)))

	for _, col := range columns {
		table, ok := p.schema.tables[col.TableOID]
		if !ok {
			return xerrors.Errorf("table with oid %d not found", col.TableOID)
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

	for _, c := range cons {
		p.schema.constraints[c.ConstraintOID] = c
	}

	conids := maps.Keys(p.schema.constraints)
	slices.Sort(conids)
	p.log.Debug("loaded constraints", zap.Int("n", len(cons)), zap.Ints("oids", conids))
	return nil
}

func (p *Parser) loadIndexes(
	ctx context.Context,
	tableOIDs []int,
) error {
	cons := maps.Keys(p.schema.constraints)
	slices.Sort(cons)
	indexes, err := p.q.Indexes(ctx, p.conn, tableOIDs, cons)
	if err != nil {
		p.log.Error("failed to query tables indexes", zap.Error(err))
		return err
	}

	for _, idx := range indexes {
		p.schema.indexes[idx.IndexOID] = idx
	}

	inds := maps.Keys(p.schema.indexes)
	slices.Sort(inds)
	p.log.Debug("loaded indexes", zap.Int("n", len(inds)), zap.Ints("oids", inds))
	return nil
}

func (p *Parser) loadTypes(ctx context.Context) error {
	typeSet := mapset.NewThreadUnsafeSet[int]()
	for _, table := range p.schema.tables {
		for _, col := range table.columns {
			typeSet.Add(col.TypeOID)
		}
	}

	for typeSet.Cardinality() != 0 {
		types := typeSet.ToSlice()
		slices.Sort(types)

		dbtypes, err := p.q.Types(ctx, p.conn, types)
		if err != nil {
			return err
		}
		p.log.Debug("loaded types", zap.Int("n", len(types)), zap.Ints("oids", types))

		typeSet.Clear()
		p.loadTypesByOIDs(dbtypes, typeSet)
	}

	return nil
}

func (p *Parser) loadTypesByOIDs(
	types []query.Type,
	typeSet mapset.Set[int],
) {
	for _, typ := range types {
		p.schema.types[typ.TypeOID] = typ

		typType, ok := pgTypType[typ.TypeType]
		if ok {
			if typType == schema.DataTypeEnum {
				p.schema.enumList = append(p.schema.enumList, typ.TypeOID)
				continue
			}
		}

		switch {
		case typ.ElemTypeOID.Valid:
			elemTypeOID := int(typ.ElemTypeOID.Int32)
			if _, ok := p.schema.types[elemTypeOID]; !ok {
				typeSet.Add(elemTypeOID)
			}
		case typ.RangeElementTypeOID.Valid:
			rangeElemTypeOID := int(typ.RangeElementTypeOID.Int32)
			if _, ok := p.schema.types[rangeElemTypeOID]; !ok {
				typeSet.Add(rangeElemTypeOID)
			}
		case typ.DomainTypeOID.Valid:
			domainTypeOID := int(typ.DomainTypeOID.Int32)
			if _, ok := p.schema.types[domainTypeOID]; !ok {
				typeSet.Add(domainTypeOID)
			}
		}
	}
}

func (p *Parser) loadEnums(ctx context.Context) error {
	enums, err := p.q.Enums(ctx, p.conn, p.schema.enumList)
	if err != nil {
		return xerrors.Errorf("error loading enums: %w", err)
	}

	for _, e := range enums {
		p.schema.enums[e.TypeOID] = e
	}

	enumIDs := maps.Keys(p.schema.indexes)
	slices.Sort(enumIDs)
	p.log.Debug("loaded enums", zap.Int("n", len(enums)), zap.Ints("oids", enumIDs))
	return nil
}
