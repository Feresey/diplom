package main

import (
	"sort"

	"github.com/Feresey/mtest/generate"
	"github.com/Feresey/mtest/schema"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
)

type CSVConverter struct{}

func (w *CSVConverter) ConvertRecords(table schema.Table, pr generate.Records) [][]string {
	res := make([][]string, 0, len(pr.Records)+1)
	res = append(res, maps.Keys(table.Columns))

	for _, precord := range pr.Records {
		record := w.partialToFullMap(precord, table.Columns)
		values := sortByKey(record)
		res = append(res, values)
	}

	return res
}

func (w *CSVConverter) partialToFullMap(
	p generate.Record,
	cols map[string]schema.Column,
) map[string]string {
	res := make(map[string]string, len(cols))
	for colname := range cols {
		res[colname] = ""
	}
	for idx, colNum := range p.Columns {
		res[colNum] = p.Values[idx]
	}
	return res
}

// func (w *CSVConverter) partialFromMap(cols map[int]string) generate.PartialRecord {
// 	var res generate.PartialRecord
// 	for colNum, value := range cols {
// 		if value == "" {
// 			continue
// 		}
// 		res.Columns = append(res.Columns, colNum)
// 		res.Values = append(res.Values, value)
// 	}

// 	return res
// }

func sortByKey[K constraints.Ordered, V any](m map[K]V) []V {
	keys := make([]K, 0, len(m))
	res := make([]V, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		res = append(res, m[k])
	}
	return res
}
