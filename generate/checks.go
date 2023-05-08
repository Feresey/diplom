package generate

import (
	"math"
	"strconv"
)

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
		"'now'::TIMESTAMP",
		"'today'::TIMESTAMP",
		"'tomorrow'::TIMESTAMP",
		"'yesterday'::TIMESTAMP",
	},
	"time": {"'now'::TIME", "'allballs'::TIME"},
}
