package generate

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Feresey/mtest/schema"
	mapset "github.com/deckarep/golang-set/v2"
)

const pgCatalogPrefix = "pg_catalog."

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

type ColumnChecks struct {
	Values []string
}

func (c *ColumnChecks) AddValues(vals ...string) {
	c.Values = append(c.Values, vals...)
}

func (c *ColumnChecks) AddValuesProcess(f func(string) string, vals ...string) {
	for _, v := range vals {
		c.Values = append(c.Values, f(v))
	}
}

// GetDefaultChecks генерирует дефолтные проверки для всех таблиц.
func (g *Generator) GetDefaultChecks(table schema.Table) Records {
	checks := g.getDefaultTableChecks(table)
	// TODO configure mergeChecks
	records := g.transformChecks(checks, true)
	return records
}

// getDefaultTableChecks генерирует дефолтные проверки для указанной таблицы.
func (g *Generator) getDefaultTableChecks(table schema.Table) map[string]ColumnChecks {
	checks := make(map[string]ColumnChecks, len(table.Columns))

	foreignColumns := mapset.NewThreadUnsafeSet[string]()
	for _, fk := range table.ForeignKeys {
		foreignColumns.Append(fk.Constraint.Columns...)
	}

	for _, col := range table.Columns {
		checks[col.Name] = g.makeChecks(col, foreignColumns)
	}

	return checks
}

func (g *Generator) makeChecks(col schema.Column, foreignCols mapset.Set[string]) ColumnChecks {
	var check ColumnChecks
	attr := col.Attributes

	if !attr.NotNullable {
		check.AddValues("NULL")
	}

	// Если тип не является встроенным в postgresql, то я его не обрабатываю.
	if !strings.HasPrefix(col.Type.String(), pgCatalogPrefix) {
		return check
	}

	// Для FK колонок нельзя делать обычные проверки на значения, т.к. они зависят от других таблиц.
	if foreignCols.Contains(col.Name) {
		return check
	}
	g.getTypeChecks(&check, col.Type)

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

	return check
}

func (g *Generator) transformChecks(
	checks map[string]ColumnChecks,
	mergeChecks bool,
) (res Records) {
	// Объединение проверок отдельных колонок в частичные записи.
	for colName, columnChecks := range checks {
		for _, value := range columnChecks.Values {
			pr := Record{
				Columns: []string{colName},
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
	switch typ.TypType() {
	case schema.DataTypeBase:
		g.baseTypesChecks(check, strings.TrimPrefix(typ.String(), pgCatalogPrefix))
	case schema.DataTypeArray:
		// TODO нужно добавить кучу проверок на разные массивы, например для INT[][]:
		// [None], [None, None], [[None]], [], [[1],[2]], [[1],[None]], [[None], [None]]
		// dims := col.Attributes.ArrayDims
	case schema.DataTypeEnum:
		check.AddValuesProcess(func(s string) string {
			return fmt.Sprintf("'%s'::%s", s, typ.String())
		}, typ.EnumValues...)
	case schema.DataTypeDomain:
		// TODO domain attributes checks
		g.getTypeChecks(check, typ.ElemType)
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
