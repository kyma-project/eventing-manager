package testing

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	apigatewayv1beta1 "github.com/kyma-project/api-gateway/apis/gateway/v1beta1"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	kschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	kdynamicfake "k8s.io/client-go/dynamic/fake"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/ems/api/events/types"
	"github.com/kyma-project/eventing-manager/pkg/object"
)

const (
	ApplicationName         = "testapp1023"
	ApplicationNameNotClean = "test-app_1-0+2=3"

	OrderCreatedUncleanEvent = "order.cre-ä+t*ed.v2"
	OrderCreatedCleanEvent   = "order.cre-ä+ted.v2"
	EventSourceUnclean       = "s>o>*u*r>c.e"
	EventSourceClean         = "source"

	EventMeshProtocol = "BEB"

	EventMeshNamespaceNS        = "/default/ns"
	EventMeshNamespace          = "/default/kyma/id"
	EventSource                 = "/default/kyma/id"
	EventTypePrefix             = "prefix"
	EventMeshPrefix             = "one.two.three"      // three segments
	InvalidEventMeshPrefix      = "one.two.three.four" // four segments
	EventTypePrefixEmpty        = ""
	OrderCreatedV1Event         = "order.created.v1"
	OrderCreatedV2Event         = "order.created.v2"
	OrderCreatedV1EventNotClean = "order.c*r%e&a!te#d.v1"
	JetStreamSubject            = "kyma" + "." + EventSourceClean + "." + OrderCreatedV1Event
	JetStreamSubjectV2          = "kyma" + "." + EventSourceClean + "." + OrderCreatedCleanEvent

	EventMeshExactType          = EventMeshPrefix + "." + ApplicationNameNotClean + "." + OrderCreatedV1EventNotClean
	EventMeshOrderCreatedV1Type = EventMeshPrefix + "." + ApplicationName + "." + OrderCreatedV1Event

	OrderCreatedEventType            = EventTypePrefix + "." + ApplicationName + "." + OrderCreatedV1Event
	OrderCreatedEventTypeNotClean    = EventTypePrefix + "." + ApplicationNameNotClean + "." + OrderCreatedV1Event
	OrderCreatedEventTypePrefixEmpty = ApplicationName + "." + OrderCreatedV1Event

	CloudEventType  = EventTypePrefix + "." + ApplicationName + ".order.created.v1"
	CloudEventData  = "{\"foo\":\"bar\"}"
	CloudEventData2 = "{\"foo\":\"bar2\"}"

	JSStreamName = "kyma"

	EventID = "8945ec08-256b-11eb-9928-acde48001122"

	CloudEventSource      = "/default/sap.kyma/id"
	CloudEventSpecVersion = "1.0"

	CeIDHeader          = "ce-id"
	CeTypeHeader        = "ce-type"
	CeSourceHeader      = "ce-source"
	CeSpecVersionHeader = "ce-specversion"
)

type APIRuleOption func(r *apigatewayv1beta1.APIRule)

// GetFreePort determines a free port on the host. It does so by delegating the job to net.ListenTCP.
// Then providing a port of 0 to net.ListenTCP, it will automatically choose a port for us.
func GetFreePort() (int, error) {
	var a *net.TCPAddr
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port //nolint:forcetypeassert // will always return a TCPAddr according to documentation
	l.Close()
	return port, err
}

func NewBEBMessagingSecret(name, namespace string) *kcorev1.Secret {
	messagingValue := `
				[{
					"broker": {
						"type": "sapmgw"
					},
					"oa2": {
						"clientid": "clientid",
						"clientsecret": "clientsecret",
						"granttype": "client_credentials",
						"tokenendpoint": "https://token"
					},
					"protocol": ["amqp10ws"],
					"uri": "wss://amqp"
				}, {
					"broker": {
						"type": "sapmgw"
					},
					"oa2": {
						"clientid": "clientid",
						"clientsecret": "clientsecret",
						"granttype": "client_credentials",
						"tokenendpoint": "https://token"
					},
					"protocol": ["amqp10ws"],
					"uri": "wss://amqp"
				}, {
					"broker": {
						"type": "saprestmgw"
					},
					"oa2": {
						"clientid": "rest-clientid",
						"clientsecret": "rest-client-secret",
						"granttype": "client_credentials",
						"tokenendpoint": "https://rest-token"
					},
					"protocol": ["httprest"],
					"uri": "https://rest-messaging"
				}]`

	return &kcorev1.Secret{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"messaging": messagingValue,
			"namespace": "test/ns",
		},
	}
}

