package app

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"sort"

	"go.uber.org/zap"

	"github.com/Feresey/diplom/migratest/schema"
)

// GeneratorConfig содержит данные для настройки генерации данных для колонок.
type GeneratorConfig struct {
	// ColumnsDataset
	// мапа датасетов для таблиц -> мапа для колонки таблицы -> путь к файлу с построчно записанными данными
	ColumnsDataset map[string]map[string]string
	// ExtraData - по идее это данные, которые будут добавлены в таблицы помимо генерации данных.
	// В простом варианте можно отключить генерацию данных и указать явно данные для записи в таблицы.
	ExtraData map[string]string
	// SkipGenerateData - пропустить генерацию данных для таблиц. Не отменяет поведения ExtraData.
	SkipGenerateData []string
	// // MustGenerateData - отменяет SkipGenerateData.
	// MustGenerateData []string
}

func (gc *GeneratorConfig) Copy() GeneratorConfig {
	res := *gc
	res.ColumnsDataset = make(map[string]map[string]string, len(gc.ColumnsDataset))
	for key, val := range gc.ColumnsDataset {
		rres := make(map[string]string, len(val))
		for kkey, vval := range val {
			rres[kkey] = vval
		}
		res.ColumnsDataset[key] = rres
	}

	res.ExtraData = make(map[string]string, len(gc.ExtraData))
	for key, val := range gc.ExtraData {
		res.ExtraData[key] = val
	}

	res.SkipGenerateData = make([]string, len(gc.SkipGenerateData))
	copy(res.SkipGenerateData, gc.SkipGenerateData)

	return res
}

// MegeWith - слияние с другим конфигом. ВСЕ ДАННЫЕ КОПИРУЮТСЯ.
func (gc *GeneratorConfig) MergeWith(rc GeneratorConfig) {
	if gc.ColumnsDataset == nil {
		gc.ColumnsDataset = make(map[string]map[string]string)
	}
	for tableName, columnsMap := range rc.ColumnsDataset {
		currColumns := gc.ColumnsDataset[tableName]
		if currColumns == nil {
			currColumns = make(map[string]string)
		}
		// дозапись колонок в мапку
		for colName, colVal := range columnsMap {
			currColumns[colName] = colVal
		}

		gc.ColumnsDataset[tableName] = currColumns
	}

	for tableName, filename := range rc.ExtraData {
		gc.ExtraData[tableName] = filename
	}

	skipMap := make(map[string]struct{})
	for _, skip := range rc.SkipGenerateData {
		skipMap[skip] = struct{}{}
	}
	for _, skip := range gc.SkipGenerateData {
		skipMap[skip] = struct{}{}
	}

	// for _, skip := range rc.MustGenerateData {
	// 	delete(skipMap, skip)
	// }

	for skip := range skipMap {
		gc.SkipGenerateData = append(gc.SkipGenerateData, skip)
	}
	sort.Strings(gc.SkipGenerateData)
}

// MegreGeneratorConfigs - по идее конфиги для разных версий миграций будут очень похожи друг на друга,
// поэтому мне показалось хорошей идееей "расширять" новые конфиги за счёт старых.
func MegreGeneratorConfigs(cc []GeneratorConfig) GeneratorConfig {
	if len(cc) == 0 {
		return GeneratorConfig{}
	}
	res := cc[0]
	for _, cnf := range cc {
		res.MergeWith(cnf)
	}
	return res
}

// map[column_name]column_value
type TableRecord []interface{}

type TableRecords struct {
	TableName  string
	SchemaName string
	Columns    []string
	Records    []TableRecord
}

// Generator - сущность для генерации записей.
// Заполняет таблицы последовательно, от основных к вторичным.
// В зависимости от конфига изменяет данные для генерации или добавляет новые.
type Generator struct {
	logger *zap.Logger
	tables []*schema.Table
	m      *MultiGrapth

	gc GeneratorConfig

	// data - список записей в таблицах
	// map[table_name]TableRecords
	data map[string]TableRecords
}

func NewGenerator(logger *zap.Logger, tables []*schema.Table, gc GeneratorConfig) *Generator {
	return &Generator{
		logger: logger.Named("generator"),
		tables: tables,
		gc:     gc,
		m:      BuildGrapth(tables),
		data:   make(map[string]TableRecords),
	}
}

