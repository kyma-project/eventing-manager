package utils

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	charset       = "abcdefghijklmnopqrstuvwxyz0123456789"
	randomNameLen = 5

	NameFormat      = "name-%s"
	NamespaceFormat = "namespace-%s"
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

func NewNamespace(name string) *v1.Namespace {
	namespace := v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return &namespace
}

func NewEventingCR(opts ...EventingOption) *v1alpha1.Eventing {
	name := fmt.Sprintf(NameFormat, GetRandString(randomNameLen))
	namespace := fmt.Sprintf(NamespaceFormat, GetRandString(randomNameLen))

	eventing := &v1alpha1.Eventing{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1alpha1",
			Kind:       "Eventing",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       "1234-5678-1234-5678",
		},
		Spec: v1alpha1.EventingSpec{
			Backends: []v1alpha1.Backend{
				{
					Type: v1alpha1.NatsBackendType,
				},
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
