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

func (t *DBType) ToLua(l *lua.LState) *lua.LTable {
	typ := l.NewTable()

	typ.RawSetString("name", lua.LString(t.String()))
	typ.RawSetString("type", lua.LString(t.Type.String()))
	if t.ElemType != nil {
		typ.RawSetString("elem_type", t.ElemType.ToLua(l))
	}
	typ.RawSetString("enum_values", luaList(l, t.EnumValues))
	if t.DomainAttributes != nil {
		typ.RawSetString("attrs", t.DomainAttributes.ToLua(l))
	}

	return typ
}

func (t *Table) ToLua(l *lua.LState) *lua.LTable {
	table := l.NewTable()
	table.RawSetString("name", lua.LString(t.Name.String()))

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
	lc.RawSetString("columns", luaList(l, c.Columns))
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
	li.RawSetString("columns", luaList(l, i.Columns))
	return li
}

func luaList(l *lua.LState, arr []string) *lua.LTable {
	lc := l.NewTable()
	for _, value := range arr {
		lc.Append(lua.LString(value))
	}
	return lc
}

func (fk *ForeignKey) ToLua(l *lua.LState, table *Table) *lua.LTable {
	lf := l.NewTable()
	lf.RawSetString("reference", lua.LString(fk.ReferenceTable))
	lf.RawSetString("local_cols", luaList(l, fk.Constraint.Columns))
	lf.RawSetString("reference_cols", luaList(l, fk.ReferenceColumns))
	return lf
}

func (c *Column) ToLua(l *lua.LState) *lua.LTable {
	lc := l.NewTable()
	lc.RawSetString("name", lua.LString(c.Name))
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
	a.DomainAttributes.ToLuaTable(la)
	return la
}

func (a *DomainAttributes) ToLua(l *lua.LState) *lua.LTable {
	dtable := l.NewTable()
	a.ToLuaTable(dtable)
	return dtable
}

func (a *DomainAttributes) ToLuaTable(dt *lua.LTable) {
	dt.RawSetString("notnull", lua.LBool(a.NotNullable))
	if a.IsNumeric {
		dt.RawSetString("scale", lua.LNumber(a.NumericScale))
		dt.RawSetString("precision", lua.LNumber(a.NumericPrecision))
	}
	if a.HasCharMaxLength {
		dt.RawSetString("char_max_length", lua.LNumber(a.CharMaxLength))
	}
	if a.ArrayDims != 0 {
		dt.RawSetString("array_dims", lua.LNumber(a.ArrayDims))
	}
}
