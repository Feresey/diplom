package generate

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Feresey/mtest/schema"
)

// var BaseTypes = map[string]struct{}{
// 	"bool": {},

// 	"int2":    {},
// 	"int4":    {},
// 	"int8":    {},
// 	"float4":  {},
// 	"float8":  {},
// 	"numeric": {},

// 	"uuid":    {},
// 	"bytea":   {},
// 	"bit":     {},
// 	"varbit":  {},
// 	"char":    {},
// 	"varchar": {},
// 	"text":    {},

// 	"date":        {},
// 	"time":        {},
// 	"timetz":      {},
// 	"timestamp":   {},
// 	"timestamptz": {},
// }

// TODO это можно вынести как конфиг или типа луа код

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
	"date":        "datetime",
	"timestamp":   "datetime",
	"timestamptz": "datetime",
}

var Checks = map[string][]string{
	"bool": {"True", "False"},

	"int":  {"0", "-1", "1"},
	"int2": {strconv.Itoa(math.MaxInt16), strconv.Itoa(math.MinInt16)},
	"int4": {strconv.Itoa(math.MaxInt32), strconv.Itoa(math.MinInt32)},
	"int8": {strconv.Itoa(math.MaxInt64), strconv.Itoa(math.MinInt64)},
	// numeric типы с явно указанными precision и scale не могут хранить +-Inf
	"numeric": {"'NaN'::NUMERIC"},
	"float":   {"0", "'NaN'::REAL", "'infinity'::REAL", "'-infinity'::REAL"},

	// нет текстовых типов с длиной меньше 1, а нолик для любого текстового типа валидный (вроде)
	"text": {"''", "' '", "'0'"},

	"datetime": {
		"'epoch'::TIMESTAMP",
		"'infinity'::TIMESTAMP",
		"'-infinity'::TIMESTAMP",
	},
	"time": {
		"'allballs'::TIME",
	},
}

// GetDefaultChecks генерирует дефолтные проверки для всех таблиц.
func (g *Generator) GetDefaultChecks(tables []int) ([]PartialRecords, error) {
	res := make([]PartialRecords, 0, len(g.order))
	for _, tableOID := range tables {
		table, ok := g.tables[tableOID]
		if !ok {
			return nil, fmt.Errorf("table with oid %d not found", tableOID)
		}
		checks := g.getDefaultTableChecks(table)
		// TODO configure mergeChecks
		records := g.transformChecks(table, checks, true)
		res = append(res, records)
	}
	return res, nil
}

// getDefaultTableChecks генерирует дефолтные проверки для указанной таблицы.
func (g *Generator) getDefaultTableChecks(table *schema.Table) map[int]*ColumnChecks {
	checks := make(map[int]*ColumnChecks, len(table.Columns))

	foreignColumns := make(map[string]struct{}, len(table.Columns))
	for _, fk := range table.ForeignKeys {
		for _, col := range fk.Foreign.Columns {
			foreignColumns[col.Name] = struct{}{}
		}
	}

	for _, col := range table.Columns {
		check := &ColumnChecks{}
		attr := col.Attributes

		if !attr.NotNullable {
			check.AddValues("NULL")
		}

		// Если тип не является встроенным в postgresql, то я его не обрабатываю.
		if col.Type.TypeName.Schema != "pg_catalog" {
			continue
		}
		checks[col.ColNum] = check

		// Для FK колонок нельзя делать обычные проверки на значения, т.к. они зависят от других таблиц.
		if _, ok := foreignColumns[col.Name]; ok {
			continue
		}
		g.getTypeChecks(check, col.Type)

		// TODO numeric min max
		if col.Attributes.IsNumeric && col.Attributes.NumericPrecision == 0 {
			check.AddValues("'infinity'::NUMERIC", "'-infinity'::NUMERIC")
		}

		// Только если это текстовый тип и он имеет аттрибут CharMaxLength, то надо сгенерить строчку максимальной длины.
		// TODO нужно ли проверять на текстовость?
		if attr.HasCharMaxLength {
			check.AddValuesProcess(
				func(s string) string { return fmt.Sprintf("'%s'", s) },
				strings.Repeat(" ", attr.CharMaxLength),
				strings.Repeat("0", attr.CharMaxLength),
			)
		}
	}

	return checks
}

func (g *Generator) transformChecks(
	table *schema.Table,
	checks map[int]*ColumnChecks,
	mergeChecks bool,
) PartialRecords {
	res := PartialRecords{
		Table: table,
	}

	// Объединение проверок отдельных колонок в частичные записи.
	for colName, columnChecks := range checks {
		for _, value := range columnChecks.Values {
			pr := PartialRecord{
				Columns: []int{colName},
				Values:  []string{value},
			}
			if mergeChecks {
				res.MergeAdd(pr)
			} else {
				res.Records = append(res.Records, pr)
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
			return fmt.Sprintf("'%s'::%s", s, typ.EnumType.TypeName.String())
		}, typ.EnumType.Values...)
	case schema.DataTypeDomain:
		g.getTypeChecks(check, typ.DomainType.ElemType)
	case schema.DataTypeComposite,
		schema.DataTypeRange,
		schema.DataTypeMultiRange,
		schema.DataTypePseudo:
	default:
	}
}

func (g *Generator) baseTypesChecks(check *ColumnChecks, typeName string) {
	check.AddValues(Checks[Aliases[typeName]]...)
	check.AddValues(Checks[typeName]...)
}
