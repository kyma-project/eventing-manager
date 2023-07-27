package k8s

import (
	"context"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockery --name=Client --outpkg=mocks --case=underscore
type Client interface {
	GetDeployment(context.Context, string, string) (*v1.Deployment, error)
	GetNATSResources(context.Context, string) (*natsv1alpha1.NATSList, error)
	PatchApply(context.Context, client.Object) error
	PeerAuthenticationCRDExists(context.Context) (bool, error)
	GetCRD(context.Context, string) (*apiextensionsv1.CustomResourceDefinition, error)
}

type KubeClient struct {
	fieldManager string
	client       client.Client
	clientset    k8sclientset.Interface
}

func NewKubeClient(client client.Client, clientset k8sclientset.Interface, fieldManager string) Client {
	return &KubeClient{
		client:       client,
		clientset:    clientset,
		fieldManager: fieldManager,
	}
}

func (c *KubeClient) GetDeployment(ctx context.Context, name, namespace string) (*v1.Deployment, error) {
	deployment := &v1.Deployment{}
	err := c.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deployment)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func (c *KubeClient) GetNATSResources(ctx context.Context, namespace string) (*natsv1alpha1.NATSList, error) {
	natsList := &natsv1alpha1.NATSList{}
	err := c.client.List(ctx, natsList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	return natsList, nil
}

func (c *KubeClient) GetCRD(ctx context.Context, name string) (*apiextensionsv1.CustomResourceDefinition, error) {
	return c.clientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
}

func (c *KubeClient) PeerAuthenticationCRDExists(ctx context.Context) (bool, error) {
	_, err := c.GetCRD(ctx, PeerAuthenticationCrdName)
	if err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

// PatchApply uses the server-side apply to create/update the resource.
// The object must define `GVK` (i.e. object.TypeMeta).
func (c *KubeClient) PatchApply(ctx context.Context, object client.Object) error {
	return c.client.Patch(ctx, object, client.Apply, &client.PatchOptions{
		Force:        pointer.Bool(true),
		FieldManager: c.fieldManager,
	})
}
