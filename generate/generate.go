package generate

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/Feresey/mtest/schema"
	mapset "github.com/deckarep/golang-set/v2"
)

const (
	defaultTopDomainIterations = 1000
	defaultTopFloatDomain      = 10.0
	defaultStepFloatDomain     = 0.1
)

type Generator struct {
	s     *schema.Schema
	order []schema.Table

	log *zap.Logger
}

func New(log *zap.Logger, s *schema.Schema) (*Generator, error) {
	graph := s.NewGraph()
	order, err := graph.TopologicalSort()
	if err != nil {
		return nil, err
	}

	tablesOrdered := make([]schema.Table, 0, len(order))
	for _, tableName := range order {
		tablesOrdered = append(tablesOrdered, s.Tables[tableName])
	}
	log.Debug("tables insert order", zap.Stringers("order", tablesOrdered))

	g := &Generator{
		log:   log,
		order: tablesOrdered,
		s:     s,
	}
	return g, nil
}

// type Records struct {
// 	Table  *schema.Table
// 	Values []map[int]string
// }

type CustomTableDomain struct {
	ColumnDomains map[int]Domain
}

type PartialRecords struct {
	Records map[string]Records
}

// GenerateRecords генерирует данные по массиву проверок значений для каждой колонки.
// если для таблицы не указаны проверки значений, то они генерируются на лету из базовых.
func (g *Generator) GenerateRecords(
	partial PartialRecords,
	domains map[int]CustomTableDomain,
) (res map[int]Records, warnings []error) {
	res = make(map[int]Records, len(g.order))

	// порядок обхода, найденный топологической сортировкой
tables:
	for _, table := range g.order {
		// TODO генерить дефолтные значения нужно вне этой функции
		// если нет проверок для текущей таблицы, то будут дефолтные проверки.
		tablePartialRecords, ok := partialRecords[table.OID()]
		if !ok {
			g.log.Info("generate default checks for table", zap.Stringer("table", table.Name))
			tablePartialRecords = g.GetDefaultChecks(table)
		}

		domain, ok := domains[table.OID()]
		if !ok {
			domain = CustomTableDomain{
				ColumnDomains: make(map[int]Domain),
			}
		}

		for _, col := range table.Columns {
			_, ok := domain.ColumnDomains[col.ColNum]
			if ok {
				continue
			}
			defaultDomain, err := g.DefaultDomain(col)
			if err != nil {
				warnings = append(warnings, err)
				g.log.Warn("get domain",
					zap.Stringer("table", table),
					zap.Stringer("column", col),
					zap.Stringer("type", col.Type),
					zap.Error(err),
				)
				continue tables
			}
			domain.ColumnDomains[col.ColNum] = defaultDomain
		}

		tgen := newTableGenerator(g.log, table, domain)

		records, err := tgen.generateTableRecords(tablePartialRecords)
		if err != nil {
			warnings = append(warnings, err)
			g.log.Warn("generate",
				zap.Stringer("table", table),
				zap.Error(err),
			)
			continue tables
		}
		res[table.OID()] = records
	}

	return res, warnings
}

func (g *Generator) DefaultDomain(col *schema.Column) (Domain, error) {
	if col.Type.Type == schema.DataTypeEnum {
		return &EnumDomain{
			// TODO нужно ли делать явное приведение типов?
			values: col.Type.EnumType.Values,
		}, nil
	}

	typeName := col.Type.TypeName
	if typeName.Schema != "pg_catalog" {
		return nil, xerrors.Errorf(
			"unable to determine default type domain for non-default postgres type %q. "+
				"table %q, column %q",
			typeName,
			col.Table.Name,
			col.Name,
		)
	}

	switch typeName.Name {
	case "bool":
		return BoolDomain(), nil
	case "int2", "int4", "int8":
		var i IntDomain
		i.Init(defaultTopDomainIterations)
		return &i, nil
	case "float4", "float8", "numeric":
		var f FloatDomain
		f.Init(numericToFloatDomainParams(
			col.Attributes.NumericPrecision,
			col.Attributes.NumericScale,
		))
		return &f, nil
	case "uuid",
		"bytea",
		"bit",
		"varbit",
		"char",
		"varchar",
		"text":
		return &UUIDDomain{}, nil
	case "date",
		"time",
		"timetz",
		"timestamp",
		"timestamptz":
		// TODO interval
		var t TimeDomain
		t.Init(time.Now(), defaultTopDomainIterations)
		return &t, nil
	default:
		return nil, xerrors.Errorf(
			"unable to determine default domain for type %q. "+
				"table %q, column %q",
			typeName,
			col.Table.Name,
			col.Name,
		)
	}
}

type tableGenerator struct {
	log     *zap.Logger
	table   *schema.Table
	domains CustomTableDomain

	// уникальные индексы таблицы
	uniqueIndexes map[int]*schema.Index
	// множество значений отдельных колонок
	// map[col_name]map[value]struct{}
	colValues map[int]mapset.Set[string]
	// для каждого уникального индекса показывает заполненные его значения
	// map[index_name]map[composite_value]struct{}
	uniqueIndexValues map[int]mapset.Set[string]
}

