package utils

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"

	kapiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kapps "k8s.io/api/apps/v1"
	krbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
)

const (
	charset       = "abcdefghijklmnopqrstuvwxyz0123456789"
	randomNameLen = 5

	Domain               = "domain.com"
	NameFormat           = "name-%s"
	NamespaceFormat      = "namespace-%s"
	PublisherProxySuffix = "publisher-proxy"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec,gochecknoglobals // used in tests

func GetRandString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

type EventingOption func(*v1alpha1.Eventing) error

func NewNamespace(name string) *kcore.Namespace {
	namespace := kcore.Namespace{
		ObjectMeta: kmeta.ObjectMeta{
			Name: name,
		},
	}
	return &namespace
}

func NewApplicationCRD() *kapiextensions.CustomResourceDefinition {
	result := &kapiextensions.CustomResourceDefinition{
		TypeMeta: kmeta.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name: "applications.applicationconnector.kyma-project.io",
		},
		Spec: kapiextensions.CustomResourceDefinitionSpec{
			Names:                 kapiextensions.CustomResourceDefinitionNames{},
			Scope:                 "Namespaced",
			PreserveUnknownFields: false,
		},
	}

	return result
}

func NewPeerAuthenticationCRD() (*kapiextensions.CustomResourceDefinition, error) {
	crdYAML, err := os.ReadFile("../../config/crd/for-tests/security.istio.io_peerauthentication.yaml")
	if err != nil {
		return nil, err
	}

	crd := &kapiextensions.CustomResourceDefinition{}
	decoder := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	if _, _, err = decoder.Decode(crdYAML, nil, crd); err != nil {
		return nil, err
	}
	return crd, nil
}

func NewAPIRuleCRD() *kapiextensions.CustomResourceDefinition {
	result := &kapiextensions.CustomResourceDefinition{
		TypeMeta: kmeta.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name: "apirules.gateway.kyma-project.io",
		},
		Spec: kapiextensions.CustomResourceDefinitionSpec{
			Names:                 kapiextensions.CustomResourceDefinitionNames{},
			Scope:                 "Namespaced",
			PreserveUnknownFields: false,
		},
	}

	return result
}

func NewEventingCR(opts ...EventingOption) *v1alpha1.Eventing {
	name := fmt.Sprintf(NameFormat, GetRandString(randomNameLen))
	namespace := fmt.Sprintf(NamespaceFormat, GetRandString(randomNameLen))

	eventing := &v1alpha1.Eventing{
		TypeMeta: kmeta.TypeMeta{
			Kind:       "Eventing",
			APIVersion: "operator.kyma-project.io/v1alpha1",
		},
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       "1234-5678-1234-5678",
		},
		Spec: v1alpha1.EventingSpec{
			Backend: v1alpha1.Backend{
				Type: v1alpha1.NatsBackendType,
			},
		},
	}

	for _, opt := range opts {
		if err := opt(eventing); err != nil {
			panic(err)
		}
	}

	return eventing
}

func HasOwnerReference(object client.Object, eventingCR v1alpha1.Eventing) bool {
	ownerReferences := object.GetOwnerReferences()

	return len(ownerReferences) > 0 && ownerReferences[0].Name == eventingCR.Name &&
		ownerReferences[0].Kind == "Eventing" &&
		ownerReferences[0].UID == eventingCR.UID
}

func IsEPPPublishServiceCorrect(svc kcore.Service, eppDeployment kapps.Deployment) bool {
	wantSpec := kcore.ServiceSpec{
		Selector: eppDeployment.Spec.Template.Labels,
		Ports: []kcore.ServicePort{
			{
				Name:       "http-client",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			},
		},
	}

	return reflect.DeepEqual(wantSpec.Selector, svc.Spec.Selector) &&
		reflect.DeepEqual(wantSpec.Ports, svc.Spec.Ports)
}

func IsEPPMetricsServiceCorrect(svc kcore.Service, eppDeployment kapps.Deployment) bool {
	wantAnnotations := map[string]string{
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "9090",
		"prometheus.io/scheme": "http",
	}

	wantSpec := kcore.ServiceSpec{
		Selector: eppDeployment.Spec.Template.Labels,
		Ports: []kcore.ServicePort{
			{
				Name:       "http-metrics",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(9090),
			},
		},
	}

	return reflect.DeepEqual(wantSpec.Selector, svc.Spec.Selector) &&
		reflect.DeepEqual(wantSpec.Ports, svc.Spec.Ports) &&
		reflect.DeepEqual(wantAnnotations, svc.Annotations)
}

func IsEPPHealthServiceCorrect(svc kcore.Service, eppDeployment kapps.Deployment) bool {
	wantSpec := kcore.ServiceSpec{
		Selector: eppDeployment.Spec.Template.Labels,
		Ports: []kcore.ServicePort{
			{
				Name:       "http-status",
				Protocol:   "TCP",
				Port:       15020,
				TargetPort: intstr.FromInt(15020),
			},
		},
	}

	return reflect.DeepEqual(wantSpec.Selector, svc.Spec.Selector) &&
		reflect.DeepEqual(wantSpec.Ports, svc.Spec.Ports)
}

