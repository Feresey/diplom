package generate

import (
	"sort"

	"github.com/Feresey/mtest/schema"
)

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

type PartialRecord struct {
	// Имена колонок
	Columns []string
	// Матрица записей
	/*
		columns: ["id", "value"]
		values: ["0","a"]
	*/
	Values []string
}

// implement sort.Interface

func (p PartialRecord) Len() int           { return len(p.Columns) }
func (p PartialRecord) Less(i, j int) bool { return p.Columns[i] < p.Columns[j] }
func (p PartialRecord) Swap(i, j int) {
	p.Columns[i], p.Columns[j] = p.Columns[j], p.Columns[i]
	p.Values[i], p.Values[j] = p.Values[j], p.Values[i]
}

type sortRecordByNames struct {
	PartialRecord
	rev map[string]int
}

func (r *sortRecordByNames) Less(i, j int) bool {
	return r.rev[r.PartialRecord.Columns[i]] < r.rev[r.PartialRecord.Columns[j]]
}

func sortPartialByNames(pr PartialRecord, columns map[int]*schema.Column) {
	rev := make(map[string]int, len(pr.Columns))
	for colNum, col := range columns {
		rev[col.Name] = colNum
	}
	sort.Sort(&sortRecordByNames{rev: rev, PartialRecord: pr})
}

type PartialRecords []PartialRecord

// MergeAdd проходится по всем частичным записям и пытается дозаписать значения текущей частичной записи.
func (p *PartialRecords) MergeAdd(r PartialRecord) {
	sort.Sort(&r)

	record := p.searchNoOverlapRecord(r.Columns)
	if record == nil {
		*p = append(*p, r)
		return
	}

	p.merge(record, r)
}

func (p *PartialRecords) merge(out *PartialRecord, curr PartialRecord) {
	out.Columns = append(out.Columns, curr.Columns...)
	out.Values = append(out.Values, curr.Values...)
	sort.Sort(out)
}

// checkNoOverlap проверяет что элементы массивов arr1 и arr2 различны.
func (p *PartialRecords) checkNoOverlap(arr1, arr2 []string) bool {
	i := 0
	j := 0

	for i < len(arr1) && j < len(arr2) {
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

func (p *PartialRecords) searchNoOverlapRecord(cols []string) *PartialRecord {
	for idx, r := range *p {
		if p.checkNoOverlap(r.Columns, cols) {
			return &(*p)[idx]
		}
	}
	return nil
}
