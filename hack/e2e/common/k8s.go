package common

import (
	"os"
	"path/filepath"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha1"
	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
)

func GetK8sClients() (*kubernetes.Clientset, client.Client, *dynamic.DynamicClient, error) {
	kubeConfigPath := ""
	if _, ok := os.LookupEnv("KUBECONFIG"); ok {
		kubeConfigPath = os.Getenv("KUBECONFIG")
	} else {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, nil, nil, err
		}
		kubeConfigPath = filepath.Join(userHomeDir, ".kube", "config")
	}

	var kubeConfig *rest.Config
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, nil, nil, err
	}

	// Set up the clientSet that is used to access regular K8s objects.
	var clientSet *kubernetes.Clientset
	clientSet, err = kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	// We need to add the NATS CRD to the scheme, so we can create a client that can access NATS objects.
	err = natsv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, nil, nil, err
	}

	// add Subscription v1alpha1 CRD to the scheme.
	err = eventingv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, nil, nil, err
	}

	// add Subscription v1alpha2 CRD to the scheme.
	err = eventingv1alpha2.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, nil, nil, err
	}

	// We need to add the Eventing CRD to the scheme, so we can create a client that can access Eventing objects.
	err = operatorv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, nil, nil, err
	}

	// Set up the k8s client, so we can access NATS CR-objects.
	// +kubebuilder:scaffold:scheme
	var k8sClient client.Client
	k8sClient, err = client.New(kubeConfig, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, nil, nil, err
	}

	// set up dynamic client
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	return clientSet, k8sClient, dynamicClient, nil
}
