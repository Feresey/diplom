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
	// Индексы колонок
	Columns []int
	// Значения колонок
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

type PartialRecords struct {
	Records []PartialRecord
	Table   *schema.Table
}

// MergeAdd проходится по всем частичным записям и пытается дозаписать значения текущей частичной записи.
func (p *PartialRecords) MergeAdd(r PartialRecord) {
	sort.Sort(&r)

	record := p.searchNoOverlapRecord(r.Columns)
	if record == nil {
		p.Records = append(p.Records, r)
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
func (p *PartialRecords) checkNoOverlap(arr1, arr2 []int) bool {
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

func (p *PartialRecords) searchNoOverlapRecord(cols []int) *PartialRecord {
	for idx, r := range p.Records {
		if p.checkNoOverlap(r.Columns, cols) {
			return &p.Records[idx]
		}
	}
	return nil
}
