package domains

import (
	_ "embed"

	"github.com/google/uuid"
	lua "github.com/yuin/gopher-lua"
	"golang.org/x/xerrors"
)

const (
	luaDomainFuncNameReset = "reset"
	luaDomainFuncNameNext  = "next"
)

type LuaDomain struct {
	l *lua.LState

	self  lua.LValue
	reset *lua.LFunction
	next  *lua.LFunction
}

func NewLuaDomain(
	l *lua.LState,
	value lua.LValue,
) (*LuaDomain, error) {
	ld := &LuaDomain{
		l:    l,
		self: value,
	}

	// mt := l.GetMetatable(value)
	// switch mt.Type() {
	// case lua.LTNil:
	// 	return nil, xerrors.New("lua domain has no metatable")
	// case lua.LTTable:
	// }

	// compiled, err := l.Load(source, name)
	// if err != nil {
	// 	return nil, xerrors.Errorf("lua source failed to compile %s: %w", name, err)
	// }

	// // загрузка скомпилированной функции в машину
	// l.Push(compiled)
	// // protected call. Не знаю почему, но так надо
	// err = l.PCall(0, 0, nil)
	// if err != nil {
	// 	return nil, xerrors.Errorf("load source code %s: %w", name, err)
	// }

	for _, v := range []struct {
		name  string
		value **lua.LFunction
	}{
		{luaDomainFuncNameReset, &ld.reset},
		{luaDomainFuncNameNext, &ld.next},
	} {
		fn, err := ld.loadFunc(value, v.name)
		if err != nil {
			return nil, err
		}
		*v.value = fn
	}

	return ld, nil
}

func (l *LuaDomain) loadFunc(value lua.LValue, name string) (*lua.LFunction, error) {
	fn := l.l.GetField(value, name)
	if fn.Type() != lua.LTFunction {
		return nil, xerrors.Errorf(
			"lua object field %q of %+#q must be a function, but it is %s",
			name, value.String(), fn.Type())
	}
	return fn.(*lua.LFunction), nil
}

func (l *LuaDomain) Reset() error {
	return l.l.CallByParam(lua.P{
		Fn:      l.reset,
		NRet:    0,
		Protect: true,
	}, l.self)
}

func (l *LuaDomain) Next() (value string, ok bool, err error) {
	err = l.l.CallByParam(lua.P{
		Fn:      l.next,
		NRet:    1,
		Protect: true,
	}, l.self)
	if err != nil {
		return "", false, err
	}
	ok = l.l.ToBool(-1)
	value = l.l.ToString(-1)
	l.l.Pop(1)
	return value, ok, nil
}

//go:embed lua/domains.lua
var domainsFile string

func RegisterModule(l *lua.LState) {
	l.RegisterModule("go_uuid", map[string]lua.LGFunction{
		"new": func(l *lua.LState) int {
			l.Push(lua.LString(uuid.NewString()))
			return 1
		},
	})
	l.Register("domains", func(l *lua.LState) int {
		if err := l.DoString(domainsFile); err != nil {
			l.Error(lua.LString(err.Error()), 9)
		}
		return 1
	})
}
