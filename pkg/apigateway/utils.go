package apigateway

import (
	"fmt"
	"reflect"
	"strings"

	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// GetServiceNameFromSink returns (name, namespace) of the service from the URL.
func GetServiceNameFromSink(url string) (string, string, error) {
	parts := strings.Split(url, ".")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid sinkURL for cluster local svc: %s", url)
	}
	return parts[0], parts[1], nil
}

func GenerateWebhookHostName(name, domain string) string {
	return fmt.Sprintf("%s-%s.%s", externalHostPrefix, name, domain)
}

func GenerateServiceURI(name, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", name, namespace)
}

func HasOwnerReference(list []metav1.OwnerReference, subscription eventingv1alpha2.Subscription) bool {
	wantOwnerReference := metav1.OwnerReference{
		APIVersion:         subscription.APIVersion,
		Kind:               subscription.Kind,
		Name:               subscription.Name,
		UID:                subscription.UID,
		BlockOwnerDeletion: pointer.Bool(true),
		Controller:         pointer.Bool(false),
	}

	for _, o := range list {
		if reflect.DeepEqual(o, wantOwnerReference) {
			return true
		}
	}
	return false
}

func AppendOwnerReference(list []metav1.OwnerReference, newOwner metav1.OwnerReference) []metav1.OwnerReference {
	var result []metav1.OwnerReference
	for _, o := range list {
		if o.UID != newOwner.UID &&
			o.Name != newOwner.Name &&
			o.Kind != newOwner.Kind &&
			o.APIVersion != newOwner.APIVersion {
			result = append(result, o)
		}
	}
	return append(result, newOwner)
}
