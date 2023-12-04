package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"

	ceevent "github.com/cloudevents/sdk-go/v2/event"
	"github.com/nats-io/nats.go"

	apigatewayv1beta1 "github.com/kyma-incubator/api-gateway/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type EventTypeInfo struct {
	OriginalType  string
	CleanType     string
	ProcessedType string
}

// NameMapper is used to map Kyma-specific resource names to their corresponding name on other
// (external) systems, e.g. on different eventing backends, the same Kyma subscription name
// could map to a different name.
type NameMapper interface {
	MapSubscriptionName(subscriptionName, subscriptionNamespace string) string
}

func APIRuleGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Version:  apigatewayv1beta1.GroupVersion.Version,
		Group:    apigatewayv1beta1.GroupVersion.Group,
		Resource: "apirules",
	}
}

func ConvertMsgToCE(msg *nats.Msg) (*ceevent.Event, error) {
	event := ceevent.New(ceevent.CloudEventsVersionV1)
	err := json.Unmarshal(msg.Data, &event)
	if err != nil {
		return nil, err
	}
	if err := event.Validate(); err != nil {
		return nil, err
	}
	return &event, nil
}

func GetExposedURLFromAPIRule(apiRule *apigatewayv1beta1.APIRule, targetURL string) (string, error) {
	// @TODO: Move this method to backend/eventmesh/utils.go once old BEB backend is depreciated
	scheme := "https://"
	path := ""

	sURL, err := url.ParseRequestURI(targetURL)
	if err != nil {
		return "", err
	}
	sURLPath := sURL.Path
	if sURL.Path == "" {
		sURLPath = "/"
	}
	for _, rule := range apiRule.Spec.Rules {
		if rule.Path == sURLPath {
			path = rule.Path
			break
		}
	}
	return fmt.Sprintf("%s%s%s", scheme, *apiRule.Spec.Host, path), nil
}

// UpdateSubscriptionStatus updates the status of all Kyma subscriptions on k8s.
func UpdateSubscriptionStatus(ctx context.Context, dClient dynamic.Interface,
	sub *eventingv1alpha2.Subscription) error {
	unstructuredObj, err := sub.ToUnstructuredSub()
	if err != nil {
		return errors.Wrap(err, "convert subscription to unstructured failed")
	}
	_, err = dClient.
		Resource(eventingv1alpha2.SubscriptionGroupVersionResource()).
		Namespace(sub.Namespace).
		UpdateStatus(ctx, unstructuredObj, kmetav1.UpdateOptions{})

	return err
}

// LoggerWithSubscription returns a logger with the given subscription (v1alpha2) details.
func LoggerWithSubscription(log *zap.SugaredLogger,
	subscription *eventingv1alpha2.Subscription) *zap.SugaredLogger {
	return log.With(
		"kind", subscription.GetObjectKind().GroupVersionKind().Kind,
		"version", subscription.GetGeneration(),
		"namespace", subscription.GetNamespace(),
		"name", subscription.GetName(),
	)
}
