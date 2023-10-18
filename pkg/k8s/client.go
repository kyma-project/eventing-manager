package k8s

import (
	"context"
	"errors"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"strings"

	clientsecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockery --name=Client --outpkg=mocks --case=underscore
type Client interface {
	GetService(context.Context, string, string) (*corev1.Service, error)
	GetDeployment(context.Context, string, string) (*v1.Deployment, error)
	UpdateDeployment(context.Context, *v1.Deployment) error
	DeleteDeployment(context.Context, string, string) error
	DeleteClusterRole(context.Context, string, string) error
	DeleteClusterRoleBinding(context.Context, string, string) error
	GetNATSResources(context.Context, string) (*natsv1alpha1.NATSList, error)
	PatchApply(context.Context, client.Object) error
	PatchApplyUnstructured(context.Context, *unstructured.Unstructured) error
	GetSecret(context.Context, string) (*corev1.Secret, error)
	GetMutatingWebHookConfiguration(ctx context.Context,
		name string) (*admissionv1.MutatingWebhookConfiguration, error)
	GetValidatingWebHookConfiguration(ctx context.Context,
		name string) (*admissionv1.ValidatingWebhookConfiguration, error)
	GetCRD(context.Context, string) (*apiextensionsv1.CustomResourceDefinition, error)
	ApplicationCRDExists(context.Context) (bool, error)
	GetVirtualService(context.Context, string, string) (*networkingv1alpha3.VirtualService, error)
	GetRequestAuthentication(context.Context, string, string) (*clientsecurityv1beta1.RequestAuthentication, error)
	GetAuthorizationPolicy(context.Context, string, string) (*clientsecurityv1beta1.AuthorizationPolicy, error)
}

type KubeClient struct {
	fieldManager  string
	client        client.Client
	clientset     k8sclientset.Interface
	dynamicClient dynamic.Interface
}

func NewKubeClient(client client.Client, clientset k8sclientset.Interface, fieldManager string,
	dynamicClient dynamic.Interface) Client {
	return &KubeClient{
		client:        client,
		clientset:     clientset,
		fieldManager:  fieldManager,
		dynamicClient: dynamicClient,
	}
}

func (c *KubeClient) GetService(ctx context.Context, name, namespace string) (*corev1.Service, error) {
	svc := &corev1.Service{}
	if err := c.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, svc); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return svc, nil
}

func (c *KubeClient) GetDeployment(ctx context.Context, name, namespace string) (*v1.Deployment, error) {
	deployment := &v1.Deployment{}
	if err := c.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deployment); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return deployment, nil
}

func (c *KubeClient) UpdateDeployment(ctx context.Context, deployment *v1.Deployment) error {
	return c.client.Update(ctx, deployment)
}

