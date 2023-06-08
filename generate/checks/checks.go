package checks

import (
	_ "embed"
	"strings"

	"github.com/Feresey/mtest/schema"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/xerrors"
)

const (
	luaChecksFuncNameGetTableChecks = "get_table_checks"
)

type LuaChecks struct {
	l *lua.LState

	self           lua.LValue
	getTableChecks *lua.LFunction
}

func NewLuaChecks(
	l *lua.LState,
	value lua.LValue,
) (*LuaChecks, error) {
	ld := &LuaChecks{
		l:    l,
		self: value,
	}

	fn, err := ld.loadFunc(value, luaChecksFuncNameGetTableChecks)
	if err != nil {
		return nil, err
	}
	ld.getTableChecks = fn
	return ld, nil
}

func (l *LuaChecks) loadFunc(value lua.LValue, name string) (*lua.LFunction, error) {
	fn := l.l.GetField(value, name)
	if fn.Type() != lua.LTFunction {
		return nil, xerrors.Errorf(
			"lua object field %q of %+#q must be a function, but it is %s",
			name, value.String(), fn.Type())
	}
	return fn.(*lua.LFunction), nil
}

func (l *LuaChecks) GetTableChecks(table schema.Table) (map[string][]string, error) {
	err := l.l.CallByParam(lua.P{
		Fn:      l.getTableChecks,
		NRet:    1,
		Protect: true,
	}, l.self, table.ToLua(l.l))
	if err != nil {
		return nil, xerrors.Errorf("lua get table checks for table %q: %w", table, err)
	}
	res := make(map[string][]string)
	lres := l.l.ToTable(-1)
	if lres == nil {
		// empty is ok
		return res, nil
	}

	l.l.ForEach(lres, func(key, value lua.LValue) {
		llist, ok := value.(*lua.LTable)
		if !ok {
			// TODO log?
			return
		}
		list := make([]string, 0, llist.Len())
		ln := llist.Len()
		for i := 1; i <= ln; i++ {
			list = append(list, llist.RawGetInt(i).String())
		}
		res[key.String()] = list
	})

	return res, nil
}

//go:embed lua/checks.lua
var checksFile string

func RegisterModule(l *lua.LState) {
	l.PreloadModule("checks", func(l *lua.LState) int {
		fn, err := l.Load(strings.NewReader(checksFile), "checks.lua")
		if err != nil {
			l.Error(lua.LString(err.Error()), 9)
		}
		l.Push(fn)
		if err := l.PCall(0, lua.MultRet, nil); err != nil {
			l.Error(lua.LString(err.Error()), 9)
		}
		return 1
	})
}
