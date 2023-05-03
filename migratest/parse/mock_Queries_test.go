// Code generated by mockery v2.26.1. DO NOT EDIT.

package parse

import (
	context "context"

	queries "github.com/Feresey/mtest/parse/queries"
	mock "github.com/stretchr/testify/mock"
)

// MockQueries is an autogenerated mock type for the Queries type
type MockQueries struct {
	mock.Mock
}

type MockQueries_Expecter struct {
	mock *mock.Mock
}

func (_m *MockQueries) EXPECT() *MockQueries_Expecter {
	return &MockQueries_Expecter{mock: &_m.Mock}
}

// ArrayTypes provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockQueries) ArrayTypes(_a0 context.Context, _a1 queries.Executor, _a2 []string) ([]queries.ArrayType, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []queries.ArrayType
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) ([]queries.ArrayType, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) []queries.ArrayType); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]queries.ArrayType)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, queries.Executor, []string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQueries_ArrayTypes_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ArrayTypes'
type MockQueries_ArrayTypes_Call struct {
	*mock.Call
}

// ArrayTypes is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 queries.Executor
//   - _a2 []string
func (_e *MockQueries_Expecter) ArrayTypes(_a0 interface{}, _a1 interface{}, _a2 interface{}) *MockQueries_ArrayTypes_Call {
	return &MockQueries_ArrayTypes_Call{Call: _e.mock.On("ArrayTypes", _a0, _a1, _a2)}
}

func (_c *MockQueries_ArrayTypes_Call) Run(run func(_a0 context.Context, _a1 queries.Executor, _a2 []string)) *MockQueries_ArrayTypes_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(queries.Executor), args[2].([]string))
	})
	return _c
}

func (_c *MockQueries_ArrayTypes_Call) Return(_a0 []queries.ArrayType, _a1 error) *MockQueries_ArrayTypes_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueries_ArrayTypes_Call) RunAndReturn(run func(context.Context, queries.Executor, []string) ([]queries.ArrayType, error)) *MockQueries_ArrayTypes_Call {
	_c.Call.Return(run)
	return _c
}

// Columns provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockQueries) Columns(_a0 context.Context, _a1 queries.Executor, _a2 []string) ([]queries.Column, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []queries.Column
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) ([]queries.Column, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) []queries.Column); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]queries.Column)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, queries.Executor, []string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQueries_Columns_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Columns'
type MockQueries_Columns_Call struct {
	*mock.Call
}

// Columns is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 queries.Executor
//   - _a2 []string
func (_e *MockQueries_Expecter) Columns(_a0 interface{}, _a1 interface{}, _a2 interface{}) *MockQueries_Columns_Call {
	return &MockQueries_Columns_Call{Call: _e.mock.On("Columns", _a0, _a1, _a2)}
}

func (_c *MockQueries_Columns_Call) Run(run func(_a0 context.Context, _a1 queries.Executor, _a2 []string)) *MockQueries_Columns_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(queries.Executor), args[2].([]string))
	})
	return _c
}

func (_c *MockQueries_Columns_Call) Return(_a0 []queries.Column, _a1 error) *MockQueries_Columns_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueries_Columns_Call) RunAndReturn(run func(context.Context, queries.Executor, []string) ([]queries.Column, error)) *MockQueries_Columns_Call {
	_c.Call.Return(run)
	return _c
}

// ConstraintColumns provides a mock function with given fields: _a0, _a1, _a2, _a3
func (_m *MockQueries) ConstraintColumns(_a0 context.Context, _a1 queries.Executor, _a2 []string, _a3 []string) ([]queries.ConstraintColumn, error) {
	ret := _m.Called(_a0, _a1, _a2, _a3)

	var r0 []queries.ConstraintColumn
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string, []string) ([]queries.ConstraintColumn, error)); ok {
		return rf(_a0, _a1, _a2, _a3)
	}
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string, []string) []queries.ConstraintColumn); ok {
		r0 = rf(_a0, _a1, _a2, _a3)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]queries.ConstraintColumn)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, queries.Executor, []string, []string) error); ok {
		r1 = rf(_a0, _a1, _a2, _a3)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQueries_ConstraintColumns_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ConstraintColumns'
type MockQueries_ConstraintColumns_Call struct {
	*mock.Call
}

// ConstraintColumns is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 queries.Executor
//   - _a2 []string
//   - _a3 []string
func (_e *MockQueries_Expecter) ConstraintColumns(_a0 interface{}, _a1 interface{}, _a2 interface{}, _a3 interface{}) *MockQueries_ConstraintColumns_Call {
	return &MockQueries_ConstraintColumns_Call{Call: _e.mock.On("ConstraintColumns", _a0, _a1, _a2, _a3)}
}

func (_c *MockQueries_ConstraintColumns_Call) Run(run func(_a0 context.Context, _a1 queries.Executor, _a2 []string, _a3 []string)) *MockQueries_ConstraintColumns_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(queries.Executor), args[2].([]string), args[3].([]string))
	})
	return _c
}

