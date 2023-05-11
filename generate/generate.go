package generate

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/Feresey/mtest/schema"
)

type Generator struct {
	tables map[int]*schema.Table
	order  []*schema.Table

	log *zap.Logger
}

func New(
	log *zap.Logger,
	tables map[int]*schema.Table,
) (*Generator, error) {
	graph := schema.NewGraph(tables)
	order, err := graph.TopologicalSort()
	if err != nil {
		return nil, err
	}
	log.Info("tables insert order", zap.Ints("order", order))

	tablesOrdered := make([]*schema.Table, 0, len(order))
	for _, tableOID := range order {
		tablesOrdered = append(tablesOrdered, tables[tableOID])
	}

	g := &Generator{
		log:    log,
		tables: tables,
		order:  tablesOrdered,
	}
	return g, nil
}

type Records struct {
	Columns []string
	Values  [][]string
}

type CustomTableDomain struct {
	ColumnDomains map[string]Domain
}

// GenerateRecords генерирует данные по массиву проверок значений для каждой колонки.
// если для таблицы не указаны проверки значений, то они генерируются на лету из базовых.
func (g *Generator) GenerateRecords(
	partialRecords map[string]PartialRecords,
	domains map[string]CustomTableDomain,
) (map[string]*Records, error) {
	res := make(map[string]*Records, len(g.order))

	// порядок обхода, найденный топологической сортировкой
	for _, table := range g.order {
		// TODO генерить дефолтные значения нужно вне этой функции
		// если нет проверок для текущей таблицы, то будут дефолтные проверки.
		tablePartialRecords, ok := partialRecords[table.Name.String()]
		if !ok {
			g.log.Info("generate default checks for table", zap.Stringer("table", table.Name))
			checks := g.getDefaultTableChecks(table)
			// TODO configure mergeChecks
			tablePartialRecords = g.transformChecks(table, checks, true)
		}

		domain, ok := domains[table.Name.String()]
		if !ok {
			domain = CustomTableDomain{
				ColumnDomains: make(map[string]Domain),
			}
		}

		for _, col := range table.Columns {
			_, ok := domain.ColumnDomains[col.Name]
			if ok {
				continue
			}
			defaultDomain, err := g.DefaultDomain(col)
			if err != nil {
				return nil, err
			}
			domain.ColumnDomains[col.Name] = defaultDomain
		}

		tgen := newTableGenerator(table, domain)

		records, err := tgen.generateTableRecords(tablePartialRecords)
		if err != nil {
			return nil, err
		}
		res[table.Name.String()] = records
	}

	return res, nil
}

const (
	defaultTopDomainIterations = 1000
	defaultTopFloatDomain      = 10.0
	defaultStepFloatDomain     = 0.1
)

func (g *Generator) DefaultDomain(col *schema.Column) (Domain, error) {
	if col.Type.Type == schema.DataTypeEnum {
		return &EnumDomain{
			// TODO нужно ли делать явное приведение типов?
			values: col.Type.EnumType.Values,
		}, nil
	}

	typeName := col.Type.TypeName
	if typeName.Schema != "pg_catalog" {
		return nil, fmt.Errorf(
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
		i.ResetWith(defaultTopDomainIterations)
		return &i, nil
	case "float4", "float8", "numeric":
		var f FloatDomain
		f.ResetWith(numericToFloatDomainParams(
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
		t.ResetWith(time.Now(), defaultTopDomainIterations)
		return &t, nil
	default:
		return nil, fmt.Errorf(
			"unable to determine default domain for type %q. "+
				"table %q, column %q",
			typeName,
			col.Table.Name,
			col.Name,
		)
	}
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

type tableGenerator struct {
	table   *schema.Table
	domains CustomTableDomain

	// уникальные индексы таблицы
	uniqueIndexes map[int]*schema.Index
	// множество значений отдельных колонок
	// map[col_name]map[value]struct{}
	colValues map[string]mapset.Set[string]
	// для каждого уникального индекса показывает заполненные его значения
	// map[index_name]map[composite_value]struct{}
	uniqueIndexValues map[int]mapset.Set[string]
}

func newTableGenerator(
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
		table:   table,
		domains: domain,

		colValues:         make(map[string]mapset.Set[string], len(table.Columns)),
		uniqueIndexes:     uniqueIndexes,
		uniqueIndexValues: make(map[int]mapset.Set[string], len(uniqueIndexes)),
	}

	for indexName := range uniqueIndexes {
		t.uniqueIndexValues[indexName] = mapset.NewThreadUnsafeSet[string]()
	}
	for _, col := range table.Columns {
		t.colValues[col.Name] = mapset.NewThreadUnsafeSet[string]()
	}

	return t
}

func (g *tableGenerator) generateTableRecords(
	partialRecords PartialRecords,
) (records *Records, err error) {
	colNames := make([]string, 0, len(g.table.Columns))
	for _, col := range g.table.Columns {
		colNames = append(colNames, col.Name)
	}
	rev := make(map[string]int, len(g.table.Columns))
	for colNum, col := range g.table.Columns {
		rev[col.Name] = colNum
	}
	sort.Slice(colNames, func(i, j int) bool {
		return rev[colNames[i]] < rev[colNames[j]]
	})

	records = &Records{
		Columns: colNames,
	}

	// Для каждой полученной частичной записи надо догенерировать значения отсутствующих колонок
	for _, precord := range partialRecords {
		record, err := g.generateRecordValues(precord)
		if err != nil {
			return nil, err
		}

		// запись колонок в таком же порядке как и colNames
		recordOrdered := make([]string, 0, len(colNames))
		for _, colName := range colNames {
			recordOrdered = append(recordOrdered, record[colName])
		}
		records.Values = append(records.Values, recordOrdered)
	}

	// TODO для всех уникальных индексов
	/* только для пересекающихся групп индексов
	   для каждой колонки группы - счетчик индексов
	   если декартово произведение уникальных значений колонок уже содержится в каком-либо индексе
	   то исключаются все индексы, основанные на этих колонках
	*/

	return records, nil
}

func (g *tableGenerator) generateRecordValues(precord PartialRecord) (map[string]string, error) {
	// record соответствует полной записи
	// TODO генерируемые колонки - их надо пропускать почти всегда. Надо это настроить
	record := make(map[string]string, len(g.table.Columns))
	for idx, colName := range precord.Columns {
		record[colName] = precord.Values[idx]
	}

	for _, col := range g.table.Columns {
		if _, ok := record[col.Name]; ok {
			continue
		}

		domain, ok := g.domains.ColumnDomains[col.Name]
		if !ok {
			return nil, fmt.Errorf(
				"internal error: unable to find column domain for column %q for table %q",
				col.Name, g.table.Name)
		}

		if ok := g.generateAndCheckValue(col, domain, record); !ok {
			// TODO по идее по исчерпании домена надо текущую запись пропускать и продолжить
			return nil, fmt.Errorf(
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
	record map[string]string,
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
		record[col.Name] = value

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
	record map[string]string,
	index *schema.Index,
) string {
	fields := make([]string, 0, len(index.Columns))
	for _, col := range index.Columns {
		fields = append(fields, strconv.Quote(record[col.Name]))
	}
	return strings.Join(fields, ",")
}
