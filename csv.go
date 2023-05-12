package main

import (
	"sort"

	"github.com/Feresey/mtest/generate"
	"github.com/Feresey/mtest/schema"
)

type CSVConverter struct{}

func (w *CSVConverter) ConvertPartialRecords(pr generate.PartialRecords) [][]string {
	keys := w.makeKeys(pr.Table.Columns)
	res := make([][]string, 0, len(pr.Records)+1)
	res = append(res, keys)

	for _, precord := range pr.Records {
		record := w.partialToFullMap(precord, pr.Table.Columns)
		values := sortByKey(record)
		res = append(res, values)
	}

	return res
}

func (w *CSVConverter) ConvertRecords(pr generate.Records) [][]string {
	keys := w.makeKeys(pr.Table.Columns)
	res := make([][]string, 0, len(pr.Values)+1)
	res = append(res, keys)

	for _, record := range pr.Values {
		values := sortByKey(record)
		res = append(res, values)
	}

	return res
}

func (w *CSVConverter) makeKeys(m map[int]*schema.Column) []string {
	cols := sortByKey(m)
	res := make([]string, 0, len(cols))
	for _, col := range cols {
		res = append(res, col.String())
	}
	return res
}

func sortByKey[K int, V any](m map[K]V) []V {
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

func (w *CSVConverter) partialToFullMap(
	p generate.PartialRecord,
	cols map[int]*schema.Column,
) map[int]string {
	res := make(map[int]string, len(cols))
	for _, col := range cols {
		res[col.ColNum] = ""
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
