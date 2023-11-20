package object

import (
	"k8s.io/apimachinery/pkg/util/intstr"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	apigatewayv1beta1 "github.com/kyma-incubator/api-gateway/api/v1beta1"
	eventingv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/env"

	"github.com/kyma-project/eventing-manager/pkg/utils"
)

func TestApiRuleEqual(t *testing.T) {
	svc := "svc"
	port := uint32(9999)
	host := "host"
	isExternal := true
	gateway := "foo.gateway"
	labels := map[string]string{
		"foo": "bar",
	}
	handler := &apigatewayv1beta1.Handler{
		Name: "handler",
	}
	rule := apigatewayv1beta1.Rule{
		Path: "path",
		Methods: []string{
			http.MethodPost,
		},
		AccessStrategies: []*apigatewayv1beta1.Authenticator{
			{
				Handler: handler,
			},
		},
	}
	apiRule := apigatewayv1beta1.APIRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
			Labels:    labels,
		},
		Spec: apigatewayv1beta1.APIRuleSpec{
			Service: &apigatewayv1beta1.Service{
				Name:       &svc,
				Port:       &port,
				IsExternal: &isExternal,
			},
			Host:    &host,
			Gateway: &gateway,
			Rules:   []apigatewayv1beta1.Rule{rule},
		},
	}
	testCases := map[string]struct {
		prep   func() *apigatewayv1beta1.APIRule
		expect bool
	}{
		"should be equal when svc, gateway, owner ref, rules are same": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				return apiRuleCopy
			},
			expect: true,
		},
		"should be unequal when svc name is diff": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				newSvcName := "new"
				apiRuleCopy.Spec.Service.Name = &newSvcName
				return apiRuleCopy
			},
			expect: false,
		},
		"should be unequal when svc port is diff": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				newSvcPort := uint32(8080)
				apiRuleCopy.Spec.Service.Port = &newSvcPort
				return apiRuleCopy
			},
			expect: false,
		},
		"should be unequal when isExternal is diff": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				newIsExternal := false
				apiRuleCopy.Spec.Service.IsExternal = &newIsExternal
				return apiRuleCopy
			},
			expect: false,
		},
		"should be unequal when gateway is diff": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				newGateway := "new-gw"
				apiRuleCopy.Spec.Gateway = &newGateway
				return apiRuleCopy
			},
			expect: false,
		},
		"should be unequal when labels are diff": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				newLabels := map[string]string{
					"new-foo": "new-bar",
				}
				apiRuleCopy.Labels = newLabels
				return apiRuleCopy
			},
			expect: false,
		},
		"should be unequal when path is diff": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				newRule := rule.DeepCopy()
				newRule.Path = "new-path"
				apiRuleCopy.Spec.Rules = []apigatewayv1beta1.Rule{*newRule}
				return apiRuleCopy
			},
			expect: false,
		},
		"should be unequal when methods are diff": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				newRule := rule.DeepCopy()
				newRule.Methods = []string{http.MethodOptions}
				apiRuleCopy.Spec.Rules = []apigatewayv1beta1.Rule{*newRule}
				return apiRuleCopy
			},
			expect: false,
		},
		"should be unequal when handlers are diff": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				newRule := rule.DeepCopy()
				newHandler := &apigatewayv1beta1.Handler{
					Name: "foo",
				}
				newRule.AccessStrategies = []*apigatewayv1beta1.Authenticator{
					{
						Handler: newHandler,
					},
				}
				apiRuleCopy.Spec.Rules = []apigatewayv1beta1.Rule{*newRule}
				return apiRuleCopy
			},
			expect: false,
		},
		"should be unequal when OwnerReferences are diff": {
			prep: func() *apigatewayv1beta1.APIRule {
				apiRuleCopy := apiRule.DeepCopy()
				newOwnerRef := metav1.OwnerReference{
					APIVersion: "foo",
					Kind:       "foo",
					Name:       "foo",
					UID:        "uid",
				}
				apiRuleCopy.OwnerReferences = []metav1.OwnerReference{
					newOwnerRef,
				}
				return apiRuleCopy
			},
			expect: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			testAPIRule := tc.prep()
			if apiRuleEqual(&apiRule, testAPIRule) != tc.expect {
				t.Errorf("expected output to be %t", tc.expect)
			}
		})
	}
}

