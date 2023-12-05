package v1alpha2

import (
	"encoding/json"

	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kschema "k8s.io/apimachinery/pkg/runtime/schema"
)

func SubscriptionGroupVersionResource() kschema.GroupVersionResource {
	return kschema.GroupVersionResource{
		Version:  GroupVersion.Version,
		Group:    GroupVersion.Group,
		Resource: "subscriptions",
	}
}

func ConvertUnstructListToSubList(unstructuredList *kunstructured.UnstructuredList) (*SubscriptionList, error) {
	subscriptionList := new(SubscriptionList)
	subscriptionListBytes, err := unstructuredList.MarshalJSON()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(subscriptionListBytes, subscriptionList)
	if err != nil {
		return nil, err
	}
	return subscriptionList, nil
}
