package k8s

import (
	"context"

	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	GetDeployment(context.Context, string, string) (*v1.Deployment, error)
}

type KubeClient struct {
	client client.Client
}

func NewKubeClient(client client.Client) Client {
	return &KubeClient{client: client}
}

func (c *KubeClient) GetDeployment(ctx context.Context, name, namespace string) (*v1.Deployment, error) {
	deployment := &v1.Deployment{}
	err := c.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deployment)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}