func TestEventingBackendEqual(t *testing.T) {
	emptyBackend := eventingv1alpha1.EventingBackend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: eventingv1alpha1.EventingBackendSpec{},
	}

	testCases := map[string]struct {
		getBackend1    func() *eventingv1alpha1.EventingBackend
		getBackend2    func() *eventingv1alpha1.EventingBackend
		expectedResult bool
	}{
		"should be unequal if labels are different": {
			getBackend1: func() *eventingv1alpha1.EventingBackend {
				b := emptyBackend.DeepCopy()
				b.Labels = map[string]string{"k1": "v1"}
				return b
			},
			getBackend2: func() *eventingv1alpha1.EventingBackend {
				return emptyBackend.DeepCopy()
			},
			expectedResult: false,
		},
		"should be equal if labels are the same": {
			getBackend1: func() *eventingv1alpha1.EventingBackend {
				b := emptyBackend.DeepCopy()
				b.Labels = map[string]string{"k1": "v1"}
				return b
			},
			getBackend2: func() *eventingv1alpha1.EventingBackend {
				b := emptyBackend.DeepCopy()
				b.Name = "bar"
				b.Labels = map[string]string{"k1": "v1"}
				return b
			},
			expectedResult: true,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if eventingBackendEqual(tc.getBackend1(), tc.getBackend2()) != tc.expectedResult {
				t.Errorf("expected output to be %t", tc.expectedResult)
			}
		})
	}
}

func TestEventingBackendStatusEqual(t *testing.T) {
	testCases := []struct {
		name                string
		givenBackendStatus1 eventingv1alpha1.EventingBackendStatus
		givenBackendStatus2 eventingv1alpha1.EventingBackendStatus
		wantResult          bool
	}{
		{
			name: "should be unequal if ready status is different",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				EventingReady: utils.BoolPtr(false),
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{
				EventingReady: utils.BoolPtr(true),
			},
			wantResult: false,
		},
		{
			name: "should be unequal if missing secret",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				EventingReady:      utils.BoolPtr(false),
				BEBSecretName:      "secret",
				BEBSecretNamespace: "default",
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{
				EventingReady: utils.BoolPtr(false),
			},
			wantResult: false,
		},
		{
			name: "should be unequal if different secretName",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				EventingReady:      utils.BoolPtr(false),
				BEBSecretName:      "secret",
				BEBSecretNamespace: "default",
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{
				EventingReady:      utils.BoolPtr(false),
				BEBSecretName:      "secretnew",
				BEBSecretNamespace: "default",
			},
			wantResult: false,
		},
		{
			name: "should be unequal if different secretNamespace",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				EventingReady:      utils.BoolPtr(false),
				BEBSecretName:      "secret",
				BEBSecretNamespace: "default",
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{
				EventingReady:      utils.BoolPtr(false),
				BEBSecretName:      "secret",
				BEBSecretNamespace: "kyma-system",
			},
			wantResult: false,
		},
		{
			name: "should be unequal if missing backend",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				Backend: eventingv1alpha1.NatsBackendType,
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{},
			wantResult:          false,
		},
		{
			name: "should be unequal if different backend",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				Backend: eventingv1alpha1.NatsBackendType,
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{
				Backend: eventingv1alpha1.BEBBackendType,
			},
			wantResult: false,
		},
		{
			name: "should be unequal if conditions different",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				Conditions: []eventingv1alpha1.Condition{
					{Type: eventingv1alpha1.ConditionPublisherProxyReady, Status: corev1.ConditionTrue},
				},
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{
				Conditions: []eventingv1alpha1.Condition{
					{Type: eventingv1alpha1.ConditionPublisherProxyReady, Status: corev1.ConditionFalse},
				},
			},
			wantResult: false,
		},
		{
			name: "should be unequal if conditions missing",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				Conditions: []eventingv1alpha1.Condition{
					{Type: eventingv1alpha1.ConditionPublisherProxyReady, Status: corev1.ConditionTrue},
				},
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{
				Conditions: []eventingv1alpha1.Condition{},
			},
			wantResult: false,
		},
		{
			name: "should be unequal if conditions different",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				Conditions: []eventingv1alpha1.Condition{
					{Type: eventingv1alpha1.ConditionPublisherProxyReady, Status: corev1.ConditionTrue},
				},
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{
				Conditions: []eventingv1alpha1.Condition{
					{Type: eventingv1alpha1.ConditionControllerReady, Status: corev1.ConditionTrue},
				},
			},
			wantResult: false,
		},
		{
			name: "should be equal if the status are the same",
			givenBackendStatus1: eventingv1alpha1.EventingBackendStatus{
				Backend: eventingv1alpha1.NatsBackendType,
				Conditions: []eventingv1alpha1.Condition{
					{Type: eventingv1alpha1.ConditionControllerReady, Status: corev1.ConditionTrue},
					{Type: eventingv1alpha1.ConditionPublisherProxyReady, Status: corev1.ConditionTrue},
				},
				EventingReady: utils.BoolPtr(true),
			},
			givenBackendStatus2: eventingv1alpha1.EventingBackendStatus{
				Backend: eventingv1alpha1.NatsBackendType,
				Conditions: []eventingv1alpha1.Condition{
					{Type: eventingv1alpha1.ConditionControllerReady, Status: corev1.ConditionTrue},
					{Type: eventingv1alpha1.ConditionPublisherProxyReady, Status: corev1.ConditionTrue},
				},
				EventingReady: utils.BoolPtr(true),
			},
			wantResult: true,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if IsBackendStatusEqual(tc.givenBackendStatus1, tc.givenBackendStatus2) != tc.wantResult {
				t.Errorf("expected output to be %t", tc.wantResult)
			}
		})
	}
}

