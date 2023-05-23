package generate

import (
	"sort"

	"golang.org/x/exp/constraints"
)

// Record описывает запись (возможно частичную)
// columns: ["id", "value"]
// values: ["0","a"]
type Record struct {
	// Имена колонок
	Columns []string
	// Значения колонок
	Values []string
	// // если это частичная запись, может ли она вливаться в другие записи
	// CanBeMerged bool
}

var _ sort.Interface = (*Record)(nil)

func (p Record) Len() int           { return len(p.Columns) }
func (p Record) Less(i, j int) bool { return p.Columns[i] < p.Columns[j] }
func (p Record) Swap(i, j int) {
	p.Columns[i], p.Columns[j] = p.Columns[j], p.Columns[i]
	p.Values[i], p.Values[j] = p.Values[j], p.Values[i]
}

type Records struct {
	Records []Record
}

// MergeAdd проходится по всем частичным записям и пытается дозаписать значения текущей частичной записи.
func (p *Records) MergeAdd(r Record) {
	sort.Sort(r)

	record := p.searchNoOverlapRecord(r.Columns)
	if record == nil {
		p.Records = append(p.Records, r)
		return
	}

	*record = p.merge(*record, r)
}

func (p *Records) merge(out, curr Record) Record {
	return Record{
		Columns: mergeTwoSortedArrays(out.Columns, curr.Columns),
		Values:  mergeTwoSortedArrays(out.Values, curr.Values),
	}
}

func (p *Records) searchNoOverlapRecord(cols []string) *Record {
	for idx, r := range p.Records {
		if checkNoOverlap(r.Columns, cols) {
			return &p.Records[idx]
		}
	}
	return nil
}

func mergeTwoSortedArrays[T constraints.Ordered](arr1, arr2 []T) []T {
	mergedArr := make([]T, 0, len(arr1)+len(arr2))
	i, j := 0, 0

	for i < len(arr1) && j < len(arr2) {
		if arr1[i] < arr2[j] {
			mergedArr = append(mergedArr, arr1[i])
			i++
		} else {
			mergedArr = append(mergedArr, arr2[j])
			j++
		}
	}

	if i < len(arr1) {
		mergedArr = append(mergedArr, arr1[i:]...)
	}

	if j < len(arr2) {
		mergedArr = append(mergedArr, arr2[j:]...)
	}

	return mergedArr
}

// checkNoOverlap проверяет что элементы отсортированных массивов arr1 и arr2 различны.
func checkNoOverlap[T constraints.Ordered](arr1, arr2 []T) bool {
	for i, j := 0, 0; i < len(arr1) && j < len(arr2); {
		switch {
		case arr1[i] < arr2[j]:
			i++
		case arr1[i] > arr2[j]:
			j++
		case arr1[i] == arr2[j]:
			return false
		}
	}
	return true
}
