// Code generated by mockery v2.30.16. DO NOT EDIT.

package mocks

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	context "context"

	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/api/apps/v1"

	v1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

type Client_Expecter struct {
	mock *mock.Mock
}

func (_m *Client) EXPECT() *Client_Expecter {
	return &Client_Expecter{mock: &_m.Mock}
}

// DeleteDeployment provides a mock function with given fields: _a0, _a1, _a2
func (_m *Client) DeleteDeployment(_a0 context.Context, _a1 string, _a2 string) error {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_DeleteDeployment_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteDeployment'
type Client_DeleteDeployment_Call struct {
	*mock.Call
}

// DeleteDeployment is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
//   - _a2 string
func (_e *Client_Expecter) DeleteDeployment(_a0 interface{}, _a1 interface{}, _a2 interface{}) *Client_DeleteDeployment_Call {
	return &Client_DeleteDeployment_Call{Call: _e.mock.On("DeleteDeployment", _a0, _a1, _a2)}
}

func (_c *Client_DeleteDeployment_Call) Run(run func(_a0 context.Context, _a1 string, _a2 string)) *Client_DeleteDeployment_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *Client_DeleteDeployment_Call) Return(_a0 error) *Client_DeleteDeployment_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_DeleteDeployment_Call) RunAndReturn(run func(context.Context, string, string) error) *Client_DeleteDeployment_Call {
	_c.Call.Return(run)
	return _c
}

// GetDeployment provides a mock function with given fields: _a0, _a1, _a2
func (_m *Client) GetDeployment(_a0 context.Context, _a1 string, _a2 string) (*v1.Deployment, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *v1.Deployment
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*v1.Deployment, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *v1.Deployment); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Deployment)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(_a0, _a1, _a2)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetDeployment_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetDeployment'
type Client_GetDeployment_Call struct {
	*mock.Call
}

// GetDeployment is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
//   - _a2 string
func (_e *Client_Expecter) GetDeployment(_a0 interface{}, _a1 interface{}, _a2 interface{}) *Client_GetDeployment_Call {
	return &Client_GetDeployment_Call{Call: _e.mock.On("GetDeployment", _a0, _a1, _a2)}
}

func (_c *Client_GetDeployment_Call) Run(run func(_a0 context.Context, _a1 string, _a2 string)) *Client_GetDeployment_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *Client_GetDeployment_Call) Return(_a0 *v1.Deployment, _a1 error) *Client_GetDeployment_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetDeployment_Call) RunAndReturn(run func(context.Context, string, string) (*v1.Deployment, error)) *Client_GetDeployment_Call {
	_c.Call.Return(run)
	return _c
}

// GetMutatingWebHookConfiguration provides a mock function with given fields: ctx, name
func (_m *Client) GetMutatingWebHookConfiguration(ctx context.Context, name string) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	ret := _m.Called(ctx, name)

	var r0 *admissionregistrationv1.MutatingWebhookConfiguration
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*admissionregistrationv1.MutatingWebhookConfiguration, error)); ok {
		return rf(ctx, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *admissionregistrationv1.MutatingWebhookConfiguration); ok {
		r0 = rf(ctx, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*admissionregistrationv1.MutatingWebhookConfiguration)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetMutatingWebHookConfiguration_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetMutatingWebHookConfiguration'
type Client_GetMutatingWebHookConfiguration_Call struct {
	*mock.Call
}

// GetMutatingWebHookConfiguration is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
func (_e *Client_Expecter) GetMutatingWebHookConfiguration(ctx interface{}, name interface{}) *Client_GetMutatingWebHookConfiguration_Call {
	return &Client_GetMutatingWebHookConfiguration_Call{Call: _e.mock.On("GetMutatingWebHookConfiguration", ctx, name)}
}

func (_c *Client_GetMutatingWebHookConfiguration_Call) Run(run func(ctx context.Context, name string)) *Client_GetMutatingWebHookConfiguration_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Client_GetMutatingWebHookConfiguration_Call) Return(_a0 *admissionregistrationv1.MutatingWebhookConfiguration, _a1 error) *Client_GetMutatingWebHookConfiguration_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetMutatingWebHookConfiguration_Call) RunAndReturn(run func(context.Context, string) (*admissionregistrationv1.MutatingWebhookConfiguration, error)) *Client_GetMutatingWebHookConfiguration_Call {
	_c.Call.Return(run)
	return _c
}

// GetNATSResources provides a mock function with given fields: _a0, _a1
func (_m *Client) GetNATSResources(_a0 context.Context, _a1 string) (*v1alpha1.NATSList, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *v1alpha1.NATSList
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*v1alpha1.NATSList, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1alpha1.NATSList); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.NATSList)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetNATSResources_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetNATSResources'
type Client_GetNATSResources_Call struct {
	*mock.Call
}

