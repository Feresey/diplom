package generate

import "math"

var Aliases = map[string]string{
	"int2":      "int",
	"int4":      "int",
	"int8":      "int",
	"float4":    "float",
	"float8":    "float",
	"timestamp": "date",
}

var Checks = map[string][]any{
	"int":   {0, -1, 1},
	"int2":  {math.MaxInt16, math.MinInt16},
	"int4":  {math.MaxInt32, math.MinInt32},
	"int8":  {math.MaxInt64, math.MinInt64},
	"bool":  {true, false},
	"float": {math.NaN(), math.Inf(1), math.Inf(-1)},
	"date":  {"epoch", "infinity", "-infinity", "now", "today", "tomorrow", "yesterday"},
	"time":  {"now", "allballs"},
}