func NewNamespace(name string) *kcorev1.Namespace {
	namespace := kcorev1.Namespace{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: name,
		},
	}
	return &namespace
}

func GetStructuredMessageHeaders() http.Header {
	return http.Header{"Content-Type": []string{"application/cloudevents+json"}}
}

func GetBinaryMessageHeaders() http.Header {
	headers := make(http.Header)
	headers.Add(CeIDHeader, EventID)                        //nolint:canonicalheader,nolintlint // used in testing.
	headers.Add(CeTypeHeader, CloudEventType)               //nolint:canonicalheader,nolintlint // used in testing.
	headers.Add(CeSourceHeader, CloudEventSource)           //nolint:canonicalheader,nolintlint // used in testing.
	headers.Add(CeSpecVersionHeader, CloudEventSpecVersion) //nolint:canonicalheader,nolintlint // used in testing.
	return headers
}

// NewAPIRule returns a valid APIRule.
func NewAPIRule(subscription *eventingv1alpha2.Subscription, opts ...APIRuleOption) *apigatewayv1beta1.APIRule {
	apiRule := &apigatewayv1beta1.APIRule{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: "foo",
			OwnerReferences: []kmetav1.OwnerReference{
				{
					APIVersion: "eventing.kyma-project.io/v1alpha1",
					Kind:       "subscriptions",
					Name:       subscription.Name,
					UID:        subscription.UID,
				},
			},
		},
	}

	for _, opt := range opts {
		opt(apiRule)
	}
	return apiRule
}

func WithService(name, host string) APIRuleOption {
	return func(r *apigatewayv1beta1.APIRule) {
		port := uint32(443) //nolint:mnd // tests
		isExternal := true
		r.Spec.Host = &host
		r.Spec.Service = &apigatewayv1beta1.Service{
			Name:       &name,
			Port:       &port,
			IsExternal: &isExternal,
		}
	}
}

func WithPath() APIRuleOption {
	return func(rule *apigatewayv1beta1.APIRule) {
		handlerOAuth := object.OAuthHandlerNameOAuth2Introspection
		handler := apigatewayv1beta1.Handler{
			Name: handlerOAuth,
		}
		authenticator := &apigatewayv1beta1.Authenticator{
			Handler: &handler,
		}
		rule.Spec.Rules = []apigatewayv1beta1.Rule{
			{
				Path: "/path",
				Methods: []apigatewayv1beta1.HttpMethod{
					apigatewayv1beta1.HttpMethod(http.MethodPost),
					apigatewayv1beta1.HttpMethod(http.MethodOptions),
				},
				AccessStrategies: []*apigatewayv1beta1.Authenticator{
					authenticator,
				},
			},
		}
	}
}

func MarkReady(rule *apigatewayv1beta1.APIRule) {
	statusOK := &apigatewayv1beta1.APIRuleResourceStatus{
		Code:        apigatewayv1beta1.StatusOK,
		Description: "",
	}

	rule.Status = apigatewayv1beta1.APIRuleStatus{
		APIRuleStatus:        statusOK,
		VirtualServiceStatus: statusOK,
		AccessRuleStatus:     statusOK,
	}
}

type SubscriptionOpt func(subscription *eventingv1alpha2.Subscription)

func NewSubscription(name, namespace string, opts ...SubscriptionOpt) *eventingv1alpha2.Subscription {
	newSub := &eventingv1alpha2.Subscription{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: eventingv1alpha2.SubscriptionSpec{
			Config: map[string]string{},
		},
	}
	for _, o := range opts {
		o(newSub)
	}
	return newSub
}

func NewEventMeshSubscription(name, contentMode string, webhookURL string, events types.Events,
	webhookAuth *types.WebhookAuth,
) *types.Subscription {
	return &types.Subscription{
		Name:            name,
		ContentMode:     contentMode,
		Qos:             types.QosAtLeastOnce,
		ExemptHandshake: true,
		Events:          events,
		WebhookAuth:     webhookAuth,
		WebhookURL:      webhookURL,
	}
}

func NewSampleEventMeshSubscription() *types.Subscription {
	eventType := []types.Event{
		{
			Source: EventSource,
			Type:   OrderCreatedEventTypeNotClean,
		},
	}

	return NewEventMeshSubscription("ev2subs1", types.ContentModeStructured, "https://webhook.xxx.com",
		eventType, nil)
}

