// Code generated by mockery v2.36.0. DO NOT EDIT.

package mocks

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"

	client "sigs.k8s.io/controller-runtime/pkg/client"

	context "context"

	corev1 "k8s.io/api/core/v1"

	mock "github.com/stretchr/testify/mock"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	v1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"

	v1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"

	v1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
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

// APIRuleCRDExists provides a mock function with given fields: _a0
func (_m *Client) APIRuleCRDExists(_a0 context.Context) (bool, error) {
	ret := _m.Called(_a0)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (bool, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_APIRuleCRDExists_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'APIRuleCRDExists'
type Client_APIRuleCRDExists_Call struct {
	*mock.Call
}

// APIRuleCRDExists is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *Client_Expecter) APIRuleCRDExists(_a0 interface{}) *Client_APIRuleCRDExists_Call {
	return &Client_APIRuleCRDExists_Call{Call: _e.mock.On("APIRuleCRDExists", _a0)}
}

func (_c *Client_APIRuleCRDExists_Call) Run(run func(_a0 context.Context)) *Client_APIRuleCRDExists_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Client_APIRuleCRDExists_Call) Return(_a0 bool, _a1 error) *Client_APIRuleCRDExists_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_APIRuleCRDExists_Call) RunAndReturn(run func(context.Context) (bool, error)) *Client_APIRuleCRDExists_Call {
	_c.Call.Return(run)
	return _c
}

// ApplicationCRDExists provides a mock function with given fields: _a0
func (_m *Client) ApplicationCRDExists(_a0 context.Context) (bool, error) {
	ret := _m.Called(_a0)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (bool, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) bool); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_ApplicationCRDExists_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ApplicationCRDExists'
type Client_ApplicationCRDExists_Call struct {
	*mock.Call
}

// ApplicationCRDExists is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *Client_Expecter) ApplicationCRDExists(_a0 interface{}) *Client_ApplicationCRDExists_Call {
	return &Client_ApplicationCRDExists_Call{Call: _e.mock.On("ApplicationCRDExists", _a0)}
}

func (_c *Client_ApplicationCRDExists_Call) Run(run func(_a0 context.Context)) *Client_ApplicationCRDExists_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Client_ApplicationCRDExists_Call) Return(_a0 bool, _a1 error) *Client_ApplicationCRDExists_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_ApplicationCRDExists_Call) RunAndReturn(run func(context.Context) (bool, error)) *Client_ApplicationCRDExists_Call {
	_c.Call.Return(run)
	return _c
}

// CreatePeerAuthentication provides a mock function with given fields: ctx, authentication
func (_m *Client) CreatePeerAuthentication(ctx context.Context, authentication *v1beta1.PeerAuthentication) error {
	ret := _m.Called(ctx, authentication)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1beta1.PeerAuthentication) error); ok {
		r0 = rf(ctx, authentication)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_CreatePeerAuthentication_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreatePeerAuthentication'
type Client_CreatePeerAuthentication_Call struct {
	*mock.Call
}

// CreatePeerAuthentication is a helper method to define mock.On call
//   - ctx context.Context
//   - authentication *v1beta1.PeerAuthentication
func (_e *Client_Expecter) CreatePeerAuthentication(ctx interface{}, authentication interface{}) *Client_CreatePeerAuthentication_Call {
	return &Client_CreatePeerAuthentication_Call{Call: _e.mock.On("CreatePeerAuthentication", ctx, authentication)}
}

func (_c *Client_CreatePeerAuthentication_Call) Run(run func(ctx context.Context, authentication *v1beta1.PeerAuthentication)) *Client_CreatePeerAuthentication_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1beta1.PeerAuthentication))
	})
	return _c
}

