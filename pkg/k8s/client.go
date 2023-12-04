package k8s

import (
	"context"
	"errors"
	"strings"

	istiosec "istio.io/client-go/pkg/apis/security/v1beta1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	kappsv1 "k8s.io/api/apps/v1"
	kcorev1 "k8s.io/api/core/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	kapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kapiclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
)

var NatsGVK = schema.GroupVersionResource{
	Group:    natsv1alpha1.GroupVersion.Group,
	Version:  natsv1alpha1.GroupVersion.Version,
	Resource: "nats",
}

//go:generate go run github.com/vektra/mockery/v2 --name=Client --outpkg=mocks --case=underscore
type Client interface {
	GetDeployment(ctx context.Context, name, namespace string) (*kappsv1.Deployment, error)
	GetDeploymentDynamic(ctx context.Context, name, namespace string) (*kappsv1.Deployment, error)
	UpdateDeployment(ctx context.Context, deployment *kappsv1.Deployment) error
	DeleteDeployment(ctx context.Context, name, namespace string) error
	DeleteClusterRole(ctx context.Context, name, namespace string) error
	DeleteClusterRoleBinding(ctx context.Context, name, namespace string) error
	DeleteResource(ctx context.Context, object client.Object) error
	GetNATSResources(ctx context.Context, namespace string) (*natsv1alpha1.NATSList, error)
	PatchApply(ctx context.Context, object client.Object) error
	GetSecret(ctx context.Context, namespacedName string) (*kcorev1.Secret, error)
	GetMutatingWebHookConfiguration(ctx context.Context, name string) (*admissionv1.MutatingWebhookConfiguration, error)
	GetValidatingWebHookConfiguration(ctx context.Context,
		name string) (*admissionv1.ValidatingWebhookConfiguration, error)
	GetCRD(ctx context.Context, name string) (*kapiextensionsv1.CustomResourceDefinition, error)
	ApplicationCRDExists(ctx context.Context) (bool, error)
	PeerAuthenticationCRDExists(ctx context.Context) (bool, error)
	APIRuleCRDExists(ctx context.Context) (bool, error)
	GetSubscriptions(ctx context.Context) (*eventingv1alpha2.SubscriptionList, error)
	GetConfigMap(ctx context.Context, name, namespace string) (*kcorev1.ConfigMap, error)
	PatchApplyPeerAuthentication(ctx context.Context, authentication *istiosec.PeerAuthentication) error
}

type KubeClient struct {
	fieldManager  string
	client        client.Client
	clientset     kapiclientset.Interface
	dynamicClient dynamic.Interface
}

func NewKubeClient(client client.Client, clientset kapiclientset.Interface, fieldManager string,
	dynamicClient dynamic.Interface) Client {
	return &KubeClient{
		client:        client,
		clientset:     clientset,
		fieldManager:  fieldManager,
		dynamicClient: dynamicClient,
	}
}

// PatchApplyPeerAuthentication creates the Istio PeerAuthentications.
func (c *KubeClient) PatchApplyPeerAuthentication(ctx context.Context, pa *istiosec.PeerAuthentication) error {
	// patch apply as unstructured because the GVK is not registered in Scheme.
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pa)
	if err != nil {
		return err
	}
	return c.PatchApply(ctx, &unstructured.Unstructured{Object: obj})
}

func (c *KubeClient) GetDeployment(ctx context.Context, name, namespace string) (*kappsv1.Deployment, error) {
	deployment := &kappsv1.Deployment{}
	if err := c.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deployment); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return deployment, nil
}

func (c *KubeClient) GetDeploymentDynamic(ctx context.Context, name, namespace string) (*kappsv1.Deployment, error) {
	deploymentRes := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	result, err := c.dynamicClient.Resource(deploymentRes).Namespace(namespace).Get(ctx, name, kmetav1.GetOptions{})
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	deployment := &kappsv1.Deployment{}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(result.Object, deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

func (c *KubeClient) UpdateDeployment(ctx context.Context, deployment *kappsv1.Deployment) error {
	return c.client.Update(ctx, deployment)
}

func (c *KubeClient) DeleteDeployment(ctx context.Context, name, namespace string) error {
	deployment := &kappsv1.Deployment{
		ObjectMeta: kmetav1.ObjectMeta{
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
	role := &krbacv1.ClusterRole{
		ObjectMeta: kmetav1.ObjectMeta{
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
	binding := &krbacv1.ClusterRoleBinding{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := c.client.Delete(ctx, binding); err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

func (c *KubeClient) DeleteResource(ctx context.Context, object client.Object) error {
	if err := c.client.Delete(ctx, object); err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

func (c *KubeClient) GetNATSResources(ctx context.Context, namespace string) (*natsv1alpha1.NATSList, error) {
	unstructuredList, err := c.dynamicClient.Resource(NatsGVK).Namespace(namespace).List(ctx, kmetav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	natsList := &natsv1alpha1.NATSList{
		Items: []natsv1alpha1.NATS{},
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredList.Object, natsList)
	if err != nil {
		return nil, err
	}

	for _, item := range unstructuredList.Items {
		nats := &natsv1alpha1.NATS{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, nats)
		if err != nil {
			return nil, err
		}
		natsList.Items = append(natsList.Items, *nats)
	}
	return natsList, nil
}

// PatchApply uses the server-side apply to create/update the resource.
// The object must define `GVK` (i.e. object.TypeMeta).
func (c *KubeClient) PatchApply(ctx context.Context, object client.Object) error {
	return c.client.Patch(ctx, object, client.Apply, &client.PatchOptions{
		Force:        ptr.To(true),
		FieldManager: c.fieldManager,
	})
}

// GetSecret returns the secret with the given namespaced name.
// namespacedName is in the format of "namespace/name".
func (c *KubeClient) GetSecret(ctx context.Context, namespacedName string) (*kcorev1.Secret, error) {
	substrings := strings.Split(namespacedName, "/")
	if len(substrings) != 2 {
		return nil, errors.New("invalid namespaced name. It must be in the format of 'namespace/name'")
	}
	secret := &kcorev1.Secret{}
	err := c.client.Get(ctx, client.ObjectKey{
		Namespace: substrings[0],
		Name:      substrings[1],
	}, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (c *KubeClient) GetCRD(ctx context.Context, name string) (*kapiextensionsv1.CustomResourceDefinition, error) {
	return c.clientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, kmetav1.GetOptions{})
}

func (c *KubeClient) ApplicationCRDExists(ctx context.Context) (bool, error) {
	_, err := c.GetCRD(ctx, ApplicationCrdName)
	if err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

func (c *KubeClient) PeerAuthenticationCRDExists(ctx context.Context) (bool, error) {
	_, err := c.GetCRD(ctx, PeerAuthenticationCRDName)
	if err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

func (c *KubeClient) APIRuleCRDExists(ctx context.Context) (bool, error) {
	_, err := c.GetCRD(ctx, APIRuleCrdName)
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

func (c *KubeClient) GetSubscriptions(ctx context.Context) (*eventingv1alpha2.SubscriptionList, error) {
	subscriptions := &eventingv1alpha2.SubscriptionList{}
	err := c.client.List(ctx, subscriptions)
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// GetConfigMap returns a ConfigMap based on the given name and namespace.
func (c *KubeClient) GetConfigMap(ctx context.Context, name, namespace string) (*kcorev1.ConfigMap, error) {
	cm := &kcorev1.ConfigMap{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := c.client.Get(ctx, key, cm); err != nil {
		return nil, err
	}
	return cm, nil
}
