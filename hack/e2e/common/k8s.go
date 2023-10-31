package common

import (
	"os"
	"path/filepath"

	ecv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	ecv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
)

func GetRestConfig() (*rest.Config, error) {
	kubeConfigPath := ""
	if _, ok := os.LookupEnv("KUBECONFIG"); ok {
		kubeConfigPath = os.Getenv("KUBECONFIG")
	} else {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeConfigPath = filepath.Join(userHomeDir, ".kube", "config")
	}

	return clientcmd.BuildConfigFromFlags("", kubeConfigPath)
}

func GetK8sClients() (*kubernetes.Clientset, client.Client, error) {
	kubeConfig, err := GetRestConfig()
	if err != nil {
		return nil, nil, err
	}

	// Set up the clientSet that is used to access regular K8s objects.
	var clientSet *kubernetes.Clientset
	clientSet, err = kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, err
	}

	// We need to add the NATS CRD to the scheme, so we can create a client that can access NATS objects.
	err = natsv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, nil, err
	}

	// add Subscription v1alpha1 CRD to the scheme.
	err = ecv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, nil, err
	}

	// add Subscription v1alpha2 CRD to the scheme.
	err = ecv1alpha2.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, nil, err
	}

	// We need to add the Eventing CRD to the scheme, so we can create a client that can access Eventing objects.
	err = eventingv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, nil, err
	}

	// Set up the k8s client, so we can access NATS CR-objects.
	// +kubebuilder:scaffold:scheme
	var k8sClient client.Client
	k8sClient, err = client.New(kubeConfig, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, nil, err
	}

	return clientSet, k8sClient, nil
}