// GetNATSResources is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
func (_e *Client_Expecter) GetNATSResources(_a0 interface{}, _a1 interface{}) *Client_GetNATSResources_Call {
	return &Client_GetNATSResources_Call{Call: _e.mock.On("GetNATSResources", _a0, _a1)}
}

func (_c *Client_GetNATSResources_Call) Run(run func(_a0 context.Context, _a1 string)) *Client_GetNATSResources_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Client_GetNATSResources_Call) Return(_a0 *v1alpha1.NATSList, _a1 error) *Client_GetNATSResources_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetNATSResources_Call) RunAndReturn(run func(context.Context, string) (*v1alpha1.NATSList, error)) *Client_GetNATSResources_Call {
	_c.Call.Return(run)
	return _c
}

// GetSecret provides a mock function with given fields: _a0, _a1
func (_m *Client) GetSecret(_a0 context.Context, _a1 string) (*corev1.Secret, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *corev1.Secret
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*corev1.Secret, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *corev1.Secret); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.Secret)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetSecret_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetSecret'
type Client_GetSecret_Call struct {
	*mock.Call
}

// GetSecret is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
func (_e *Client_Expecter) GetSecret(_a0 interface{}, _a1 interface{}) *Client_GetSecret_Call {
	return &Client_GetSecret_Call{Call: _e.mock.On("GetSecret", _a0, _a1)}
}

func (_c *Client_GetSecret_Call) Run(run func(_a0 context.Context, _a1 string)) *Client_GetSecret_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Client_GetSecret_Call) Return(_a0 *corev1.Secret, _a1 error) *Client_GetSecret_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetSecret_Call) RunAndReturn(run func(context.Context, string) (*corev1.Secret, error)) *Client_GetSecret_Call {
	_c.Call.Return(run)
	return _c
}

// GetValidatingWebHookConfiguration provides a mock function with given fields: ctx, name
func (_m *Client) GetValidatingWebHookConfiguration(ctx context.Context, name string) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	ret := _m.Called(ctx, name)

	var r0 *admissionregistrationv1.ValidatingWebhookConfiguration
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*admissionregistrationv1.ValidatingWebhookConfiguration, error)); ok {
		return rf(ctx, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *admissionregistrationv1.ValidatingWebhookConfiguration); ok {
		r0 = rf(ctx, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*admissionregistrationv1.ValidatingWebhookConfiguration)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetValidatingWebHookConfiguration_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetValidatingWebHookConfiguration'
type Client_GetValidatingWebHookConfiguration_Call struct {
	*mock.Call
}

// GetValidatingWebHookConfiguration is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
func (_e *Client_Expecter) GetValidatingWebHookConfiguration(ctx interface{}, name interface{}) *Client_GetValidatingWebHookConfiguration_Call {
	return &Client_GetValidatingWebHookConfiguration_Call{Call: _e.mock.On("GetValidatingWebHookConfiguration", ctx, name)}
}

func (_c *Client_GetValidatingWebHookConfiguration_Call) Run(run func(ctx context.Context, name string)) *Client_GetValidatingWebHookConfiguration_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Client_GetValidatingWebHookConfiguration_Call) Return(_a0 *admissionregistrationv1.ValidatingWebhookConfiguration, _a1 error) *Client_GetValidatingWebHookConfiguration_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetValidatingWebHookConfiguration_Call) RunAndReturn(run func(context.Context, string) (*admissionregistrationv1.ValidatingWebhookConfiguration, error)) *Client_GetValidatingWebHookConfiguration_Call {
	_c.Call.Return(run)
	return _c
}

// PatchApply provides a mock function with given fields: _a0, _a1
func (_m *Client) PatchApply(_a0 context.Context, _a1 client.Object) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, client.Object) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_PatchApply_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PatchApply'
type Client_PatchApply_Call struct {
	*mock.Call
}

// PatchApply is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 client.Object
func (_e *Client_Expecter) PatchApply(_a0 interface{}, _a1 interface{}) *Client_PatchApply_Call {
	return &Client_PatchApply_Call{Call: _e.mock.On("PatchApply", _a0, _a1)}
}

func (_c *Client_PatchApply_Call) Run(run func(_a0 context.Context, _a1 client.Object)) *Client_PatchApply_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(client.Object))
	})
	return _c
}

func (_c *Client_PatchApply_Call) Return(_a0 error) *Client_PatchApply_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_PatchApply_Call) RunAndReturn(run func(context.Context, client.Object) error) *Client_PatchApply_Call {
	_c.Call.Return(run)
	return _c
}

// NewClient creates a new instance of Client. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *Client {
	mock := &Client{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