func Test_isSubscriptionStatusEqual(t *testing.T) {
	testCases := []struct {
		name                string
		subscriptionStatus1 eventingv1alpha2.SubscriptionStatus
		subscriptionStatus2 eventingv1alpha2.SubscriptionStatus
		wantEqualStatus     bool
	}{
		{
			name: "should not be equal if the conditions are not equal",
			subscriptionStatus1: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: corev1.ConditionTrue},
				},
				Ready: true,
			},
			subscriptionStatus2: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: corev1.ConditionFalse},
				},
				Ready: true,
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the ready status is not equal",
			subscriptionStatus1: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: corev1.ConditionTrue},
				},
				Ready: true,
			},
			subscriptionStatus2: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: corev1.ConditionTrue},
				},
				Ready: false,
			},
			wantEqualStatus: false,
		},
		{
			name: "should be equal if all the fields are equal",
			subscriptionStatus1: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: corev1.ConditionTrue},
				},
				Ready: true,
				Backend: eventingv1alpha2.Backend{
					APIRuleName: "APIRule",
				},
			},
			subscriptionStatus2: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: corev1.ConditionTrue},
				},
				Ready: true,
				Backend: eventingv1alpha2.Backend{
					APIRuleName: "APIRule",
				},
			},
			wantEqualStatus: true,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotEqualStatus := IsSubscriptionStatusEqual(tc.subscriptionStatus1, tc.subscriptionStatus2)
			require.Equal(t, tc.wantEqualStatus, gotEqualStatus)
		})
	}
}

