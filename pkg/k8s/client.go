package k8s

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
)

//go:generate mockery --name=Client --outpkg=mocks --case=underscore
type Client interface{}

type KubeClient struct {
	client       client.Client
	clientset    k8sclientset.Interface
	fieldManager string
}

func NewKubeClient(client client.Client, clientset k8sclientset.Interface, fieldManager string) Client {
	return &KubeClient{
		client:       client,
		clientset:    clientset,
		fieldManager: fieldManager,
	}
}