func (_c *Client_CreatePeerAuthentication_Call) Return(_a0 error) *Client_CreatePeerAuthentication_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_CreatePeerAuthentication_Call) RunAndReturn(run func(context.Context, *v1beta1.PeerAuthentication) error) *Client_CreatePeerAuthentication_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteClusterRole provides a mock function with given fields: _a0, _a1, _a2
func (_m *Client) DeleteClusterRole(_a0 context.Context, _a1 string, _a2 string) error {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_DeleteClusterRole_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteClusterRole'
type Client_DeleteClusterRole_Call struct {
	*mock.Call
}

// DeleteClusterRole is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
//   - _a2 string
func (_e *Client_Expecter) DeleteClusterRole(_a0 interface{}, _a1 interface{}, _a2 interface{}) *Client_DeleteClusterRole_Call {
	return &Client_DeleteClusterRole_Call{Call: _e.mock.On("DeleteClusterRole", _a0, _a1, _a2)}
}

func (_c *Client_DeleteClusterRole_Call) Run(run func(_a0 context.Context, _a1 string, _a2 string)) *Client_DeleteClusterRole_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *Client_DeleteClusterRole_Call) Return(_a0 error) *Client_DeleteClusterRole_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_DeleteClusterRole_Call) RunAndReturn(run func(context.Context, string, string) error) *Client_DeleteClusterRole_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteClusterRoleBinding provides a mock function with given fields: _a0, _a1, _a2
func (_m *Client) DeleteClusterRoleBinding(_a0 context.Context, _a1 string, _a2 string) error {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_DeleteClusterRoleBinding_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteClusterRoleBinding'
type Client_DeleteClusterRoleBinding_Call struct {
	*mock.Call
}

// DeleteClusterRoleBinding is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
//   - _a2 string
func (_e *Client_Expecter) DeleteClusterRoleBinding(_a0 interface{}, _a1 interface{}, _a2 interface{}) *Client_DeleteClusterRoleBinding_Call {
	return &Client_DeleteClusterRoleBinding_Call{Call: _e.mock.On("DeleteClusterRoleBinding", _a0, _a1, _a2)}
}

func (_c *Client_DeleteClusterRoleBinding_Call) Run(run func(_a0 context.Context, _a1 string, _a2 string)) *Client_DeleteClusterRoleBinding_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *Client_DeleteClusterRoleBinding_Call) Return(_a0 error) *Client_DeleteClusterRoleBinding_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_DeleteClusterRoleBinding_Call) RunAndReturn(run func(context.Context, string, string) error) *Client_DeleteClusterRoleBinding_Call {
	_c.Call.Return(run)
	return _c
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

// GetCRD provides a mock function with given fields: _a0, _a1
func (_m *Client) GetCRD(_a0 context.Context, _a1 string) (*v1.CustomResourceDefinition, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *v1.CustomResourceDefinition
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*v1.CustomResourceDefinition, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *v1.CustomResourceDefinition); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.CustomResourceDefinition)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetCRD_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetCRD'
type Client_GetCRD_Call struct {
	*mock.Call
}

// GetCRD is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
func (_e *Client_Expecter) GetCRD(_a0 interface{}, _a1 interface{}) *Client_GetCRD_Call {
	return &Client_GetCRD_Call{Call: _e.mock.On("GetCRD", _a0, _a1)}
}

func (_c *Client_GetCRD_Call) Run(run func(_a0 context.Context, _a1 string)) *Client_GetCRD_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Client_GetCRD_Call) Return(_a0 *v1.CustomResourceDefinition, _a1 error) *Client_GetCRD_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetCRD_Call) RunAndReturn(run func(context.Context, string) (*v1.CustomResourceDefinition, error)) *Client_GetCRD_Call {
	_c.Call.Return(run)
	return _c
}

// GetConfigMap provides a mock function with given fields: ctx, name, namespace
func (_m *Client) GetConfigMap(ctx context.Context, name string, namespace string) (*corev1.ConfigMap, error) {
	ret := _m.Called(ctx, name, namespace)

	var r0 *corev1.ConfigMap
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*corev1.ConfigMap, error)); ok {
		return rf(ctx, name, namespace)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *corev1.ConfigMap); ok {
		r0 = rf(ctx, name, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.ConfigMap)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, name, namespace)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetConfigMap_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetConfigMap'