func TestPublisherProxyDeploymentEqual(t *testing.T) {
	publisherCfg := env.PublisherConfig{
		Image:          "publisher",
		PortNum:        0,
		MetricsPortNum: 0,
		ServiceAccount: "publisher-sa",
		Replicas:       1,
		RequestsCPU:    "32m",
		RequestsMemory: "64Mi",
		LimitsCPU:      "64m",
		LimitsMemory:   "128Mi",
	}
	natsConfig := env.NATSConfig{
		EventTypePrefix: "prefix",
		JSStreamName:    "kyma",
	}
	defaultNATSPublisher := deployment.NewNATSPublisherDeployment(natsConfig, publisherCfg)
	defaultBEBPublisher := deployment.NewBEBPublisherDeployment(publisherCfg)

	testCases := map[string]struct {
		getPublisher1  func() *appsv1.Deployment
		getPublisher2  func() *appsv1.Deployment
		expectedResult bool
	}{
		"should be equal if same default NATS publisher": {
			getPublisher1: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Name = "publisher1"
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Name = "publisher2"
				return p
			},
			expectedResult: true,
		},
		"should be equal if same default BEB publisher": {
			getPublisher1: func() *appsv1.Deployment {
				p := defaultBEBPublisher.DeepCopy()
				p.Name = "publisher1"
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				p := defaultBEBPublisher.DeepCopy()
				p.Name = "publisher2"
				return p
			},
			expectedResult: true,
		},
		"should be unequal if publisher types are different": {
			getPublisher1: func() *appsv1.Deployment {
				return defaultBEBPublisher.DeepCopy()
			},
			getPublisher2: func() *appsv1.Deployment {
				return defaultNATSPublisher.DeepCopy()
			},
			expectedResult: false,
		},
		"should be unequal if publisher image changes": {
			getPublisher1: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Spec.Containers[0].Image = "new-publisher-img"
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				return defaultNATSPublisher.DeepCopy()
			},
			expectedResult: false,
		},
		"should be unequal if env var changes": {
			getPublisher1: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Spec.Containers[0].Env[0].Value = "new-value"
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				return defaultNATSPublisher.DeepCopy()
			},
			expectedResult: false,
		},
		"should be equal if replicas changes": {
			getPublisher1: func() *appsv1.Deployment {
				replicas := int32(1)
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Replicas = &replicas
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				replicas := int32(2)
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Replicas = &replicas
				return p
			},
			expectedResult: true,
		},
		"should be equal if replicas are the same": {
			getPublisher1: func() *appsv1.Deployment {
				replicas := int32(2)
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Replicas = &replicas
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				replicas := int32(2)
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Replicas = &replicas
				return p
			},
			expectedResult: true,
		},
		"should be equal if spec annotations are nil and empty": {
			getPublisher1: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Annotations = nil
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Annotations = map[string]string{}
				return p
			},
			expectedResult: true,
		},
		"should be unequal if spec annotations changes": {
			getPublisher1: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Annotations = map[string]string{"key": "value1"}
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Annotations = map[string]string{"key": "value2"}
				return p
			},
			expectedResult: false,
		},
		"should be equal if spec Labels are nil and empty": {
			getPublisher1: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Labels = nil
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Labels = map[string]string{}
				return p
			},
			expectedResult: true,
		},
		"should be unequal if spec Labels changes": {
			getPublisher1: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Labels = map[string]string{"key": "value1"}
				return p
			},
			getPublisher2: func() *appsv1.Deployment {
				p := defaultNATSPublisher.DeepCopy()
				p.Spec.Template.Labels = map[string]string{"key": "value2"}
				return p
			},
			expectedResult: false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if publisherProxyDeploymentEqual(tc.getPublisher1(), tc.getPublisher2()) != tc.expectedResult {
				t.Errorf("expected output to be %t", tc.expectedResult)
			}
		})
	}
}