func newTableGenerator(
	log *zap.Logger,
	table *schema.Table,
	domain CustomTableDomain,
) *tableGenerator {
	uniqueIndexes := make(map[int]*schema.Index)
	for indexName, index := range table.Indexes {
		if index.IsUnique {
			uniqueIndexes[indexName] = index
		}
	}

	t := &tableGenerator{
		log:     log.With(zap.Stringer("table", table)),
		table:   table,
		domains: domain,

		colValues:         make(map[int]mapset.Set[string], len(table.Columns)),
		uniqueIndexes:     uniqueIndexes,
		uniqueIndexValues: make(map[int]mapset.Set[string], len(uniqueIndexes)),
	}

	for indexName := range uniqueIndexes {
		t.uniqueIndexValues[indexName] = mapset.NewThreadUnsafeSet[string]()
	}
	for _, col := range table.Columns {
		t.colValues[col.ColNum] = mapset.NewThreadUnsafeSet[string]()
	}

	return t
}

func (g *tableGenerator) generateTableRecords(
	partialRecords Records,
) (records Records, err error) {
	g.log.Debug("generate records")
	defer func() {
		g.log.Debug("generate records finished", zap.Error(err))
	}()

	// Для каждой полученной частичной записи надо догенерировать значения отсутствующих колонок
	for _, precord := range partialRecords.Records {
		vals, err := g.generateRecordValues(precord)
		if err != nil {
			return records, err
		}
		records.Records = append(records.Records, g.recordFromMap(vals))
	}

	// TODO для всех уникальных индексов
	/* только для пересекающихся групп индексов
	   для каждой колонки группы - счетчик индексов
	   если декартово произведение уникальных значений колонок уже содержится в каком-либо индексе
	   то исключаются все индексы, основанные на этих колонках
	*/

	return records, nil
}

func (g *tableGenerator) recordFromMap(m map[int]string) (res Record) {
	for colNum, value := range m {
		res.Columns = append(res.Columns, colNum)
		res.Values = append(res.Values, value)
	}
	sort.Sort(res)
	return res
}

func (g *tableGenerator) generateRecordValues(precord Record) (map[int]string, error) {
	// record соответствует полной записи
	// TODO генерируемые колонки - их надо пропускать почти всегда. Надо это настроить
	record := make(map[int]string, len(g.table.Columns))
	for idx, colName := range precord.Columns {
		record[colName] = precord.Values[idx]
	}

	for _, col := range g.table.Columns {
		if _, ok := record[col.ColNum]; ok {
			continue
		}

		domain, ok := g.domains.ColumnDomains[col.ColNum]
		if !ok {
			return nil, xerrors.Errorf(
				"internal error: unable to find column domain for column %q for table %q",
				col.Name, g.table.Name)
		}

		// TODO перебирать можно только заполненные записи
		if ok := g.generateAndCheckValue(col, domain, record); !ok {
			// TODO по идее по исчерпании домена надо текущую запись пропускать и продолжить
			return nil, xerrors.Errorf(
				"unable to generate values within expiration of domain. column %q, table %q",
				col.Name, g.table.Name)
		}
	}

	for indexName, index := range g.uniqueIndexes {
		key := g.concatIndexColumnsFromRecord(record, index)
		g.uniqueIndexValues[indexName].Add(key)
	}
	for colName, value := range record {
		g.colValues[colName].Add(value)
	}

	return record, nil
}

func (g *tableGenerator) generateAndCheckValue(
	col *schema.Column,
	domain Domain,
	record map[int]string,
) bool {
	domain.Reset()
domainLoop:
	for domain.Next() {
		// TODO add explicit type cast to result only if needed
		value := domain.Value()
		// FIXME +/-1 ?
		if col.Attributes.HasCharMaxLength && len(value) > col.Attributes.CharMaxLength {
			value = value[:col.Attributes.CharMaxLength]
		}
		record[col.ColNum] = value

		// TODO тут выделяется куча памяти
		for indexName, index := range g.uniqueIndexes {
			key := g.concatIndexColumnsFromRecord(record, index)
			if g.uniqueIndexValues[indexName].Contains(key) {
				continue domainLoop //nolint:gocritic // fp
			}
		}
	}

	return false
}

func (g *tableGenerator) concatIndexColumnsFromRecord(
	record map[int]string,
	index *schema.Index,
) string {
	fields := make([]string, 0, len(index.Columns))
	for _, col := range index.Columns {
		fields = append(fields, strconv.Quote(record[col.ColNum]))
	}
	return strings.Join(fields, ",")
}

func numericToFloatDomainParams(precision, scale int) (top, step float64) {
	if precision == 0 {
		return defaultTopFloatDomain, defaultStepFloatDomain
	}
	if scale == 0 {
		return math.Pow10(precision) - 1, 1
	}
	return math.Pow10(precision-scale) - math.Pow10(-scale), math.Pow10(-scale)
}