type Client_GetConfigMap_Call struct {
	*mock.Call
}

// GetConfigMap is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - namespace string
func (_e *Client_Expecter) GetConfigMap(ctx interface{}, name interface{}, namespace interface{}) *Client_GetConfigMap_Call {
	return &Client_GetConfigMap_Call{Call: _e.mock.On("GetConfigMap", ctx, name, namespace)}
}

func (_c *Client_GetConfigMap_Call) Run(run func(ctx context.Context, name string, namespace string)) *Client_GetConfigMap_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *Client_GetConfigMap_Call) Return(_a0 *corev1.ConfigMap, _a1 error) *Client_GetConfigMap_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetConfigMap_Call) RunAndReturn(run func(context.Context, string, string) (*corev1.ConfigMap, error)) *Client_GetConfigMap_Call {
	_c.Call.Return(run)
	return _c
}

// GetDeployment provides a mock function with given fields: _a0, _a1, _a2
func (_m *Client) GetDeployment(_a0 context.Context, _a1 string, _a2 string) (*appsv1.Deployment, error) {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 *appsv1.Deployment
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*appsv1.Deployment, error)); ok {
		return rf(_a0, _a1, _a2)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *appsv1.Deployment); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.Deployment)
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

func (_c *Client_GetDeployment_Call) Return(_a0 *appsv1.Deployment, _a1 error) *Client_GetDeployment_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetDeployment_Call) RunAndReturn(run func(context.Context, string, string) (*appsv1.Deployment, error)) *Client_GetDeployment_Call {
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

// GetSubscriptions provides a mock function with given fields: ctx
func (_m *Client) GetSubscriptions(ctx context.Context) (*v1alpha2.SubscriptionList, error) {
	ret := _m.Called(ctx)

	var r0 *v1alpha2.SubscriptionList
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*v1alpha2.SubscriptionList, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *v1alpha2.SubscriptionList); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha2.SubscriptionList)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetSubscriptions_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetSubscriptions'
type Client_GetSubscriptions_Call struct {
	*mock.Call
}

// GetSubscriptions is a helper method to define mock.On call
//   - ctx context.Context
func (_e *Client_Expecter) GetSubscriptions(ctx interface{}) *Client_GetSubscriptions_Call {
	return &Client_GetSubscriptions_Call{Call: _e.mock.On("GetSubscriptions", ctx)}
}

func (_c *Client_GetSubscriptions_Call) Run(run func(ctx context.Context)) *Client_GetSubscriptions_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Client_GetSubscriptions_Call) Return(_a0 *v1alpha2.SubscriptionList, _a1 error) *Client_GetSubscriptions_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetSubscriptions_Call) RunAndReturn(run func(context.Context) (*v1alpha2.SubscriptionList, error)) *Client_GetSubscriptions_Call {
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

// UpdateDeployment provides a mock function with given fields: _a0, _a1
func (_m *Client) UpdateDeployment(_a0 context.Context, _a1 *appsv1.Deployment) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *appsv1.Deployment) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Client_UpdateDeployment_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateDeployment'
type Client_UpdateDeployment_Call struct {
	*mock.Call
}

// UpdateDeployment is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *appsv1.Deployment
func (_e *Client_Expecter) UpdateDeployment(_a0 interface{}, _a1 interface{}) *Client_UpdateDeployment_Call {
	return &Client_UpdateDeployment_Call{Call: _e.mock.On("UpdateDeployment", _a0, _a1)}
}

func (_c *Client_UpdateDeployment_Call) Run(run func(_a0 context.Context, _a1 *appsv1.Deployment)) *Client_UpdateDeployment_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*appsv1.Deployment))
	})
	return _c
}

func (_c *Client_UpdateDeployment_Call) Return(_a0 error) *Client_UpdateDeployment_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Client_UpdateDeployment_Call) RunAndReturn(run func(context.Context, *appsv1.Deployment) error) *Client_UpdateDeployment_Call {
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
