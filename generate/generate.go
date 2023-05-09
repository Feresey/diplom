package generate

import (
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/Feresey/mtest/schema"
)

type Generator struct {
	log   *zap.Logger
	g     *schema.Graph
	order []string
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

	g := &Generator{
		log:   log,
		g:     graph,
		order: order,
	}
	return g, nil
}

type Records struct {
	Columns []string
	Values  [][]string
}

// GenerateData генерирует данные по массиву проверок значений для каждой колонки.
// если для таблицы не указаны проверки значений, то они генерируются на лету из базовых.
func (g *Generator) GenerateData(partialRecords map[string]PartialRecords) (map[string]*Records, error) {
	res := make(map[string]*Records, len(g.order))

	// порядок обхода, найденный топологической сортировкой
	for _, tableName := range g.order {
		table, ok := g.g.Schema.Tables[tableName]
		if !ok {
			return nil, fmt.Errorf("table %q not found in schema", tableName)
		}
		// если нет проверок для текущей таблицы, то будут дефолтные проверки.
		tablePartialRecords, ok := partialRecords[tableName]
		if !ok {
			g.log.Info("generate default checks for table", zap.String("table", tableName))
			// TODO configure mergeChecks
			tablePartialRecords = g.getDefaultTableChecks(table, true)
		}

		records, err := g.generateTableRecords(table, tablePartialRecords)
		if err != nil {
			// TODO что тут могут быть за ошибки?
			return nil, err
		}
		res[tableName] = records
	}

	return res, nil
}

func (g *Generator) generateTableRecords(
	table *schema.Table,
	columnsDomain map[string]Domain,
	partialRecords PartialRecords,
) (records *Records, err error) {
	records = &Records{
		Columns: table.ColumnNames,
	}

	// уникальные индексы таблицы
	uniqueIndexes := g.uniqueIndexes(table)
	// множество значений отдельных колонок
	// map[col_name]map[value]struct{}
	colValues := make(map[string]mapset.Set[string], len(table.Columns))
	// для каждого уникального индекса показывает заполненные его значения
	// map[index_name]map[composite_value]struct{}
	uniqueIndexValues := make(map[string]mapset.Set[string], len(uniqueIndexes))

	// Для каждой полученной частичной записи надо догенерировать значения отсутствующих колонок
	for _, precord := range partialRecords {
		// record соответствует полной записи
		// TODO генерируемые колонки - их надо пропускать почти всегда. Надо это настроить
		record := make(map[string]string, len(table.Columns))
		for idx, colName := range precord.Columns {
			record[colName] = precord.Values[idx]
		}

		for _, colName := range table.ColumnNames {
			if _, ok := record[colName]; ok {
				continue
			}
			// col, ok := table.Columns[colName]
			// if !ok {
			// 	return nil, fmt.Errorf("internal error: column %q not found for table %q", colName, table.Name)
			// }

			domain, ok := columnsDomain[colName]
			if !ok {
				return nil,
					fmt.Errorf("internal error: unable to find column domain for column %q for table %q",
						colName, table.Name)
			}
			domain.Reset()

			var isValid bool
		domainLoop:
			for domain.Next() {
				// TODO сделать максимальное ограничение на количество итераций. Возможно через конфиг
				record[colName] = domain.Value()

				for indexName, index := range uniqueIndexes {
					key := g.concatIndexColumnsFromRecord(record, index)
					set, ok := uniqueIndexValues[indexName]
					if !ok {
						set = mapset.NewThreadUnsafeSet[string]()
						uniqueIndexValues[indexName] = set
					}

					if set.Add(key) {
						isValid = true
						break domainLoop
					}
				}
			}

			if !isValid {
				// TODO по идее по исчерпании домена надо текущую запись пропускать и продолжить
				return nil,
					fmt.Errorf("unable to generate values within expiration of domain. column %q, table %q",
						colName, table.Name)
			}

			recordOrdered := make([]string, 0, len(table.ColumnNames))
			for _, colName := range table.ColumnNames {
				value := record[colName]
				recordOrdered = append(recordOrdered, value)
				set, ok := colValues[colName]
				if !ok {
					set = mapset.NewThreadUnsafeSet[string]()
					colValues[colName] = set
				}
				set.Add(value)
			}
			records.Values = append(records.Values, recordOrdered)
		}
	}

	return records, nil
}

func (g *Generator) concatIndexColumnsFromRecord(
	record map[string]string,
	index *schema.Index,
) string {
	fields := make([]string, 0, len(index.Columns))
	for colName := range index.Columns {
		fields = append(fields, strconv.Quote(record[colName]))
	}
	return strings.Join(fields, ",")
}

// uniqueIndexes возвращает только уникальные индексы таблицы
func (g *Generator) uniqueIndexes(table *schema.Table) map[string]*schema.Index {
	uniqueIndexes := make(map[string]*schema.Index)
	for indexName, index := range table.Indexes {
		if index.IsUnique {
			uniqueIndexes[indexName] = index
		}
	}
	return uniqueIndexes
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
