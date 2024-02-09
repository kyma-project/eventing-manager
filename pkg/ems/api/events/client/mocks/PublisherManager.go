// Code generated by mockery v2.40.2. DO NOT EDIT.

package mocks

import (
	event "github.com/cloudevents/sdk-go/v2/event"
	mock "github.com/stretchr/testify/mock"

	types "github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
)

// PublisherManager is an autogenerated mock type for the PublisherManager type
type PublisherManager struct {
	mock.Mock
}

type PublisherManager_Expecter struct {
	mock *mock.Mock
}

func (_m *PublisherManager) EXPECT() *PublisherManager_Expecter {
	return &PublisherManager_Expecter{mock: &_m.Mock}
}

// Create provides a mock function with given fields: subscription
func (_m *PublisherManager) Create(subscription *types.Subscription) (*types.CreateResponse, error) {
	ret := _m.Called(subscription)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *types.CreateResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(*types.Subscription) (*types.CreateResponse, error)); ok {
		return rf(subscription)
	}
	if rf, ok := ret.Get(0).(func(*types.Subscription) *types.CreateResponse); ok {
		r0 = rf(subscription)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.CreateResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(*types.Subscription) error); ok {
		r1 = rf(subscription)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PublisherManager_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type PublisherManager_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - subscription *types.Subscription
func (_e *PublisherManager_Expecter) Create(subscription interface{}) *PublisherManager_Create_Call {
	return &PublisherManager_Create_Call{Call: _e.mock.On("Create", subscription)}
}

func (_c *PublisherManager_Create_Call) Run(run func(subscription *types.Subscription)) *PublisherManager_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*types.Subscription))
	})
	return _c
}

func (_c *PublisherManager_Create_Call) Return(_a0 *types.CreateResponse, _a1 error) *PublisherManager_Create_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *PublisherManager_Create_Call) RunAndReturn(run func(*types.Subscription) (*types.CreateResponse, error)) *PublisherManager_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: name
func (_m *PublisherManager) Delete(name string) (*types.DeleteResponse, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 *types.DeleteResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*types.DeleteResponse, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) *types.DeleteResponse); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.DeleteResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PublisherManager_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type PublisherManager_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - name string
func (_e *PublisherManager_Expecter) Delete(name interface{}) *PublisherManager_Delete_Call {
	return &PublisherManager_Delete_Call{Call: _e.mock.On("Delete", name)}
}

func (_c *PublisherManager_Delete_Call) Run(run func(name string)) *PublisherManager_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *PublisherManager_Delete_Call) Return(_a0 *types.DeleteResponse, _a1 error) *PublisherManager_Delete_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *PublisherManager_Delete_Call) RunAndReturn(run func(string) (*types.DeleteResponse, error)) *PublisherManager_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: name
func (_m *PublisherManager) Get(name string) (*types.Subscription, *types.Response, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *types.Subscription
	var r1 *types.Response
	var r2 error
	if rf, ok := ret.Get(0).(func(string) (*types.Subscription, *types.Response, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) *types.Subscription); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Subscription)
		}
	}

	if rf, ok := ret.Get(1).(func(string) *types.Response); ok {
		r1 = rf(name)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*types.Response)
		}
	}

	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(name)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// PublisherManager_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type PublisherManager_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - name string
func (_e *PublisherManager_Expecter) Get(name interface{}) *PublisherManager_Get_Call {
	return &PublisherManager_Get_Call{Call: _e.mock.On("Get", name)}
}

func (_c *PublisherManager_Get_Call) Run(run func(name string)) *PublisherManager_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *PublisherManager_Get_Call) Return(_a0 *types.Subscription, _a1 *types.Response, _a2 error) *PublisherManager_Get_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *PublisherManager_Get_Call) RunAndReturn(run func(string) (*types.Subscription, *types.Response, error)) *PublisherManager_Get_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields:
func (_m *PublisherManager) List() (*types.Subscriptions, *types.Response, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for List")
	}

	var r0 *types.Subscriptions
	var r1 *types.Response
	var r2 error
	if rf, ok := ret.Get(0).(func() (*types.Subscriptions, *types.Response, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *types.Subscriptions); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Subscriptions)
		}
	}

	if rf, ok := ret.Get(1).(func() *types.Response); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*types.Response)
		}
	}

	if rf, ok := ret.Get(2).(func() error); ok {
		r2 = rf()
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// PublisherManager_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type PublisherManager_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
func (_e *PublisherManager_Expecter) List() *PublisherManager_List_Call {
	return &PublisherManager_List_Call{Call: _e.mock.On("List")}
}

func (_c *PublisherManager_List_Call) Run(run func()) *PublisherManager_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *PublisherManager_List_Call) Return(_a0 *types.Subscriptions, _a1 *types.Response, _a2 error) *PublisherManager_List_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *PublisherManager_List_Call) RunAndReturn(run func() (*types.Subscriptions, *types.Response, error)) *PublisherManager_List_Call {
	_c.Call.Return(run)
	return _c
}

