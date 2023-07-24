package k8s

import (
	"context"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	GetDeployment(context.Context, string, string) (*v1.Deployment, error)
	GetNATSResources(context.Context, string) (*natsv1alpha1.NATSList, error)
}

type KubeClient struct {
	client.Client
}

func NewKubeClient(client client.Client) Client {
	return &KubeClient{client}
}

func (c *KubeClient) GetDeployment(ctx context.Context, name, namespace string) (*v1.Deployment, error) {
	deployment := &v1.Deployment{}
	err := c.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deployment)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func (c *KubeClient) GetNATSResources(ctx context.Context, namespace string) (*natsv1alpha1.NATSList, error) {
	natsList := &natsv1alpha1.NATSList{}
	err := c.List(ctx, natsList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	return natsList, nil
}