func (_c *MockQueries_ConstraintColumns_Call) Return(_a0 []queries.ConstraintColumn, _a1 error) *MockQueries_ConstraintColumns_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueries_ConstraintColumns_Call) RunAndReturn(run func(context.Context, queries.Executor, []string, []string) ([]queries.ConstraintColumn, error)) *MockQueries_ConstraintColumns_Call {
	_c.Call.Return(run)
	return _c
}

// Enums provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockQueries) Enums(_a0 context.Context, _a1 queries.Executor, _a2 []string) ([]queries.Enum, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []queries.Enum
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) ([]queries.Enum, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) []queries.Enum); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]queries.Enum)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, queries.Executor, []string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQueries_Enums_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Enums'
type MockQueries_Enums_Call struct {
	*mock.Call
}

// Enums is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 queries.Executor
//   - _a2 []string
func (_e *MockQueries_Expecter) Enums(_a0 interface{}, _a1 interface{}, _a2 interface{}) *MockQueries_Enums_Call {
	return &MockQueries_Enums_Call{Call: _e.mock.On("Enums", _a0, _a1, _a2)}
}

func (_c *MockQueries_Enums_Call) Run(run func(_a0 context.Context, _a1 queries.Executor, _a2 []string)) *MockQueries_Enums_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(queries.Executor), args[2].([]string))
	})
	return _c
}

func (_c *MockQueries_Enums_Call) Return(_a0 []queries.Enum, _a1 error) *MockQueries_Enums_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueries_Enums_Call) RunAndReturn(run func(context.Context, queries.Executor, []string) ([]queries.Enum, error)) *MockQueries_Enums_Call {
	_c.Call.Return(run)
	return _c
}

// ForeignKeys provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockQueries) ForeignKeys(_a0 context.Context, _a1 queries.Executor, _a2 []string) ([]queries.ForeignKey, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []queries.ForeignKey
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) ([]queries.ForeignKey, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) []queries.ForeignKey); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]queries.ForeignKey)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, queries.Executor, []string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQueries_ForeignKeys_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ForeignKeys'
type MockQueries_ForeignKeys_Call struct {
	*mock.Call
}

// ForeignKeys is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 queries.Executor
//   - _a2 []string
func (_e *MockQueries_Expecter) ForeignKeys(_a0 interface{}, _a1 interface{}, _a2 interface{}) *MockQueries_ForeignKeys_Call {
	return &MockQueries_ForeignKeys_Call{Call: _e.mock.On("ForeignKeys", _a0, _a1, _a2)}
}

func (_c *MockQueries_ForeignKeys_Call) Run(run func(_a0 context.Context, _a1 queries.Executor, _a2 []string)) *MockQueries_ForeignKeys_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(queries.Executor), args[2].([]string))
	})
	return _c
}

func (_c *MockQueries_ForeignKeys_Call) Return(_a0 []queries.ForeignKey, _a1 error) *MockQueries_ForeignKeys_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueries_ForeignKeys_Call) RunAndReturn(run func(context.Context, queries.Executor, []string) ([]queries.ForeignKey, error)) *MockQueries_ForeignKeys_Call {
	_c.Call.Return(run)
	return _c
}