func WithFakeSubscriptionStatus() SubscriptionOpt {
	return func(s *eventingv1alpha2.Subscription) {
		s.Status.Conditions = []eventingv1alpha2.Condition{
			{
				Type:    "foo",
				Status:  "foo",
				Reason:  "foo-reason",
				Message: "foo-message",
			},
		}
	}
}

func WithSource(source string) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		sub.Spec.Source = source
	}
}

func WithTypes(types []string) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		sub.Spec.Types = types
	}
}

func WithSink(sink string) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		sub.Spec.Sink = sink
	}
}

func WithConditions(conditions []eventingv1alpha2.Condition) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		sub.Status.Conditions = conditions
	}
}

func WithStatus(status bool) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		sub.Status.Ready = status
	}
}

func WithFinalizers(finalizers []string) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		sub.ObjectMeta.Finalizers = finalizers
	}
}

func WithStatusTypes(cleanEventTypes []eventingv1alpha2.EventType) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		if cleanEventTypes == nil {
			sub.Status.InitializeEventTypes()
		} else {
			sub.Status.Types = cleanEventTypes
		}
	}
}

func WithStatusJSBackendTypes(types []eventingv1alpha2.JetStreamTypes) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		sub.Status.Backend.Types = types
	}
}

func WithEmsSubscriptionStatus(status string) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		sub.Status.Backend.EventMeshSubscriptionStatus = &eventingv1alpha2.EventMeshSubscriptionStatus{
			Status: status,
		}
	}
}

func WithWebhookAuthForEventMesh() SubscriptionOpt {
	return func(s *eventingv1alpha2.Subscription) {
		s.Spec.Config = map[string]string{
			eventingv1alpha2.Protocol:                        EventMeshProtocol,
			eventingv1alpha2.ProtocolSettingsContentMode:     "BINARY",
			eventingv1alpha2.ProtocolSettingsExemptHandshake: "true",
			eventingv1alpha2.ProtocolSettingsQos:             "AT_LEAST_ONCE",
			eventingv1alpha2.WebhookAuthType:                 "oauth2",
			eventingv1alpha2.WebhookAuthGrantType:            "client_credentials",
			eventingv1alpha2.WebhookAuthClientID:             "xxx",
			eventingv1alpha2.WebhookAuthClientSecret:         "xxx",
			eventingv1alpha2.WebhookAuthTokenURL:             "https://oauth2.xxx.com/oauth2/token",
			eventingv1alpha2.WebhookAuthScope:                "guid-identifier,root",
		}
	}
}

func WithInvalidProtocolSettingsQos() SubscriptionOpt {
	return func(s *eventingv1alpha2.Subscription) {
		if s.Spec.Config == nil {
			s.Spec.Config = map[string]string{}
		}
		s.Spec.Config[eventingv1alpha2.ProtocolSettingsQos] = "AT_INVALID_ONCE"
	}
}

func WithInvalidWebhookAuthType() SubscriptionOpt {
	return func(s *eventingv1alpha2.Subscription) {
		if s.Spec.Config == nil {
			s.Spec.Config = map[string]string{}
		}
		s.Spec.Config[eventingv1alpha2.WebhookAuthType] = "abcd"
	}
}

func WithInvalidWebhookAuthGrantType() SubscriptionOpt {
	return func(s *eventingv1alpha2.Subscription) {
		if s.Spec.Config == nil {
			s.Spec.Config = map[string]string{}
		}
		s.Spec.Config[eventingv1alpha2.WebhookAuthGrantType] = "invalid"
	}
}

func WithProtocolEventMesh() SubscriptionOpt {
	return func(s *eventingv1alpha2.Subscription) {
		if s.Spec.Config == nil {
			s.Spec.Config = map[string]string{}
		}
		s.Spec.Config[eventingv1alpha2.Protocol] = EventMeshProtocol
	}
}

// AddEventType adds a new type to the subscription.
func AddEventType(eventType string, subscription *eventingv1alpha2.Subscription) {
	subscription.Spec.Types = append(subscription.Spec.Types, eventType)
}

// WithEventType is a SubscriptionOpt for creating a Subscription with a specific event type,
// that itself gets created from the passed eventType.
func WithEventType(eventType string) SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) { AddEventType(eventType, subscription) }
}