func (c *KubeClient) DeleteDeployment(ctx context.Context, name, namespace string) error {
	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := c.client.Delete(ctx, deployment); err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

func (c *KubeClient) DeleteClusterRole(ctx context.Context, name, namespace string) error {
	role := &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := c.client.Delete(ctx, role); err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

func (c *KubeClient) DeleteClusterRoleBinding(ctx context.Context, name, namespace string) error {
	binding := &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := c.client.Delete(ctx, binding); err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

func (c *KubeClient) GetNATSResources(ctx context.Context, namespace string) (*natsv1alpha1.NATSList, error) {
	natsList := &natsv1alpha1.NATSList{}
	err := c.client.List(ctx, natsList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	return natsList, nil
}

// PatchApply uses the server-side apply to create/update the resource.
// The object must define `GVK` (i.e. object.TypeMeta).
func (c *KubeClient) PatchApply(ctx context.Context, object client.Object) error {
	return c.client.Patch(ctx, object, client.Apply, &client.PatchOptions{
		Force:        pointer.Bool(true),
		FieldManager: c.fieldManager,
	})
}

// PatchApplyUnstructured uses the server-side apply to create/update the resource.
// It will work even if the Scheme for resource is not defined in client.
// The object must define `GVK` (i.e. object.TypeMeta).
func (c *KubeClient) PatchApplyUnstructured(ctx context.Context, object *unstructured.Unstructured) error {
	return c.client.Patch(ctx, object, client.Apply, &client.PatchOptions{
		Force:        pointer.Bool(true),
		FieldManager: c.fieldManager,
	})
}

// GetSecret returns the secret with the given namespaced name.
// namespacedName is in the format of "namespace/name".
func (c *KubeClient) GetSecret(ctx context.Context, namespacedName string) (*corev1.Secret, error) {
	substrings := strings.Split(namespacedName, "/")
	if len(substrings) != 2 {
		return nil, errors.New("invalid namespaced name. It must be in the format of 'namespace/name'")
	}
	secret := &corev1.Secret{}
	err := c.client.Get(ctx, client.ObjectKey{
		Namespace: substrings[0],
		Name:      substrings[1],
	}, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (c *KubeClient) GetCRD(ctx context.Context, name string) (*apiextensionsv1.CustomResourceDefinition, error) {
	return c.clientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
}

func (c *KubeClient) ApplicationCRDExists(ctx context.Context) (bool, error) {
	_, err := c.GetCRD(ctx, ApplicationCrdName)
	if err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

// GetMutatingWebHookConfiguration returns the MutatingWebhookConfiguration k8s resource.
func (c *KubeClient) GetMutatingWebHookConfiguration(ctx context.Context,
	name string) (*admissionv1.MutatingWebhookConfiguration, error) {
	var mutatingWH admissionv1.MutatingWebhookConfiguration
	mutatingWHKey := client.ObjectKey{
		Name: name,
	}
	if err := c.client.Get(ctx, mutatingWHKey, &mutatingWH); err != nil {
		return nil, err
	}

	return &mutatingWH, nil
}

// GetValidatingWebHookConfiguration returns the ValidatingWebhookConfiguration k8s resource.
func (c *KubeClient) GetValidatingWebHookConfiguration(ctx context.Context,
	name string) (*admissionv1.ValidatingWebhookConfiguration, error) {
	var validatingWH admissionv1.ValidatingWebhookConfiguration
	validatingWHKey := client.ObjectKey{
		Name: name,
	}
	if err := c.client.Get(ctx, validatingWHKey, &validatingWH); err != nil {
		return nil, err
	}
	return &validatingWH, nil
}

// GetVirtualService returns the Istio VirtualService k8s resource.
func (c *KubeClient) GetVirtualService(ctx context.Context, name, namespace string) (*networkingv1alpha3.VirtualService, error) {
	gvk := schema.GroupVersionResource{
		Group:    networkingv1alpha3.SchemeGroupVersion.Group,
		Version:  networkingv1alpha3.SchemeGroupVersion.Version,
		Resource: "virtualservices",
	}

	unstructuredObj, err := c.dynamicClient.Resource(gvk).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	result := &networkingv1alpha3.VirtualService{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetRequestAuthentication returns the Istio RequestAuthentication k8s resource.
func (c *KubeClient) GetRequestAuthentication(ctx context.Context, name, namespace string) (*clientsecurityv1beta1.RequestAuthentication, error) {
	gvk := schema.GroupVersionResource{
		Group:    clientsecurityv1beta1.SchemeGroupVersion.Group,
		Version:  clientsecurityv1beta1.SchemeGroupVersion.Version,
		Resource: "requestauthentications",
	}

	unstructuredObj, err := c.dynamicClient.Resource(gvk).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	result := &clientsecurityv1beta1.RequestAuthentication{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetAuthorizationPolicy returns the Istio AuthorizationPolicy k8s resource.
func (c *KubeClient) GetAuthorizationPolicy(ctx context.Context, name, namespace string) (*clientsecurityv1beta1.AuthorizationPolicy, error) {
	gvk := schema.GroupVersionResource{
		Group:    clientsecurityv1beta1.SchemeGroupVersion.Group,
		Version:  clientsecurityv1beta1.SchemeGroupVersion.Version,
		Resource: "authorizationpolicies",
	}

	unstructuredObj, err := c.dynamicClient.Resource(gvk).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	result := &clientsecurityv1beta1.AuthorizationPolicy{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