// TableConstraints provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockQueries) TableConstraints(_a0 context.Context, _a1 queries.Executor, _a2 []string) ([]queries.TableConstraint, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []queries.TableConstraint
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) ([]queries.TableConstraint, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) []queries.TableConstraint); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]queries.TableConstraint)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, queries.Executor, []string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQueries_TableConstraints_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'TableConstraints'
type MockQueries_TableConstraints_Call struct {
	*mock.Call
}

// TableConstraints is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 queries.Executor
//   - _a2 []string
func (_e *MockQueries_Expecter) TableConstraints(_a0 interface{}, _a1 interface{}, _a2 interface{}) *MockQueries_TableConstraints_Call {
	return &MockQueries_TableConstraints_Call{Call: _e.mock.On("TableConstraints", _a0, _a1, _a2)}
}

func (_c *MockQueries_TableConstraints_Call) Run(run func(_a0 context.Context, _a1 queries.Executor, _a2 []string)) *MockQueries_TableConstraints_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(queries.Executor), args[2].([]string))
	})
	return _c
}

func (_c *MockQueries_TableConstraints_Call) Return(_a0 []queries.TableConstraint, _a1 error) *MockQueries_TableConstraints_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueries_TableConstraints_Call) RunAndReturn(run func(context.Context, queries.Executor, []string) ([]queries.TableConstraint, error)) *MockQueries_TableConstraints_Call {
	_c.Call.Return(run)
	return _c
}

// Tables provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockQueries) Tables(_a0 context.Context, _a1 queries.Executor, _a2 []queries.TablesPattern) ([]queries.Tables, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []queries.Tables
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []queries.TablesPattern) ([]queries.Tables, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []queries.TablesPattern) []queries.Tables); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]queries.Tables)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, queries.Executor, []queries.TablesPattern) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQueries_Tables_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Tables'
type MockQueries_Tables_Call struct {
	*mock.Call
}

// Tables is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 queries.Executor
//   - _a2 []queries.TablesPattern
func (_e *MockQueries_Expecter) Tables(_a0 interface{}, _a1 interface{}, _a2 interface{}) *MockQueries_Tables_Call {
	return &MockQueries_Tables_Call{Call: _e.mock.On("Tables", _a0, _a1, _a2)}
}

func (_c *MockQueries_Tables_Call) Run(run func(_a0 context.Context, _a1 queries.Executor, _a2 []queries.TablesPattern)) *MockQueries_Tables_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(queries.Executor), args[2].([]queries.TablesPattern))
	})
	return _c
}

func (_c *MockQueries_Tables_Call) Return(_a0 []queries.Tables, _a1 error) *MockQueries_Tables_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueries_Tables_Call) RunAndReturn(run func(context.Context, queries.Executor, []queries.TablesPattern) ([]queries.Tables, error)) *MockQueries_Tables_Call {
	_c.Call.Return(run)
	return _c
}

// Types provides a mock function with given fields: _a0, _a1, _a2
func (_m *MockQueries) Types(_a0 context.Context, _a1 queries.Executor, _a2 []string) ([]queries.Type, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 []queries.Type
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) ([]queries.Type, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, queries.Executor, []string) []queries.Type); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]queries.Type)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, queries.Executor, []string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQueries_Types_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Types'
type MockQueries_Types_Call struct {
	*mock.Call
}

// Types is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 queries.Executor
//   - _a2 []string
func (_e *MockQueries_Expecter) Types(_a0 interface{}, _a1 interface{}, _a2 interface{}) *MockQueries_Types_Call {
	return &MockQueries_Types_Call{Call: _e.mock.On("Types", _a0, _a1, _a2)}
}

func (_c *MockQueries_Types_Call) Run(run func(_a0 context.Context, _a1 queries.Executor, _a2 []string)) *MockQueries_Types_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(queries.Executor), args[2].([]string))
	})
	return _c
}

func (_c *MockQueries_Types_Call) Return(_a0 []queries.Type, _a1 error) *MockQueries_Types_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQueries_Types_Call) RunAndReturn(run func(context.Context, queries.Executor, []string) ([]queries.Type, error)) *MockQueries_Types_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewMockQueries interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockQueries creates a new instance of MockQueries. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockQueries(t mockConstructorTestingTNewMockQueries) *MockQueries {
	mock := &MockQueries{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