// WithEventSource is a SubscriptionOpt for creating a Subscription with a specific event source,.
func WithEventSource(source string) SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) { subscription.Spec.Source = source }
}

// WithExactTypeMatching is a SubscriptionOpt for creating a Subscription with an exact type matching.
func WithExactTypeMatching() SubscriptionOpt {
	return WithTypeMatching(eventingv1alpha2.TypeMatchingExact)
}

// WithStandardTypeMatching is a SubscriptionOpt for creating a Subscription with a standard type matching.
func WithStandardTypeMatching() SubscriptionOpt {
	return WithTypeMatching(eventingv1alpha2.TypeMatchingStandard)
}

// WithTypeMatching is a SubscriptionOpt for creating a Subscription with a specific type matching,.
func WithTypeMatching(typeMatching eventingv1alpha2.TypeMatching) SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) { subscription.Spec.TypeMatching = typeMatching }
}

// WithNotCleanType initializes subscription with a not clean event-type
// A not clean event-type means it contains none-alphanumeric characters.
func WithNotCleanType() SubscriptionOpt {
	return WithEventType(OrderCreatedV1EventNotClean)
}

func WithEmptyStatus() SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) {
		subscription.Status = eventingv1alpha2.SubscriptionStatus{}
	}
}

func WithEmptyConfig() SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) {
		subscription.Spec.Config = map[string]string{}
	}
}

func WithConfigValue(key, value string) SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) {
		if subscription.Spec.Config == nil {
			subscription.Spec.Config = map[string]string{}
		}
		subscription.Spec.Config[key] = value
	}
}

func WithOrderCreatedFilter() SubscriptionOpt {
	return WithEventType(OrderCreatedEventType)
}

func WithEventMeshExactType() SubscriptionOpt {
	return WithEventType(EventMeshExactType)
}

func WithOrderCreatedV1Event() SubscriptionOpt {
	return WithEventType(OrderCreatedV1Event)
}

func WithDefaultSource() SubscriptionOpt {
	return WithEventSource(ApplicationName)
}

func WithNotCleanSource() SubscriptionOpt {
	return WithEventSource(ApplicationNameNotClean)
}

// WithValidSink is a SubscriptionOpt for creating a subscription with a valid sink that itself gets created from
// the svcNamespace and the svcName.
func WithValidSink(svcNamespace, svcName string) SubscriptionOpt {
	return WithSinkURL(ValidSinkURL(svcNamespace, svcName))
}

// WithSinkURLFromSvc sets a kubernetes service as the sink.
func WithSinkURLFromSvc(svc *kcorev1.Service) SubscriptionOpt {
	return WithSinkURL(ValidSinkURL(svc.Namespace, svc.Name))
}

// ValidSinkURL converts a namespace and service name to a valid sink url.
func ValidSinkURL(namespace, svcName string) string {
	return fmt.Sprintf("https://%s.%s.svc.cluster.local", svcName, namespace)
}

// ValidSinkURLWithPath converts a namespace and service name to a valid sink url with path.
func ValidSinkURLWithPath(namespace, svcName, path string) string {
	return fmt.Sprintf("https://%s.%s.svc.cluster.local/%s", svcName, namespace, path)
}

// WithSinkURL is a SubscriptionOpt for creating a subscription with a specific sink.
func WithSinkURL(sinkURL string) SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) { subscription.Spec.Sink = sinkURL }
}

// WithNonZeroDeletionTimestamp sets the deletion timestamp of the subscription to Now().
func WithNonZeroDeletionTimestamp() SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) {
		now := kmetav1.Now()
		subscription.DeletionTimestamp = &now
	}
}

// SetSink sets the subscription's sink to a valid sink created from svcNameSpace and svcName.
func SetSink(svcNamespace, svcName string, subscription *eventingv1alpha2.Subscription) {
	subscription.Spec.Sink = ValidSinkURL(svcNamespace, svcName)
}

func NewSubscriberSvc(name, namespace string) *kcorev1.Service {
	return &kcorev1.Service{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kcorev1.ServiceSpec{
			Ports: []kcorev1.ServicePort{
				{
					Protocol: "TCP",
					Port:     443, //nolint:mnd // tests
					TargetPort: intstr.IntOrString{
						IntVal: 8080, //nolint:mnd // tests
					},
				},
			},
			Selector: map[string]string{
				"test": "test",
			},
		},
	}
}

