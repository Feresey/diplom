package generate

import (
	"fmt"
	"strconv"
	"strings"

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

// GenerateRecords генерирует данные по массиву проверок значений для каждой колонки.
// если для таблицы не указаны проверки значений, то они генерируются на лету из базовых.
func (g *Generator) GenerateRecords(
	partialRecords map[string]PartialRecords,
) (map[string]*Records, error) {
	res := make(map[string]*Records, len(g.order))

	// порядок обхода, найденный топологической сортировкой
	for _, tableName := range g.order {
		table, ok := g.tables[tableName]
		if !ok {
			return nil, fmt.Errorf("table %q not found in schema", tableName)
		}
		// если нет проверок для текущей таблицы, то будут дефолтные проверки.
		tablePartialRecords, ok := partialRecords[tableName]
		if !ok {
			g.log.Info("generate default checks for table", zap.String("table", tableName))
			checks := g.cgen.getDefaultTableChecks(table)
			// TODO configure mergeChecks
			tablePartialRecords = g.cgen.transformChecks(checks, true)
		}

		// FIXME deretmine domains
		tgen := newTableGenerator(table, nil)

		records, err := tgen.generateTableRecords(tablePartialRecords)
		if err != nil {
			return nil, err
		}
		res[tableName] = records
	}

	return res, nil
}

type tableGenerator struct {
	table         *schema.Table
	columnsDomain map[string]Domain

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
	columnsDomain map[string]Domain,
) *tableGenerator {
	uniqueIndexes := make(map[string]*schema.Index)
	for indexName, index := range table.Indexes {
		if index.IsUnique {
			uniqueIndexes[indexName] = index
		}
	}

	return &tableGenerator{
		table:         table,
		columnsDomain: columnsDomain,

		colValues:         make(map[string]mapset.Set[string], len(table.Columns)),
		uniqueIndexes:     uniqueIndexes,
		uniqueIndexValues: make(map[string]mapset.Set[string], len(uniqueIndexes)),
	}
}

func (g *tableGenerator) processRecord(precord PartialRecord) (map[string]string, error) {
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
		// col, ok := table.Columns[colName]
		// if !ok {
		// 	return nil, fmt.Errorf("internal error: column %q not found for table %q", colName, table.Name)
		// }

		domain, ok := g.columnsDomain[colName]
		if !ok {
			return nil, fmt.Errorf(
				"internal error: unable to find column domain for column %q for table %q",
				colName, g.table.Name)
		}
		domain.Reset()

		var isValid bool
	domainLoop:
		for domain.Next() {
			// TODO сделать максимальное ограничение на количество итераций. Возможно через конфиг
			record[colName] = domain.Value()

			for indexName, index := range g.uniqueIndexes {
				key := g.concatIndexColumnsFromRecord(record, index)
				set, ok := g.uniqueIndexValues[indexName]
				if !ok {
					set = mapset.NewThreadUnsafeSet[string]()
					g.uniqueIndexValues[indexName] = set
				}

				if set.Add(key) {
					isValid = true
					break domainLoop
				}
			}
		}

		if !isValid {
			// TODO по идее по исчерпании домена надо текущую запись пропускать и продолжить
			return nil, fmt.Errorf(
				"unable to generate values within expiration of domain. column %q, table %q",
				colName, g.table.Name)
		}
	}

	return record, nil
}

func (g *tableGenerator) generateTableRecords(
	partialRecords PartialRecords,
) (records *Records, err error) {
	records = &Records{
		Columns: g.table.ColumnNames,
	}

	// Для каждой полученной частичной записи надо догенерировать значения отсутствующих колонок
	for _, precord := range partialRecords {
		record, err := g.processRecord(precord)
		if err != nil {
			return nil, err
		}

		recordOrdered := make([]string, 0, len(g.table.ColumnNames))
		for _, colName := range g.table.ColumnNames {
			value := record[colName]
			recordOrdered = append(recordOrdered, value)
			set, ok := g.colValues[colName]
			if !ok {
				set = mapset.NewThreadUnsafeSet[string]()
				g.colValues[colName] = set
			}
			set.Add(value)
		}
		records.Values = append(records.Values, recordOrdered)
	}

	return records, nil
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

// // getColumnSortedIndexes сопоставляет имя строки и её индекс в отсортированном массиве строк
// // map[column_name]index
// func getSortedValueIndexes(list []string) map[string]int {
// 	valueIndex := make(map[string]int, len(list))
// 	sorted := make([]string, len(list))
// 	copy(sorted, list)
// 	sort.Strings(sorted)
// 	for sortedIdx, colName := range sorted {
// 		valueIndex[colName] = sortedIdx
// 	}
// 	return valueIndex
// }

// func (g *Generator) addDuplicateValues(checks []*ColumnChecks, uniqueIndexes map[string]*schema.Index) {
// }

//nolint:lll // comment
// TODO для всех уникальных индексов
/* только для пересекающихся групп индексов
для каждой колонки группы - счетчик индексов
если декартово произведение уникальных значений колонок уже содержится в каком-либо индексе
то исключаются все индексы, основанные на этих колонках


1. для каждого индекса перебрать все возможные сочетания уникальных значений его колонок
2. каждое такое сочетание проверить на то что оно не нарушает этот индекс
3. если такого сочетания нет, то исключить этот индекс из перебора и перейти к шагу 1
4. если найдено такое сочетание то запомнить это сочетание и перейти к следующему индексу
5. для следующего индекса перебрать все возможные сочетания уникальных значений его колонок, за исключением выбранных колонок
6. если для следующего индекса такого сочетания нет, то вернуться к предыдущему индексу.
7. если предыдущего индекса нет, то исключить текущий индекс из перебора и перейти к шагу 1
8. если следующего индекса нет, выбранные значения колонок - собраны из значений колонок, которые уже есть в таблице.

*/
// for _, index := range uniqueIndexes {
// }
// // уникальные индексы таблицы
// uniqueIndexes := g.uniqueIndexes(table)
// // уникальные значения колонок таблицы
// // map[col_name]map[value]struct{}
// colValues := make(map[string]map[string]struct{}, len(table.Columns))
// // для каждого уникального индекса показывает заполненные его значения
// // map[index_name]map[composite_value]struct{}
// uniqueIndexValues := make(map[string]map[string]struct{}, len(uniqueIndexes))

// func (g *Generator) addDuplicateValues(checks []*ColumnChecks, uniqueIndexes map[string]*schema.Index) {
// }