func Test_ownerReferencesDeepEqual(t *testing.T) {
	ownerReference := func(version, kind, name, uid string, controller, block *bool) metav1.OwnerReference {
		return metav1.OwnerReference{
			APIVersion:         version,
			Kind:               kind,
			Name:               name,
			UID:                types.UID(uid),
			Controller:         controller,
			BlockOwnerDeletion: block,
		}
	}

	tests := []struct {
		name                  string
		givenOwnerReferences1 []metav1.OwnerReference
		givenOwnerReferences2 []metav1.OwnerReference
		wantEqual             bool
	}{
		{
			name:                  "both OwnerReferences are nil",
			givenOwnerReferences1: nil,
			givenOwnerReferences2: nil,
			wantEqual:             true,
		},
		{
			name:                  "both OwnerReferences are empty",
			givenOwnerReferences1: []metav1.OwnerReference{},
			givenOwnerReferences2: []metav1.OwnerReference{},
			wantEqual:             true,
		},
		{
			name: "same OwnerReferences and same order",
			givenOwnerReferences1: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
				ownerReference("v-1", "k-1", "n-1", "u-1", ptr.To(false), ptr.To(false)),
				ownerReference("v-2", "k-2", "n-2", "u-2", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
				ownerReference("v-1", "k-1", "n-1", "u-1", ptr.To(false), ptr.To(false)),
				ownerReference("v-2", "k-2", "n-2", "u-2", ptr.To(false), ptr.To(false)),
			},
			wantEqual: true,
		},
		{
			name: "same OwnerReferences but different order",
			givenOwnerReferences1: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
				ownerReference("v-1", "k-1", "n-1", "u-1", ptr.To(false), ptr.To(false)),
				ownerReference("v-2", "k-2", "n-2", "u-2", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []metav1.OwnerReference{
				ownerReference("v-2", "k-2", "n-2", "u-2", ptr.To(false), ptr.To(false)),
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
				ownerReference("v-1", "k-1", "n-1", "u-1", ptr.To(false), ptr.To(false)),
			},
			wantEqual: true,
		},
		{
			name: "different OwnerReference APIVersion",
			givenOwnerReferences1: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []metav1.OwnerReference{
				ownerReference("v-1", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference Kind",
			givenOwnerReferences1: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []metav1.OwnerReference{
				ownerReference("v-0", "k-1", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference Name",
			givenOwnerReferences1: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-1", "u-0", ptr.To(false), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference UID",
			givenOwnerReferences1: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-1", ptr.To(false), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference Controller",
			givenOwnerReferences1: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(true), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference BlockOwnerDeletion",
			givenOwnerReferences1: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []metav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(true)),
			},
			wantEqual: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantEqual, ownerReferencesDeepEqual(tc.givenOwnerReferences1, tc.givenOwnerReferences2))
		})
	}
}

func Test_containerEqual(t *testing.T) {
	quantityA, _ := resource.ParseQuantity("5m")
	quantityB, _ := resource.ParseQuantity("10k")

	type args struct {
		c1 *corev1.Container
		c2 *corev1.Container
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "container are equal",
			args: args{
				c1: &corev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []corev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler:        corev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
				c2: &corev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []corev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler:        corev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
			},
			want: true,
		},
		{
			name: "ContainerPort are not equal",
			args: args{
				c1: &corev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []corev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler:        corev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
				c2: &corev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []corev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 3,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler:        corev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
			},
			want: false,
		},
		{
			name: "resources are not equal",
			args: args{
				c1: &corev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []corev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler:        corev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
				c2: &corev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []corev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu": quantityB,
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"mem": quantityB,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler:        corev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
			},
			want: false,
		},
		{
			name: "ports are not equal",
			args: args{
				c1: &corev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []corev1.ContainerPort{
						{
							Name:          "testport-0",
							HostPort:      1,
							ContainerPort: 2,
							Protocol:      "http",
							HostIP:        "192.168.1.1",
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler:        corev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
				c2: &corev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []corev1.ContainerPort{
						{
							Name:          "testport-0",
							HostPort:      1,
							ContainerPort: 2,
							Protocol:      "http",
							HostIP:        "192.168.1.1",
						},
						{
							Name:          "testport-1",
							HostPort:      1,
							ContainerPort: 2,
							Protocol:      "http",
							HostIP:        "192.168.1.1",
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[corev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler:        corev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containerEqual(tt.args.c1, tt.args.c2); got != tt.want {
				t.Errorf("containerEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_serviceAccountEqual(t *testing.T) {
	const (
		name0 = "name-0"
		name1 = "name-1"

		namespace0 = "ns-0"
		namespace1 = "ns-1"
	)

	var (
		labels0 = map[string]string{"key": "val-0"}
		labels1 = map[string]string{"key": "val-1"}

		ownerReferences0 = []metav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []metav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		serviceAccount = &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name0,
				Namespace:       namespace0,
				Labels:          labels0,
				OwnerReferences: ownerReferences0,
			},
		}
	)

	type args struct {
		a *corev1.ServiceAccount
		b *corev1.ServiceAccount
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ServiceAccount refs are equal",
			args: args{
				a: serviceAccount,
				b: serviceAccount,
			},
			want: true,
		},
		{
			name: "one ServiceAccount is Nil",
			args: args{
				a: nil,
				b: serviceAccount,
			},
			want: false,
		},
		{
			name: "both ServiceAccounts are Nil",
			args: args{
				a: nil,
				b: nil,
			},
			want: true,
		},
		{
			name: "ServiceAccounts are equal",
			args: args{
				a: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
			},
			want: true,
		},
		{
			name: "ServiceAccount names are not equal",
			args: args{
				a: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name1,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
			},
			want: false,
		},
		{
			name: "ServiceAccount namespaces are not equal",
			args: args{
				a: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace1,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
			},
			want: false,
		},
		{
			name: "ServiceAccount labels are not equal",
			args: args{
				a: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels1,
						OwnerReferences: ownerReferences0,
					},
				},
			},
			want: false,
		},
		{
			name: "ServiceAccount OwnerReferences are not equal",
			args: args{
				a: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences1,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serviceAccountEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("serviceAccountEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clusterRoleEqual(t *testing.T) {
	const (
		name0 = "name-0"
		name1 = "name-1"
	)

	var (
		labels0 = map[string]string{"key": "val-0"}
		labels1 = map[string]string{"key": "val-1"}

		ownerReferences0 = []metav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []metav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		rules0 = []rbacv1.PolicyRule{
			{
				Verbs:           []string{"val-0"},
				APIGroups:       []string{"val-0"},
				Resources:       []string{"val-0"},
				ResourceNames:   []string{"val-0"},
				NonResourceURLs: []string{"val-0"},
			},
		}
		rules1 = []rbacv1.PolicyRule{
			{
				Verbs:           []string{"val-1"},
				APIGroups:       []string{"val-0"},
				Resources:       []string{"val-0"},
				ResourceNames:   []string{"val-0"},
				NonResourceURLs: []string{"val-0"},
			},
		}

		clusterRole = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name0,
				Labels:          labels0,
				OwnerReferences: ownerReferences0,
			},
			Rules: rules0,
		}
	)

	type args struct {
		a *rbacv1.ClusterRole
		b *rbacv1.ClusterRole
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ClusterRole refs are equal",
			args: args{
				a: clusterRole,
				b: clusterRole,
			},
			want: true,
		},
		{
			name: "one ClusterRole is nil",
			args: args{
				a: nil,
				b: clusterRole,
			},
			want: false,
		},
		{
			name: "both ClusterRoles are nil",
			args: args{
				a: nil,
				b: nil,
			},
			want: true,
		},
		{
			name: "ClusterRoles are equal",
			args: args{
				a: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
			},
			want: true,
		},
		{
			name: "ClusterRole names are not equal",
			args: args{
				a: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name1,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
			},
			want: false,
		},
		{
			name: "ClusterRole labels are not equal",
			args: args{
				a: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Labels:          labels1,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
			},
			want: false,
		},
		{
			name: "ClusterRole OwnerReferences are not equal",
			args: args{
				a: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences1,
					},
					Rules: rules0,
				},
			},
			want: false,
		},
		{
			name: "ClusterRole Rules are not equal",
			args: args{
				a: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules1,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clusterRoleEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("clusterRoleEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clusterRoleBindingEqual(t *testing.T) {
	const (
		name0 = "name-0"
		name1 = "name-1"
	)

	var (
		ownerReferences0 = []metav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []metav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		subjects0 = []rbacv1.Subject{
			{
				Kind:      "kind-0",
				APIGroup:  "group-0",
				Name:      "name-0",
				Namespace: "namespace-0",
			},
		}
		subjects1 = []rbacv1.Subject{
			{
				Kind:      "kind-1",
				APIGroup:  "group-0",
				Name:      "name-0",
				Namespace: "namespace-0",
			},
		}

		roleRef0 = rbacv1.RoleRef{
			APIGroup: "group-0",
			Kind:     "kind-0",
			Name:     "name-0",
		}
		roleRef1 = rbacv1.RoleRef{
			APIGroup: "group-1",
			Kind:     "kind-0",
			Name:     "name-0",
		}
	)

	type args struct {
		a *rbacv1.ClusterRoleBinding
		b *rbacv1.ClusterRoleBinding
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ClusterRoleBindings are equal",
			args: args{
				a: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
			},
			want: true,
		},
		{
			name: "ClusterRoleBinding names are not equal",
			args: args{
				a: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name1,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
			},
			want: false,
		},
		{
			name: "ClusterRoleBinding OwnerReferences are not equal",
			args: args{
				a: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences1,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
			},
			want: false,
		},
		{
			name: "ClusterRoleBinding Subjects are not equal",
			args: args{
				a: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects1,
					RoleRef:  roleRef0,
				},
			},
			want: false,
		},
		{
			name: "ClusterRoleBinding RoleRefs are not equal",
			args: args{
				a: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef1,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clusterRoleBindingEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("clusterRoleBindingEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_serviceEqual(t *testing.T) {
	const (
		name0 = "name-0"
		name1 = "name-1"

		namespace0 = "namespace0"
		namespace1 = "namespace1"
	)

	var (
		ownerReferences0 = []metav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []metav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		ports0 = []corev1.ServicePort{
			{
				Name:        "name-0",
				Protocol:    "protocol-0",
				AppProtocol: nil,
				Port:        0,
				TargetPort: intstr.IntOrString{
					Type:   0,
					IntVal: 0,
					StrVal: "val-0",
				},
				NodePort: 0,
			},
		}
		ports1 = []corev1.ServicePort{
			{
				Name:        "name-1",
				Protocol:    "protocol-0",
				AppProtocol: nil,
				Port:        0,
				TargetPort: intstr.IntOrString{
					Type:   0,
					IntVal: 0,
					StrVal: "val-0",
				},
				NodePort: 0,
			},
		}

		selector0 = map[string]string{
			"key": "val-0",
		}
		selector1 = map[string]string{
			"key": "val-1",
		}
	)

	type args struct {
		a *corev1.Service
		b *corev1.Service
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Services are equal",
			args: args{
				a: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
			},
			want: true,
		},
		{
			name: "Service names are not equal",
			args: args{
				a: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name1,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
			},
			want: false,
		},
		{
			name: "Service namespaces are not equal",
			args: args{
				a: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace1,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
			},
			want: false,
		},
		{
			name: "Service OwnerReferences are not equal",
			args: args{
				a: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences1,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
			},
			want: false,
		},
		{
			name: "Service ports are not equal",
			args: args{
				a: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports1,
						Selector: selector0,
					},
				},
			},
			want: false,
		},
		{
			name: "Service selectors are not equal",
			args: args{
				a: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: corev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector1,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serviceEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("serviceEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hpaEqual(t *testing.T) {
	const (
		name0 = "name-0"
		name1 = "name-1"

		namespace0 = "ns-0"
		namespace1 = "ns-1"
	)

	var (
		ownerReferences0 = []metav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []metav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		scaleTargetRef0 = v2.CrossVersionObjectReference{
			Name:       "name-0",
			Kind:       "kind-0",
			APIVersion: "version-0",
		}
		scaleTargetRef1 = v2.CrossVersionObjectReference{
			Name:       "name-1",
			Kind:       "kind-0",
			APIVersion: "version-0",
		}

		minReplicas0 = ptr.To(int32(1))
		minReplicas1 = ptr.To(int32(2))

		maxReplicas0 = int32(3)
		maxReplicas1 = int32(4)

		metrics0 = []v2.MetricSpec{
			{
				Type:              v2.MetricSourceType("type-0"),
				Object:            nil,
				Pods:              nil,
				Resource:          nil,
				ContainerResource: nil,
				External:          nil,
			},
		}
		metrics1 = []v2.MetricSpec{
			{
				Type:              v2.MetricSourceType("type-1"),
				Object:            nil,
				Pods:              nil,
				Resource:          nil,
				ContainerResource: nil,
				External:          nil,
			},
		}
	)

	type args struct {
		a *v2.HorizontalPodAutoscaler
		b *v2.HorizontalPodAutoscaler
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "HPAs are equal",
			args: args{
				a: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
			},
			want: true,
		},
		{
			name: "HPA names are not equal",
			args: args{
				a: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name1,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
			},
			want: false,
		},
		{
			name: "HPA namespaces are not equal",
			args: args{
				a: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace1,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
			},
			want: false,
		},
		{
			name: "HPA OwnerReferences are not equal",
			args: args{
				a: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences1,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
			},
			want: false,
		},
		{
			name: "HPA ScaleTargetRefs are not equal",
			args: args{
				a: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef1,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
			},
			want: false,
		},
		{
			name: "HPA MinReplicas are not equal",
			args: args{
				a: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas1,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
			},
			want: false,
		},
		{
			name: "HPA MaxReplicas are not equal",
			args: args{
				a: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas1,
						Metrics:        metrics0,
					},
				},
			},
			want: false,
		},
		{
			name: "HPA metrics are not equal",
			args: args{
				a: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &v2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: v2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics1,
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hpaEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("hpaEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_envEqual(t *testing.T) {
	type args struct {
		e1 []corev1.EnvVar
		e2 []corev1.EnvVar
	}

	var11 := corev1.EnvVar{
		Name:  "var1",
		Value: "var1",
	}
	var12 := corev1.EnvVar{
		Name:  "var1",
		Value: "var2",
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "envs equal, order equals",
			args: args{
				e1: []corev1.EnvVar{var11, var12},
				e2: []corev1.EnvVar{var11, var12},
			},
			want: true,
		},
		{
			name: "envs equal, different order",
			args: args{
				e1: []corev1.EnvVar{var11, var12},
				e2: []corev1.EnvVar{var12, var11},
			},
			want: true,
		},
		{
			name: "different length",
			args: args{
				e1: []corev1.EnvVar{var11, var11},
				e2: []corev1.EnvVar{var11},
			},
			want: false,
		},
		{
			name: "envs different",
			args: args{
				e1: []corev1.EnvVar{var11, var12},
				e2: []corev1.EnvVar{var11, var11},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := envEqual(tt.args.e1, tt.args.e2); got != tt.want {
				t.Errorf("envEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_probeEqual(t *testing.T) {
	var (
		probe = &corev1.Probe{}
	)

	type args struct {
		p1 *corev1.Probe
		p2 *corev1.Probe
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Probe refs are equal",
			args: args{
				p1: probe,
				p2: probe,
			},
			want: true,
		},
		{
			name: "one Probe is Nil",
			args: args{
				p1: nil,
				p2: probe,
			},
			want: false,
		},
		{
			name: "both Probes are Nil",
			args: args{
				p1: nil,
				p2: nil,
			},
			want: true,
		},
		{
			name: "Probes are not equal",
			args: args{
				p1: &corev1.Probe{
					InitialDelaySeconds: 1,
				},
				p2: &corev1.Probe{
					InitialDelaySeconds: 2,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := probeEqual(tt.args.p1, tt.args.p2); got != tt.want {
				t.Errorf("probeEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