// ToSubscription converts an unstructured subscription into a typed one.
func ToSubscription(unstructuredSub *kunstructured.Unstructured) (*eventingv1alpha2.Subscription, error) {
	sub := new(eventingv1alpha2.Subscription)
	err := kruntime.DefaultUnstructuredConverter.FromUnstructured(unstructuredSub.Object, sub)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// ToUnstructuredAPIRule converts an APIRule object into a unstructured APIRule.
func ToUnstructuredAPIRule(obj interface{}) (*kunstructured.Unstructured, error) {
	unstrct := &kunstructured.Unstructured{}
	unstructuredObj, err := kruntime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	unstrct.Object = unstructuredObj
	return unstrct, nil
}

// SetupSchemeOrDie add a scheme to eventing API schemes.
func SetupSchemeOrDie() (*kruntime.Scheme, error) {
	scheme := kruntime.NewScheme()
	if err := kcorev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	if err := eventingv1alpha2.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

// SubscriptionGroupVersionResource returns the GVR of a subscription.
func SubscriptionGroupVersionResource() kschema.GroupVersionResource {
	return kschema.GroupVersionResource{
		Version:  eventingv1alpha2.GroupVersion.Version,
		Group:    eventingv1alpha2.GroupVersion.Group,
		Resource: "subscriptions",
	}
}

// NewFakeSubscriptionClient returns a fake dynamic subscription client.
func NewFakeSubscriptionClient(sub *eventingv1alpha2.Subscription) (dynamic.Interface, error) {
	scheme, err := SetupSchemeOrDie()
	if err != nil {
		return nil, err
	}

	dynamicClient := kdynamicfake.NewSimpleDynamicClient(scheme, sub)
	return dynamicClient, nil
}

// AddSource adds the source value to the subscription.
func AddSource(source string, subscription *eventingv1alpha2.Subscription) {
	subscription.Spec.Source = source
}

// WithSourceAndType is a SubscriptionOpt for creating a Subscription with a specific eventSource and eventType.
func WithSourceAndType(eventSource, eventType string) SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) {
		AddSource(eventSource, subscription)
		AddEventType(eventType, subscription)
	}
}

// WithCleanEventTypeOld is a SubscriptionOpt that initializes subscription with a not clean event type from v1alpha2.
func WithCleanEventTypeOld() SubscriptionOpt {
	return WithSourceAndType(EventSourceClean, OrderCreatedEventType)
}

// WithCleanEventSourceAndType is a SubscriptionOpt that initializes subscription with a not clean event source and
// type.
func WithCleanEventSourceAndType() SubscriptionOpt {
	return WithSourceAndType(EventSourceClean, OrderCreatedV1Event)
}

// WithNotCleanEventSourceAndType is a SubscriptionOpt that initializes subscription with a not clean event source
// and type.
func WithNotCleanEventSourceAndType() SubscriptionOpt {
	return WithSourceAndType(EventSourceUnclean, OrderCreatedUncleanEvent)
}

// WithTypeMatchingStandard is a SubscriptionOpt that initializes the subscription with type matching to standard.
func WithTypeMatchingStandard() SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) {
		subscription.Spec.TypeMatching = eventingv1alpha2.TypeMatchingStandard
	}
}

// WithTypeMatchingExact is a SubscriptionOpt that initializes the subscription with type matching to exact.
func WithTypeMatchingExact() SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) {
		subscription.Spec.TypeMatching = eventingv1alpha2.TypeMatchingExact
	}
}

// WithMaxInFlight is a SubscriptionOpt that sets the status with the maxInFlightMessages int value.
func WithMaxInFlight(maxInFlight int) SubscriptionOpt {
	return func(subscription *eventingv1alpha2.Subscription) {
		subscription.Spec.Config = map[string]string{
			eventingv1alpha2.MaxInFlightMessages: strconv.Itoa(maxInFlight),
		}
	}
}

// WithMaxInFlightMessages is a SubscriptionOpt that sets the status with the maxInFlightMessages string value.
func WithMaxInFlightMessages(maxInFlight string) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		if sub.Spec.Config == nil {
			sub.Spec.Config = map[string]string{}
		}
		sub.Spec.Config[eventingv1alpha2.MaxInFlightMessages] = maxInFlight
	}
}

// WithBackend is a SubscriptionOpt that sets the status with the Backend value.
func WithBackend(backend eventingv1alpha2.Backend) SubscriptionOpt {
	return func(sub *eventingv1alpha2.Subscription) {
		sub.Status.Backend = backend
	}
}
