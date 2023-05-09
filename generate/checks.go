package generate

import (
	"fmt"
	"math"
	"strconv"

	"github.com/Feresey/mtest/schema"
)

// TODO это можно вынести как конфиг или типа луа код

var BaseTypes = map[string]struct{}{
	"bool": {},

	"int2":    {},
	"int4":    {},
	"int8":    {},
	"float4":  {},
	"float8":  {},
	"numeric": {},

	"uuid":    {},
	"bytea":   {},
	"bit":     {},
	"varbit":  {},
	"char":    {},
	"varchar": {},
	"text":    {},

	"date":        {},
	"time":        {},
	"timetz":      {},
	"timestamp":   {},
	"timestamptz": {},
}

var Aliases = map[string]string{
	"int2":   "int",
	"int4":   "int",
	"int8":   "int",
	"float4": "float",
	"float8": "float",

	"uuid":    "text",
	"bytea":   "text",
	"bit":     "text",
	"varbit":  "text",
	"char":    "text",
	"varchar": "text",

	"timetz":      "time",
	"timestamp":   "date",
	"timestamptz": "date",
}

// TODO add explicit type cast to result only if needed

var Checks = map[string][]string{
	"bool": {"True", "False"},

	"int":  {"0", "-1", "1"},
	"int2": {strconv.Itoa(math.MaxInt16), strconv.Itoa(math.MinInt16)},
	"int4": {strconv.Itoa(math.MaxInt32), strconv.Itoa(math.MinInt32)},
	"int8": {strconv.Itoa(math.MaxInt64), strconv.Itoa(math.MinInt64)},
	// numeric типы с явно указанными precision и scale не могут хранить +-Inf
	"numeric": {"0", "-1", "1", "'NaN'::NUMERIC"},
	"float":   {"0", "-1", "1", "'NaN'::REAL", "'infinity'::REAL", "'-infinity'::REAL"},

	// нет текстовых типов с длиной меньше 1, а нолик для любого текстового типа валидный (вроде)
	"text": {"", " ", "0"},

	"date": {
		"'epoch'::TIMESTAMP",
		"'infinity'::TIMESTAMP",
		"'-infinity'::TIMESTAMP",
		// TODO это разве ок?
		"'now'::TIMESTAMP",
		"'today'::TIMESTAMP",
		"'tomorrow'::TIMESTAMP",
		"'yesterday'::TIMESTAMP",
	},
	"time": {
		// TODO это разве ок?
		"'now'::TIME",
		"'allballs'::TIME",
	},
}

// GetDefaultChecks генерирует дефолтные проверки для всех таблиц
func (g *Generator) GetDefaultChecks() map[string]PartialRecords {
	res := make(map[string]PartialRecords, len(g.order))
	for _, tableName := range g.order {
		table := g.g.Schema.Tables[tableName]

		// TODO configurate mergeChecks
		res[tableName] = g.getDefaultTableChecks(table, true)
	}
	return res
}

// getDefaultTableChecks генерирует дефолтные проверки для указанной таблицы
func (g *Generator) getDefaultTableChecks(table *schema.Table, mergeChecks bool) PartialRecords {
	checks := make(map[string]*ColumnChecks, len(table.Columns))

	foreignColumns := make(map[string]struct{}, len(table.Columns))
	for _, fk := range table.ForeignKeys {
		for colName := range fk.Foreign.Columns {
			foreignColumns[colName] = struct{}{}
		}
	}

	for colName, col := range table.Columns {
		check := &ColumnChecks{}
		attr := col.Attributes
		typ := col.Type.TypeName.Name

		if !attr.NotNullable {
			check.AddValuesQuote("NULL")
		}

		// если тип не является встроенным в postgresql, то я его не обрабатываю
		if col.Type.TypeName.Schema != "pg_catalog" {
			continue
		}
		checks[colName] = check

		// Для FK колонок нельзя делать обычные проверки на значения, т.к. они зависят от других таблиц
		if _, ok := foreignColumns[colName]; ok {
			continue
		}
		g.getTypeChecks(check, col.Type)

		// Только если это текстовый тип и он имеет аттрибут CharMaxLength, то надо сгенерить строчку максимальной длины
		if typ == "text" || Aliases[typ] == "text" {
			if attr.HasCharMaxLength {
				check.AddValues(fmt.Sprintf("makestrlen(%d)::%s", attr.CharMaxLength, typ))
			}
		}
	}

	var res PartialRecords

	// объединение проверок отдельных колонок в частичные записи
	for colName, columnChecks := range checks {
		for _, value := range columnChecks.Values {
			pr := PartialRecord{
				Columns: []string{colName},
				Values:  []string{value},
			}
			if mergeChecks {
				res.MergeAdd(pr)
			} else {
				res = append(res, pr)
			}
		}
	}

	return res
}

func (g *Generator) getTypeChecks(check *ColumnChecks, typ *schema.DBType) {
	switch typ.Type {
	case schema.DataTypeBase:
		g.baseTypesChecks(check, typ.TypeName.Name)
	case schema.DataTypeArray:
		// TODO нужно добавить кучу проверок на разные массивы, например для INT[][]:
		// [None], [None, None], [[None]], [], [[1],[2]], [[1],[None]], [[None], [None]]
		// dims := col.Attributes.ArrayDims
	case schema.DataTypeEnum:
		check.AddValuesProcess(func(s string) string {
			return fmt.Sprintf("'%s'::%s", s, typ.EnumType.TypeName)
		}, typ.EnumType.Values...)
	case schema.DataTypeDomain:
		g.baseTypesChecks(check, typ.DomainType.ElemType.TypeName.Name)
	case schema.DataTypeComposite,
		schema.DataTypeRange,
		schema.DataTypeMultiRange,
		schema.DataTypePseudo:
	default:
	}
}

func (g *Generator) baseTypesChecks(check *ColumnChecks, typeName string) {
	check.AddValuesQuote(Checks[Aliases[typeName]]...)
	check.AddValuesQuote(Checks[typeName]...)
}