// Publish provides a mock function with given fields: cloudEvent, qos
func (_m *PublisherManager) Publish(cloudEvent event.Event, qos types.Qos) (*types.PublishResponse, error) {
	ret := _m.Called(cloudEvent, qos)

	if len(ret) == 0 {
		panic("no return value specified for Publish")
	}

	var r0 *types.PublishResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(event.Event, types.Qos) (*types.PublishResponse, error)); ok {
		return rf(cloudEvent, qos)
	}
	if rf, ok := ret.Get(0).(func(event.Event, types.Qos) *types.PublishResponse); ok {
		r0 = rf(cloudEvent, qos)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.PublishResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(event.Event, types.Qos) error); ok {
		r1 = rf(cloudEvent, qos)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PublisherManager_Publish_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Publish'
type PublisherManager_Publish_Call struct {
	*mock.Call
}

// Publish is a helper method to define mock.On call
//   - cloudEvent event.Event
//   - qos types.Qos
func (_e *PublisherManager_Expecter) Publish(cloudEvent interface{}, qos interface{}) *PublisherManager_Publish_Call {
	return &PublisherManager_Publish_Call{Call: _e.mock.On("Publish", cloudEvent, qos)}
}

func (_c *PublisherManager_Publish_Call) Run(run func(cloudEvent event.Event, qos types.Qos)) *PublisherManager_Publish_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(event.Event), args[1].(types.Qos))
	})
	return _c
}

func (_c *PublisherManager_Publish_Call) Return(_a0 *types.PublishResponse, _a1 error) *PublisherManager_Publish_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *PublisherManager_Publish_Call) RunAndReturn(run func(event.Event, types.Qos) (*types.PublishResponse, error)) *PublisherManager_Publish_Call {
	_c.Call.Return(run)
	return _c
}

// TriggerHandshake provides a mock function with given fields: name
func (_m *PublisherManager) TriggerHandshake(name string) (*types.TriggerHandshake, error) {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for TriggerHandshake")
	}

	var r0 *types.TriggerHandshake
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*types.TriggerHandshake, error)); ok {
		return rf(name)
	}
	if rf, ok := ret.Get(0).(func(string) *types.TriggerHandshake); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.TriggerHandshake)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PublisherManager_TriggerHandshake_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'TriggerHandshake'
type PublisherManager_TriggerHandshake_Call struct {
	*mock.Call
}

// TriggerHandshake is a helper method to define mock.On call
//   - name string
func (_e *PublisherManager_Expecter) TriggerHandshake(name interface{}) *PublisherManager_TriggerHandshake_Call {
	return &PublisherManager_TriggerHandshake_Call{Call: _e.mock.On("TriggerHandshake", name)}
}

func (_c *PublisherManager_TriggerHandshake_Call) Run(run func(name string)) *PublisherManager_TriggerHandshake_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *PublisherManager_TriggerHandshake_Call) Return(_a0 *types.TriggerHandshake, _a1 error) *PublisherManager_TriggerHandshake_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *PublisherManager_TriggerHandshake_Call) RunAndReturn(run func(string) (*types.TriggerHandshake, error)) *PublisherManager_TriggerHandshake_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: name, webhookAuth
func (_m *PublisherManager) Update(name string, webhookAuth *types.WebhookAuth) (*types.UpdateResponse, error) {
	ret := _m.Called(name, webhookAuth)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *types.UpdateResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(string, *types.WebhookAuth) (*types.UpdateResponse, error)); ok {
		return rf(name, webhookAuth)
	}
	if rf, ok := ret.Get(0).(func(string, *types.WebhookAuth) *types.UpdateResponse); ok {
		r0 = rf(name, webhookAuth)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.UpdateResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(string, *types.WebhookAuth) error); ok {
		r1 = rf(name, webhookAuth)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PublisherManager_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type PublisherManager_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - name string
//   - webhookAuth *types.WebhookAuth
func (_e *PublisherManager_Expecter) Update(name interface{}, webhookAuth interface{}) *PublisherManager_Update_Call {
	return &PublisherManager_Update_Call{Call: _e.mock.On("Update", name, webhookAuth)}
}

func (_c *PublisherManager_Update_Call) Run(run func(name string, webhookAuth *types.WebhookAuth)) *PublisherManager_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(*types.WebhookAuth))
	})
	return _c
}

func (_c *PublisherManager_Update_Call) Return(_a0 *types.UpdateResponse, _a1 error) *PublisherManager_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *PublisherManager_Update_Call) RunAndReturn(run func(string, *types.WebhookAuth) (*types.UpdateResponse, error)) *PublisherManager_Update_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateState provides a mock function with given fields: name, state
func (_m *PublisherManager) UpdateState(name string, state types.State) (*types.UpdateStateResponse, error) {
	ret := _m.Called(name, state)

	if len(ret) == 0 {
		panic("no return value specified for UpdateState")
	}

	var r0 *types.UpdateStateResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(string, types.State) (*types.UpdateStateResponse, error)); ok {
		return rf(name, state)
	}
	if rf, ok := ret.Get(0).(func(string, types.State) *types.UpdateStateResponse); ok {
		r0 = rf(name, state)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.UpdateStateResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(string, types.State) error); ok {
		r1 = rf(name, state)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PublisherManager_UpdateState_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateState'
type PublisherManager_UpdateState_Call struct {
	*mock.Call
}

// UpdateState is a helper method to define mock.On call
//   - name string
//   - state types.State
func (_e *PublisherManager_Expecter) UpdateState(name interface{}, state interface{}) *PublisherManager_UpdateState_Call {
	return &PublisherManager_UpdateState_Call{Call: _e.mock.On("UpdateState", name, state)}
}

func (_c *PublisherManager_UpdateState_Call) Run(run func(name string, state types.State)) *PublisherManager_UpdateState_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(types.State))
	})
	return _c
}

func (_c *PublisherManager_UpdateState_Call) Return(_a0 *types.UpdateStateResponse, _a1 error) *PublisherManager_UpdateState_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *PublisherManager_UpdateState_Call) RunAndReturn(run func(string, types.State) (*types.UpdateStateResponse, error)) *PublisherManager_UpdateState_Call {
	_c.Call.Return(run)
	return _c
}

// NewPublisherManager creates a new instance of PublisherManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPublisherManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *PublisherManager {
	mock := &PublisherManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