func IsEPPClusterRoleCorrect(clusterRole krbac.ClusterRole) bool {
	wantRules := []krbac.PolicyRule{
		{
			APIGroups: []string{"eventing.kyma-project.io"},
			Resources: []string{"subscriptions"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"applicationconnector.kyma-project.io"},
			Resources: []string{"applications"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}
	return reflect.DeepEqual(wantRules, clusterRole.Rules)
}

func IsEPPClusterRoleBindingCorrect(clusterRoleBinding krbac.ClusterRoleBinding, eventingCR v1alpha1.Eventing) bool {
	wantRoleRef := krbac.RoleRef{
		Kind:     "ClusterRole",
		Name:     fmt.Sprintf("%s-%s", eventingCR.GetName(), PublisherProxySuffix),
		APIGroup: "rbac.authorization.k8s.io",
	}
	wantSubjects := []krbac.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      fmt.Sprintf("%s-%s", eventingCR.GetName(), PublisherProxySuffix),
			Namespace: eventingCR.Namespace,
		},
	}

	return reflect.DeepEqual(wantRoleRef, clusterRoleBinding.RoleRef) &&
		reflect.DeepEqual(wantSubjects, clusterRoleBinding.Subjects)
}

func NewDeployment(name, namespace string, annotations map[string]string) *kapps.Deployment {
	return &kapps.Deployment{
		ObjectMeta: kmeta.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: kapps.DeploymentSpec{
			Template: kcore.PodTemplateSpec{
				ObjectMeta: kmeta.ObjectMeta{
					Name:        name,
					Namespace:   namespace,
					Annotations: annotations,
				},
				Spec: kcore.PodSpec{
					Containers: []kcore.Container{
						{
							Name:  "publisher",
							Image: "test-image",
						},
					},
				},
			},
		},
	}
}

func NewEventMeshSecret(name, namespace string) *kcore.Secret {
	return &kcore.Secret{
		ObjectMeta: kmeta.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string][]byte{
			"management": []byte("foo"),
			"messaging": []byte(`[
			  {
				"broker": {
				  "type": "bar"
				},
				"oa2": {
				  "clientid": "foo",
				  "clientsecret": "foo",
				  "granttype": "client_credentials",
				  "tokenendpoint": "bar"
				},
				"protocol": [
				  "amqp10ws"
				],
				"uri": "foo"
			  },
			  {
				"broker": {
				  "type": "foo"
				},
				"oa2": {
				  "clientid": "bar",
				  "clientsecret": "bar",
				  "granttype": "client_credentials",
				  "tokenendpoint": "foo"
				},
				"protocol": [
				  "bar"
				],
				"uri": "bar"
			  },
			  {
				"broker": {
				  "type": "foo"
				},
				"oa2": {
				  "clientid": "foo",
				  "clientsecret": "bar",
				  "granttype": "client_credentials",
				  "tokenendpoint": "foo"
				},
				"protocol": [
				  "httprest"
				],
				"uri": "bar"
			  }
			]`),
			"namespace":         []byte("bar"),
			"serviceinstanceid": []byte("foo"),
			"xsappname":         []byte("bar"),
		},
		Type: "Opaque",
	}
}

func NewOAuthSecret(name, namespace string) *kcore.Secret {
	secret := &kcore.Secret{
		ObjectMeta: kmeta.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string][]byte{
			"client_id":     []byte("foo"),
			"client_secret": []byte("bar"),
			"token_url":     []byte("token-url"),
			"certs_url":     []byte("certs-url"),
		},
		Type: "Opaque",
	}
	return secret
}

func NewSubscription(name, namespace string) *eventingv1alpha2.Subscription {
	return &eventingv1alpha2.Subscription{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: eventingv1alpha2.SubscriptionSpec{
			Sink:   "test-sink",
			Source: "test-source",
			Types:  []string{"test1.nats.type", "test2.nats.type"},
		},
	}
}

func NewConfigMap(name, namespace string) *kcore.ConfigMap {
	cm := &kcore.ConfigMap{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return cm
}

func FindObjectByKind(kind string, objects []client.Object) (client.Object, error) {
	for _, obj := range objects {
		if obj.GetObjectKind().GroupVersionKind().Kind == kind {
			return obj, nil
		}
	}

	return nil, errors.New("not found")
}

func FindServiceFromK8sObjects(name string, objects []client.Object) (client.Object, error) {
	for _, obj := range objects {
		if obj.GetObjectKind().GroupVersionKind().Kind == "Service" &&
			obj.GetName() == name {
			return obj, nil
		}
	}

	return nil, errors.New("not found")
}