func (g *Generator) GenerateTablesData() (map[string]TableRecords, error) {
	// Я не знаю как называется этот алгоритм.

	// На каждой итерации ищется вершина (таблица), на которую никто не ссылается.
	// Если такой таблицы нет, то найдена циклическая связь.
	// Если такая таблица есть, то в неё записываются данные и в дальнейших итерациях она пропускается.
	// Таким образом таблицы обходятся как дерево.
	processedTables := make(map[int]bool, len(g.tables))

	var isAnyRootTable bool
	for isAnyRootTable {
		isAnyRootTable = false
		// tableRels - связи текущей таблицы с остальными.
		for tableIdx, tableRels := range g.m.Rels {
			if processedTables[tableIdx] {
				continue
			}

			var isRootTable bool
			// Очевидно что если связей нет, то таблица подходит.
			for _, rel := range tableRels {
				if len(rel) != 0 {
					isRootTable = false
					break
				}
			}
			if !isRootTable {
				break
			}

			// Корневая таблица нашлась, надо её заполнить.
			isAnyRootTable = true

			tbl := g.m.TableValByIndex[tableIdx]
			if err := g.generateTableData(tbl); err != nil {
				return nil, fmt.Errorf("generate records for table %s: %w", tbl.Name, err)
			}
			processedTables[tableIdx] = true
		}
	}

	for _, isProcessed := range processedTables {
		if !isProcessed {
			// TODO
			panic("index cycle detected!")
		}
	}

	return g.data, nil
}

// generateTableData генерирует данные для одной таблицы; для всех внешних ключей уже сгенерированы данные.
func (g *Generator) generateTableData(t *schema.Table) error {
	// TODO это костыльный метод - загрузка таблицы из файла

	rawData, ok := g.gc.ExtraData[t.Name]
	if !ok {
		g.logger.Info("extra data not found for table", zap.String("table", t.Name))
		return nil
	}

	file, err := os.Open(rawData)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			g.logger.Info("dataset for table not found", zap.String("filename", rawData), zap.String("table", t.Name))
			return nil
		}
		return fmt.Errorf("open table contents file: %w", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return fmt.Errorf("read file contents: %w", err)
	}

	res := TableRecords{
		TableName:  t.Name,
		SchemaName: t.SchemaName,
	}

	res.Columns = records[0]
	for _, vals := range records[1:] {
		rec := make(TableRecord, len(res.Columns))
		for idx, val := range vals {
			rec[idx] = val
		}
		res.Records = append(res.Records, rec)
	}

	g.data[t.Name] = res

	return nil

	// Проход по всем колонкам, для каждой колонки может быть несколько вариантов.
	// Например TEXT - [ NULL, "some text" ].
	// Так покроются все случаи (ну да, большинство).
	// Записи будут представлять собой все возможные комбинации из вариантов колонок.

	// Однако внешних индексов может оказаться недостаточно,
	// например если будет внешний ключ UNIQUE NOT NULL и 20 колонок типа TEXT.
	// Тогда надо будет добавлять записи в таблицу на которую указывает внешний ключ.

	// Итого нужно 2 штуки.
	// - generateColumnData, которая сможет не только создавать данные для колонки, но и расширять уже существующие.
	//   Мб добавить возможность указывать dataset для значений столбцов?
	//   С индексами на колонки будет оч больно --- особенно с NULL-able.
	// - recursiveExtend, добавление данных в таблицу, на которую указывает внешний ключ (это если данных недостаточно).
	//   каким-то образом надо сначала добавить данные во внешнюю таблицу, а потом в текущую.

	// TODO loadColumnDataset(filepath string) []string - опционально
	// TODO recursiveExtend(table string, count int)
	// TODO generateColumnData(t *schema.Table, column string, current []string, wantSize int) []string

	// var (
	// 	res              []TableRecord
	// 	processedColumns = make(map[string]bool, len(t.Columns))
	// 	// количество записей, которые будут добавлены в таблицу
	// 	maxVariations = 0
	// )

	// for _, col := range t.Columns {
	// 	curr := g.countMaxVariations(col, t)
	// 	if curr > maxVariations {
	// 		maxVariations = curr
	// 	}
	// }

	// for _, oneIdx := range t.ToOneRelationships {
	// 	columnData := g.generateColumn(oneIdx.Column, t)

	// 	newLen := len(columnData)
	// 	oldLen := len(res)
	// 	if oldLen < newLen {
	// 		res = append(res, make([]TableRecord, newLen-oldLen)...)
	// 	}

	// 	for colName, processed := range processedColumns {
	// 		if !processed {
	// 			continue
	// 		}

	// 		// дозапись старыми записями
	// 		for idx := range res[oldLen+1:] {
	// 		}
	// 	}
	// 	processedColumns[oneIdx.Column] = true
	// }
}

// func (g *Generator) generateColumnData(columnName string, t *schema.Table, count int) []string {
// }
