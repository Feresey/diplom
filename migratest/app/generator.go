package app

import "github.com/Feresey/diplom/migratest/schema"

type GeneratorConfig struct {
	ColumnsDataset map[string]map[string]string
	ExtraData      map[string]string
}

// map[column_name]column_value
type TableRecord map[string]string

type Generator struct {
	tables []*schema.Table
	m      *MultiGrapth

	// data - список записей в таблицах
	// map[table_name][]TableRecord
	data map[string][]TableRecord
}

func NewGenerator(tables []*schema.Table) *Generator {
	return &Generator{
		tables: tables,
		m:      BuildGrapth(tables),
		data:   make(map[string][]TableRecord),
	}
}

func (g *Generator) GenerateTablesData() map[string][]TableRecord {
	processedTables := make(map[int]bool, len(g.tables))

	var isAnyRootTable bool
	for isAnyRootTable {
		isAnyRootTable = false
		for tableIdx, tableRels := range g.m.Rels {
			if processedTables[tableIdx] {
				continue
			}

			var isRootTable bool
			for _, rel := range tableRels {
				if len(rel) != 0 {
					isRootTable = false
					break
				}
			}
			if !isRootTable {
				break
			}

			// корневая таблица нашлась, надо её заполнить
			isAnyRootTable = true

			tbl := g.m.TableValByIndex[tableIdx]
			g.generateTableData(tbl)
			processedTables[tableIdx] = true
		}
	}

	for _, isProcessed := range processedTables {
		if !isProcessed {
			// TODO
			panic("index cycle detected!")
		}
	}

	return g.data
}

// generateTableData генерирует данные для одной таблицы; для всех внешних ключей уже сгенерированы данные.
func (g *Generator) generateTableData(t *schema.Table) (res []TableRecord) {
	// Проход по всем колонкам, для каждой колонки может быть несколько вариаций.
	// Например TEXT - [ NULL, "some text" ].
	// Так покроются все случаи (ну да, большинство).

	// Однако внешних индексов может оказаться недостаточно,
	// например если будет внешний ключ UNIQUE NOT NULL и 20 колонок типа TEXT.
	// Тогда надо будет добавлять записи в таблицу на которую указывает внешний ключ.

	// Итого нужно 2 штуки.
	// - generateColumnData, которая сможет не только создавать данные, но и расширять уже существующие.
	//   мб добавить возможность указывать dataset для значений столюцов?
	//   С индексами на поля будет оч больно --- особенно с NULL-able.
	// - recursiveExtend, добавление данных в таблицу, на которую указывает внешний ключ (это если данных недостаточно).

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

	return res
}

// func (g *Generator) generateColumnData(columnName string, t *schema.Table, count int) []string {
// }
