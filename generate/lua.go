package generate

import (
	"fmt"
	"io"

	"github.com/Feresey/mtest/schema"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

const (
	luaDomainFuncNameInit  = "init"
	luaDomainFuncNameReset = "reset"
	luaDomainFuncNameNext  = "next"
	luaDomainFuncNameValue = "value"
)

type LuaDomain struct {
	l   *lua.LState
	log *zap.Logger

	init  lua.LValue
	reset lua.LValue
	next  lua.LValue
	value lua.LValue
}

func NewLuaDomain(
	log *zap.Logger,
	s *schema.Schema,
	source io.Reader, name string,
) (*LuaDomain, error) {
	l := lua.NewState()

	ld := &LuaDomain{
		l:   l,
		log: log.Named("lua-domain").With(zap.String("name", name)),
	}
	l.SetGlobal("schema", s.ToLua(l))

	compiled, err := l.Load(source, name)
	if err != nil {
		return nil, fmt.Errorf("lua source failed to compile: %w", err)
	}

	// загрузка скомпилированной функции в машину
	l.Push(compiled)
	// protected call. Не знаю почему, но так надо
	err = l.PCall(0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("load source code filter %w", err)
	}

	for _, v := range []struct {
		name  string
		value *lua.LValue
	}{
		{luaDomainFuncNameInit, &ld.init},
		{luaDomainFuncNameReset, &ld.reset},
		{luaDomainFuncNameNext, &ld.next},
		{luaDomainFuncNameValue, &ld.value},
	} {
		fn, err := ld.loadFunc(v.name)
		if err != nil {
			return nil, err
		}
		*v.value = fn
	}

	return ld, nil
}

func (l *LuaDomain) loadFunc(name string) (lua.LValue, error) {
	fn := l.l.GetGlobal(name)
	if fn.Type() != lua.LTFunction {
		return nil, fmt.Errorf(
			"lua filter must be a Lua function, but %s is %s",
			name, fn.Type().String())
	}
	return fn, nil
}

var _ Domain = (*LuaDomain)(nil)

func (l *LuaDomain) Init(table, column string) error {
	Lcol := l.l.NewUserData()
	Lcol.Value = column
	return l.l.CallByParam(lua.P{
		Fn:      l.init,
		NRet:    0,
		Protect: true,
	}, lua.LString(table), lua.LString(column))
}

func (l *LuaDomain) Reset() {
	err := l.l.CallByParam(lua.P{
		Fn:      l.reset,
		NRet:    0,
		Protect: true,
	})
	if err != nil {
		l.log.Error("apply method reset", zap.Error(err))
	}
}

func (l *LuaDomain) Next() bool {
	err := l.l.CallByParam(lua.P{
		Fn:      l.next,
		NRet:    1,
		Protect: true,
	})
	if err != nil {
		l.log.Error("apply method next", zap.Error(err))
		return false
	}
	next := l.l.ToBool(-1)
	l.l.Pop(1)
	return next
}

func (l *LuaDomain) Value() string {
	err := l.l.CallByParam(lua.P{
		Fn:      l.value,
		NRet:    1,
		Protect: true,
	})
	if err != nil {
		l.log.Error("apply method value", zap.Error(err))
		return err.Error()
	}
	value := l.l.ToString(-1)
	l.l.Pop(1)
	return value
}
