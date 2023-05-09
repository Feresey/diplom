package generate

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/Feresey/mtest/schema"
)

type generatorBase struct {
	tables map[string]*schema.Table
	order  []string
}

type Generator struct {
	generatorBase

	log  *zap.Logger
	cgen *ChecksGenerator
}

func New(
	log *zap.Logger,
	graph *schema.Graph,
) (*Generator, error) {
	order, err := graph.TopologicalSort()
	if err != nil {
		return nil, err
	}
	log.Info("tables insert order", zap.Strings("order", order))

	base := generatorBase{
		tables: graph.Schema.Tables,
		order:  order,
	}

	g := &Generator{
		log:           log,
		generatorBase: base,
		cgen: &ChecksGenerator{
			base,
		},
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
	for _, tableName := range g.order {
		table, ok := g.tables[tableName]
		if !ok {
			return nil, fmt.Errorf("table %q not found in schema", tableName)
		}

		// TODO генерить дефолтные значения нужно вне этой функции
		// если нет проверок для текущей таблицы, то будут дефолтные проверки.
		tablePartialRecords, ok := partialRecords[tableName]
		if !ok {
			g.log.Info("generate default checks for table", zap.String("table", tableName))
			checks := g.cgen.getDefaultTableChecks(table)
			// TODO configure mergeChecks
			tablePartialRecords = g.cgen.transformChecks(checks, true)
		}

		domain, ok := domains[tableName]
		if !ok {
			domain = CustomTableDomain{
				ColumnDomains: make(map[string]Domain),
			}
		}

		for colName, col := range table.Columns {
			_, ok := domain.ColumnDomains[colName]
			if ok {
				continue
			}
			defaultDomain, err := g.DefaultDomain(col)
			if err != nil {
				return nil, err
			}
			domain.ColumnDomains[colName] = defaultDomain
		}

		tgen := newTableGenerator(table, domain)

		records, err := tgen.generateTableRecords(tablePartialRecords)
		if err != nil {
			return nil, err
		}
		res[tableName] = records
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
		f.ResetWith(NumericToFloatDomainParams(
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

func NumericToFloatDomainParams(precision, scale int) (top, step float64) {
	if precision == 0 {
		return defaultTopFloatDomain, defaultStepFloatDomain
	}

	if scale == 0 {
		top = math.Pow10(precision) - 1
		step = 1
	} else {
		top = math.Pow10(precision-scale) - math.Pow10(-scale)
		step = math.Pow10(-scale)
	}
	return
}

type tableGenerator struct {
	table   *schema.Table
	domains CustomTableDomain

	// уникальные индексы таблицы
	uniqueIndexes map[string]*schema.Index
	// множество значений отдельных колонок
	// map[col_name]map[value]struct{}
	colValues map[string]mapset.Set[string]
	// для каждого уникального индекса показывает заполненные его значения
	// map[index_name]map[composite_value]struct{}
	uniqueIndexValues map[string]mapset.Set[string]
}

func newTableGenerator(
	table *schema.Table,
	domain CustomTableDomain,
) *tableGenerator {
	uniqueIndexes := make(map[string]*schema.Index)
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
		uniqueIndexValues: make(map[string]mapset.Set[string], len(uniqueIndexes)),
	}

	for indexName := range uniqueIndexes {
		t.uniqueIndexValues[indexName] = mapset.NewThreadUnsafeSet[string]()
	}
	for colName := range table.Columns {
		t.colValues[colName] = mapset.NewThreadUnsafeSet[string]()
	}

	return t
}

func (g *tableGenerator) generateTableRecords(
	partialRecords PartialRecords,
) (records *Records, err error) {
	records = &Records{
		Columns: g.table.ColumnNames,
	}

	// Для каждой полученной частичной записи надо догенерировать значения отсутствующих колонок
	for _, precord := range partialRecords {
		record, err := g.generateRecordValues(precord)
		if err != nil {
			return nil, err
		}

		// запись колонок в таком же порядке как и g.table.ColumnNames
		recordOrdered := make([]string, 0, len(g.table.ColumnNames))
		for _, colName := range g.table.ColumnNames {
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

	for _, colName := range g.table.ColumnNames {
		if _, ok := record[colName]; ok {
			continue
		}
		col, ok := g.table.Columns[colName]
		if !ok {
			return nil, fmt.Errorf(
				"internal error: column %q not found for table %q",
				colName, g.table.Name)
		}

		domain, ok := g.domains.ColumnDomains[colName]
		if !ok {
			return nil, fmt.Errorf(
				"internal error: unable to find column domain for column %q for table %q",
				colName, g.table.Name)
		}

		if ok := g.generateAndCheckValue(col, domain, record); !ok {
			// TODO по идее по исчерпании домена надо текущую запись пропускать и продолжить
			return nil, fmt.Errorf(
				"unable to generate values within expiration of domain. column %q, table %q",
				colName, g.table.Name)
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
	for colName := range index.Columns {
		fields = append(fields, strconv.Quote(record[colName]))
	}
	return strings.Join(fields, ",")
}
