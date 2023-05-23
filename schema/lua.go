package schema

import (
	lua "github.com/yuin/gopher-lua"
)

func (s *Schema) ToLua(l *lua.LState) *lua.LTable {
	schema := l.NewTable()

	tables := l.NewTable()
	schema.RawSetString("tables", tables)
	for _, table := range s.Tables {
		tables.RawSetString(table.String(), table.ToLua(l))
	}

	return schema
}

// func (t *Type) ToLua(l *lua.LState) *lua.LTable {
// 	lt := l.NewTable()

// 	lt.RawSetString("type", lua.LString(t.Type.String()))
// 	switch tt := t.ConcreteType.(type) {
// 	case *ElemType:
// 		lt.RawSetString("elem_type", lua.LString(tt.ElemType.String()))
// 	case *DomainType:
// 		dat := l.NewTable()
// 		tt.Attributes.ToLuaTable(l, dat)
// 		lt.RawSetString("domain_attributes", dat)
// 		lt.RawSetString("elem_type", lua.LString(tt.ElemType.String()))
// 	case *EnumType:
// 		ev := l.NewTable()
// 		for _, value := range tt.Values {
// 			ev.Append(lua.LString(value))
// 		}
// 		lt.RawSetString("enum", ev)
// 	}

// 	return lt
// }

func (t *Table) ToLua(l *lua.LState) *lua.LTable {
	table := l.NewTable()
	if t.PrimaryKey != nil {
		table.RawSetString("pk", lua.LString(t.PrimaryKey.String()))
	}

	cols := l.NewTable()
	table.RawSetString("columns", cols)
	for _, col := range t.Columns {
		cols.RawSetString(col.Name, col.ToLua(l))
	}

	fks := l.NewTable()
	table.RawSetString("fk", fks)
	for _, fk := range t.ForeignKeys {
		fks.RawSetString(fk.String(), fk.ToLua(l, t))
	}

	indexes := l.NewTable()
	table.RawSetString("indexes", indexes)
	for _, index := range t.Indexes {
		indexes.RawSetString(index.String(), index.ToLua(l))
	}

	constraints := l.NewTable()
	table.RawSetString("constraints", constraints)
	for _, constraint := range t.Constraints {
		constraints.RawSetString(constraint.String(), constraint.ToLua(l))
	}

	return table
}

func (c *Constraint) ToLua(l *lua.LState) *lua.LTable {
	lc := l.NewTable()
	lc.RawSetString("type", lua.LString(c.Type.String()))
	lc.RawSetString("definition", lua.LString(c.Definition))
	lc.RawSetString("columns", luaColNames(l, c.Columns))
	if c.Index != nil {
		lc.RawSetString("index", lua.LString(c.Index.String()))
	}
	return lc
}

func (i *Index) ToLua(l *lua.LState) *lua.LTable {
	li := l.NewTable()
	li.RawSetString("definition", lua.LString(i.Definition))
	li.RawSetString("is_unique", lua.LBool(i.IsUnique))
	li.RawSetString("is_primary", lua.LBool(i.IsPrimary))
	li.RawSetString("is_nulls_not_distinct", lua.LBool(i.IsNullsNotDistinct))
	li.RawSetString("columns", luaColNames(l, i.Columns))
	return li
}

func luaColNames(l *lua.LState, cols []string) *lua.LTable {
	lc := l.NewTable()
	for _, col := range cols {
		lc.Append(lua.LString(col))
	}
	return lc
}

func (fk *ForeignKey) ToLua(l *lua.LState, table *Table) *lua.LTable {
	lf := l.NewTable()
	lf.RawSetString("reference", lua.LString(fk.ReferenceTable))
	lf.RawSetString("local_cols", luaColNames(l, fk.Constraint.Columns))
	lf.RawSetString("reference_cols", luaColNames(l, fk.ReferenceColumns))
	return lf
}

func (c *Column) ToLua(l *lua.LState) *lua.LTable {
	lc := l.NewTable()
	lc.RawSetString("type", c.Type.ToLua(l))
	lc.RawSetString("attr", c.Attributes.ToLua(l))
	return lc
}

func (a *ColumnAttributes) ToLua(l *lua.LState) *lua.LTable {
	la := l.NewTable()
	if a.HasDefault {
		if a.IsGenerated {
			la.RawSetString("generated", lua.LString(a.Default))
		} else {
			la.RawSetString("default", lua.LString(a.Default))
		}
	}
	a.DomainAttributes.ToLuaTable(l, la)
	return la
}

func (a *DomainAttributes) ToLuaTable(l *lua.LState, dt *lua.LTable) {
	dt.RawSetString("notnull", lua.LBool(a.NotNullable))
	dt.RawSetString("scale", lua.LNumber(a.NumericScale))
	dt.RawSetString("precision", lua.LNumber(a.NumericPrecision))
	dt.RawSetString("char_max_length", lua.LNumber(a.CharMaxLength))
	dt.RawSetString("array_dims", lua.LNumber(a.ArrayDims))
}
