// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	env "github.com/kyma-project/eventing-manager/pkg/env"
	manager "sigs.k8s.io/controller-runtime/pkg/manager"

	mock "github.com/stretchr/testify/mock"

	subscriptionmanagermanager "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager"
)

// Manager is an autogenerated mock type for the Manager type
type Manager struct {
	mock.Mock
}

type Manager_Expecter struct {
	mock *mock.Mock
}

func (_m *Manager) EXPECT() *Manager_Expecter {
	return &Manager_Expecter{mock: &_m.Mock}
}

// Init provides a mock function with given fields: mgr
func (_m *Manager) Init(mgr manager.Manager) error {
	ret := _m.Called(mgr)

	if len(ret) == 0 {
		panic("no return value specified for Init")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(manager.Manager) error); ok {
		r0 = rf(mgr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Manager_Init_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Init'
type Manager_Init_Call struct {
	*mock.Call
}

// Init is a helper method to define mock.On call
//   - mgr manager.Manager
func (_e *Manager_Expecter) Init(mgr interface{}) *Manager_Init_Call {
	return &Manager_Init_Call{Call: _e.mock.On("Init", mgr)}
}

func (_c *Manager_Init_Call) Run(run func(mgr manager.Manager)) *Manager_Init_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(manager.Manager))
	})
	return _c
}

func (_c *Manager_Init_Call) Return(_a0 error) *Manager_Init_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Manager_Init_Call) RunAndReturn(run func(manager.Manager) error) *Manager_Init_Call {
	_c.Call.Return(run)
	return _c
}

// Start provides a mock function with given fields: defaultSubsConfig, params
func (_m *Manager) Start(defaultSubsConfig env.DefaultSubscriptionConfig, params subscriptionmanagermanager.Params) error {
	ret := _m.Called(defaultSubsConfig, params)

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(env.DefaultSubscriptionConfig, subscriptionmanagermanager.Params) error); ok {
		r0 = rf(defaultSubsConfig, params)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Manager_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type Manager_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
//   - defaultSubsConfig env.DefaultSubscriptionConfig
//   - params subscriptionmanagermanager.Params
func (_e *Manager_Expecter) Start(defaultSubsConfig interface{}, params interface{}) *Manager_Start_Call {
	return &Manager_Start_Call{Call: _e.mock.On("Start", defaultSubsConfig, params)}
}

func (_c *Manager_Start_Call) Run(run func(defaultSubsConfig env.DefaultSubscriptionConfig, params subscriptionmanagermanager.Params)) *Manager_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(env.DefaultSubscriptionConfig), args[1].(subscriptionmanagermanager.Params))
	})
	return _c
}

func (_c *Manager_Start_Call) Return(_a0 error) *Manager_Start_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Manager_Start_Call) RunAndReturn(run func(env.DefaultSubscriptionConfig, subscriptionmanagermanager.Params) error) *Manager_Start_Call {
	_c.Call.Return(run)
	return _c
}

// Stop provides a mock function with given fields: runCleanup
func (_m *Manager) Stop(runCleanup bool) error {
	ret := _m.Called(runCleanup)

	if len(ret) == 0 {
		panic("no return value specified for Stop")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(bool) error); ok {
		r0 = rf(runCleanup)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Manager_Stop_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Stop'
type Manager_Stop_Call struct {
	*mock.Call
}

// Stop is a helper method to define mock.On call
//   - runCleanup bool
func (_e *Manager_Expecter) Stop(runCleanup interface{}) *Manager_Stop_Call {
	return &Manager_Stop_Call{Call: _e.mock.On("Stop", runCleanup)}
}

func (_c *Manager_Stop_Call) Run(run func(runCleanup bool)) *Manager_Stop_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(bool))
	})
	return _c
}

func (_c *Manager_Stop_Call) Return(_a0 error) *Manager_Stop_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Manager_Stop_Call) RunAndReturn(run func(bool) error) *Manager_Stop_Call {
	_c.Call.Return(run)
	return _c
}

// NewManager creates a new instance of Manager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *Manager {
	mock := &Manager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
