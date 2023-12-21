//nolint:dupl // these comparison functions all look very similar
package object

import (
	"net/http"
	"testing"

	apigatewayv1beta1 "github.com/kyma-project/api-gateway/apis/gateway/v1beta1"
	"github.com/stretchr/testify/require"
	kautoscalingv2 "k8s.io/api/autoscaling/v2"
	kcorev1 "k8s.io/api/core/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
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
		ObjectMeta: kmetav1.ObjectMeta{
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
				newOwnerRef := kmetav1.OwnerReference{
					APIVersion: "foo",
					Kind:       "foo",
					Name:       "foo",
					UID:        "uid",
				}
				apiRuleCopy.OwnerReferences = []kmetav1.OwnerReference{
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

func Test_isSubscriptionStatusEqual(t *testing.T) {
	t.Parallel()
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
					{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				},
				Ready: true,
			},
			subscriptionStatus2: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionFalse},
				},
				Ready: true,
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the ready status is not equal",
			subscriptionStatus1: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				},
				Ready: true,
			},
			subscriptionStatus2: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				},
				Ready: false,
			},
			wantEqualStatus: false,
		},
		{
			name: "should be equal if all the fields are equal",
			subscriptionStatus1: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
				},
				Ready: true,
				Backend: eventingv1alpha2.Backend{
					APIRuleName: "APIRule",
				},
			},
			subscriptionStatus2: eventingv1alpha2.SubscriptionStatus{
				Conditions: []eventingv1alpha2.Condition{
					{Type: eventingv1alpha2.ConditionSubscribed, Status: kcorev1.ConditionTrue},
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
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			gotEqualStatus := IsSubscriptionStatusEqual(testcase.subscriptionStatus1, testcase.subscriptionStatus2)
			require.Equal(t, testcase.wantEqualStatus, gotEqualStatus)
		})
	}
}

// func TestPublisherProxyDeploymentEqual(t *testing.T) {
//	publisherCfg := env.PublisherConfig{
//		Image:          "publisher",
//		PortNum:        0,
//		MetricsPortNum: 0,
//		ServiceAccount: "publisher-sa",
//		Replicas:       1,
//		RequestsCPU:    "32m",
//		RequestsMemory: "64Mi",
//		LimitsCPU:      "64m",
//		LimitsMemory:   "128Mi",
//	}
//	natsConfig := env.NATSConfig{
//		EventTypePrefix: "prefix",
//		JSStreamName:    "kyma",
//	}
//	defaultNATSPublisher := eventingpkg.NewNATSPublisherDeployment(natsConfig, publisherCfg)
//	defaultBEBPublisher := eventingpkg.NewBEBPublisherDeployment(publisherCfg)
//
//	testCases := map[string]struct {
//		getPublisher1  func() *appsv1.Deployment
//		getPublisher2  func() *appsv1.Deployment
//		expectedResult bool
//	}{
//		"should be equal if same default NATS publisher": {
//			getPublisher1: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Name = "publisher1"
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Name = "publisher2"
//				return p
//			},
//			expectedResult: true,
//		},
//		"should be equal if same default BEB publisher": {
//			getPublisher1: func() *appsv1.Deployment {
//				p := defaultBEBPublisher.DeepCopy()
//				p.Name = "publisher1"
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				p := defaultBEBPublisher.DeepCopy()
//				p.Name = "publisher2"
//				return p
//			},
//			expectedResult: true,
//		},
//		"should be unequal if publisher types are different": {
//			getPublisher1: func() *appsv1.Deployment {
//				return defaultBEBPublisher.DeepCopy()
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				return defaultNATSPublisher.DeepCopy()
//			},
//			expectedResult: false,
//		},
//		"should be unequal if publisher image changes": {
//			getPublisher1: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Spec.Containers[0].Image = "new-publisher-img"
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				return defaultNATSPublisher.DeepCopy()
//			},
//			expectedResult: false,
//		},
//		"should be unequal if env var changes": {
//			getPublisher1: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Spec.Containers[0].Env[0].Value = "new-value"
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				return defaultNATSPublisher.DeepCopy()
//			},
//			expectedResult: false,
//		},
//		"should be equal if replicas changes": {
//			getPublisher1: func() *appsv1.Deployment {
//				replicas := int32(1)
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Replicas = &replicas
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				replicas := int32(2)
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Replicas = &replicas
//				return p
//			},
//			expectedResult: true,
//		},
//		"should be equal if replicas are the same": {
//			getPublisher1: func() *appsv1.Deployment {
//				replicas := int32(2)
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Replicas = &replicas
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				replicas := int32(2)
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Replicas = &replicas
//				return p
//			},
//			expectedResult: true,
//		},
//		"should be equal if spec annotations are nil and empty": {
//			getPublisher1: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Annotations = nil
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Annotations = map[string]string{}
//				return p
//			},
//			expectedResult: true,
//		},
//		"should be unequal if spec annotations changes": {
//			getPublisher1: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Annotations = map[string]string{"key": "value1"}
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Annotations = map[string]string{"key": "value2"}
//				return p
//			},
//			expectedResult: false,
//		},
//		"should be equal if spec Labels are nil and empty": {
//			getPublisher1: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Labels = nil
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Labels = map[string]string{}
//				return p
//			},
//			expectedResult: true,
//		},
//		"should be unequal if spec Labels changes": {
//			getPublisher1: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Labels = map[string]string{"key": "value1"}
//				return p
//			},
//			getPublisher2: func() *appsv1.Deployment {
//				p := defaultNATSPublisher.DeepCopy()
//				p.Spec.Template.Labels = map[string]string{"key": "value2"}
//				return p
//			},
//			expectedResult: false,
//		},
//	}
//	for name, tc := range testCases {
//		t.Run(name, func(t *testing.T) {
//			if publisherProxyDeploymentEqual(tc.getPublisher1(), tc.getPublisher2()) != tc.expectedResult {
//				t.Errorf("expected output to be %t", tc.expectedResult)
//			}
//		})
//	}
//}

func Test_ownerReferencesDeepEqual(t *testing.T) {
	ownerReference := func(version, kind, name, uid string, controller, block *bool) kmetav1.OwnerReference {
		return kmetav1.OwnerReference{
			APIVersion:         version,
			Kind:               kind,
			Name:               name,
			UID:                types.UID(uid),
			Controller:         controller,
			BlockOwnerDeletion: block,
		}
	}

	testCases := []struct {
		name                  string
		givenOwnerReferences1 []kmetav1.OwnerReference
		givenOwnerReferences2 []kmetav1.OwnerReference
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
			givenOwnerReferences1: []kmetav1.OwnerReference{},
			givenOwnerReferences2: []kmetav1.OwnerReference{},
			wantEqual:             true,
		},
		{
			name: "same OwnerReferences and same order",
			givenOwnerReferences1: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
				ownerReference("v-1", "k-1", "n-1", "u-1", ptr.To(false), ptr.To(false)),
				ownerReference("v-2", "k-2", "n-2", "u-2", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
				ownerReference("v-1", "k-1", "n-1", "u-1", ptr.To(false), ptr.To(false)),
				ownerReference("v-2", "k-2", "n-2", "u-2", ptr.To(false), ptr.To(false)),
			},
			wantEqual: true,
		},
		{
			name: "same OwnerReferences but different order",
			givenOwnerReferences1: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
				ownerReference("v-1", "k-1", "n-1", "u-1", ptr.To(false), ptr.To(false)),
				ownerReference("v-2", "k-2", "n-2", "u-2", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []kmetav1.OwnerReference{
				ownerReference("v-2", "k-2", "n-2", "u-2", ptr.To(false), ptr.To(false)),
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
				ownerReference("v-1", "k-1", "n-1", "u-1", ptr.To(false), ptr.To(false)),
			},
			wantEqual: true,
		},
		{
			name: "different OwnerReference APIVersion",
			givenOwnerReferences1: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []kmetav1.OwnerReference{
				ownerReference("v-1", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference Kind",
			givenOwnerReferences1: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-1", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference Name",
			givenOwnerReferences1: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-1", "u-0", ptr.To(false), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference UID",
			givenOwnerReferences1: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-1", ptr.To(false), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference Controller",
			givenOwnerReferences1: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(true), ptr.To(false)),
			},
			wantEqual: false,
		},
		{
			name: "different OwnerReference BlockOwnerDeletion",
			givenOwnerReferences1: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(false)),
			},
			givenOwnerReferences2: []kmetav1.OwnerReference{
				ownerReference("v-0", "k-0", "n-0", "u-0", ptr.To(false), ptr.To(true)),
			},
			wantEqual: false,
		},
	}

	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			require.Equal(t, testcase.wantEqual, ownerReferencesDeepEqual(testcase.givenOwnerReferences1, testcase.givenOwnerReferences2))
		})
	}
}

func Test_containerEqual(t *testing.T) {
	quantityA, _ := resource.ParseQuantity("5m")
	quantityB, _ := resource.ParseQuantity("10k")

	type args struct {
		c1 *kcorev1.Container
		c2 *kcorev1.Container
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "container are equal",
			args: args{
				c1: &kcorev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []kcorev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: kcorev1.ResourceRequirements{
						Limits: map[kcorev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[kcorev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &kcorev1.Probe{
						ProbeHandler:        kcorev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
				c2: &kcorev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []kcorev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: kcorev1.ResourceRequirements{
						Limits: map[kcorev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[kcorev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &kcorev1.Probe{
						ProbeHandler:        kcorev1.ProbeHandler{},
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
				c1: &kcorev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []kcorev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: kcorev1.ResourceRequirements{
						Limits: map[kcorev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[kcorev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &kcorev1.Probe{
						ProbeHandler:        kcorev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
				c2: &kcorev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []kcorev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 3,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: kcorev1.ResourceRequirements{
						Limits: map[kcorev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[kcorev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &kcorev1.Probe{
						ProbeHandler:        kcorev1.ProbeHandler{},
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
				c1: &kcorev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []kcorev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: kcorev1.ResourceRequirements{
						Limits: map[kcorev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[kcorev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &kcorev1.Probe{
						ProbeHandler:        kcorev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
				c2: &kcorev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []kcorev1.ContainerPort{{
						Name:          "testport",
						HostPort:      1,
						ContainerPort: 2,
						Protocol:      "http",
						HostIP:        "192.168.1.1",
					}},
					Resources: kcorev1.ResourceRequirements{
						Limits: map[kcorev1.ResourceName]resource.Quantity{
							"cpu": quantityB,
						},
						Requests: map[kcorev1.ResourceName]resource.Quantity{
							"mem": quantityB,
						},
					},
					ReadinessProbe: &kcorev1.Probe{
						ProbeHandler:        kcorev1.ProbeHandler{},
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
				c1: &kcorev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []kcorev1.ContainerPort{
						{
							Name:          "testport-0",
							HostPort:      1,
							ContainerPort: 2,
							Protocol:      "http",
							HostIP:        "192.168.1.1",
						},
					},
					Resources: kcorev1.ResourceRequirements{
						Limits: map[kcorev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[kcorev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &kcorev1.Probe{
						ProbeHandler:        kcorev1.ProbeHandler{},
						InitialDelaySeconds: 0,
						TimeoutSeconds:      0,
						PeriodSeconds:       0,
						SuccessThreshold:    0,
						FailureThreshold:    0,
					},
				},
				c2: &kcorev1.Container{
					Name:       "test",
					Image:      "bla",
					Command:    []string{"1", "2"},
					Args:       []string{"a", "b"},
					WorkingDir: "foodir",
					Ports: []kcorev1.ContainerPort{
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
					Resources: kcorev1.ResourceRequirements{
						Limits: map[kcorev1.ResourceName]resource.Quantity{
							"cpu": quantityA,
						},
						Requests: map[kcorev1.ResourceName]resource.Quantity{
							"mem": quantityA,
						},
					},
					ReadinessProbe: &kcorev1.Probe{
						ProbeHandler:        kcorev1.ProbeHandler{},
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

		ownerReferences0 = []kmetav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []kmetav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		serviceAccount = &kcorev1.ServiceAccount{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:            name0,
				Namespace:       namespace0,
				Labels:          labels0,
				OwnerReferences: ownerReferences0,
			},
		}
	)

	type args struct {
		a *kcorev1.ServiceAccount
		b *kcorev1.ServiceAccount
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
				a: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
				},
				b: &kcorev1.ServiceAccount{
					ObjectMeta: kmetav1.ObjectMeta{
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

		ownerReferences0 = []kmetav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []kmetav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		rules0 = []krbacv1.PolicyRule{
			{
				Verbs:           []string{"val-0"},
				APIGroups:       []string{"val-0"},
				Resources:       []string{"val-0"},
				ResourceNames:   []string{"val-0"},
				NonResourceURLs: []string{"val-0"},
			},
		}
		rules1 = []krbacv1.PolicyRule{
			{
				Verbs:           []string{"val-1"},
				APIGroups:       []string{"val-0"},
				Resources:       []string{"val-0"},
				ResourceNames:   []string{"val-0"},
				NonResourceURLs: []string{"val-0"},
			},
		}

		clusterRole = &krbacv1.ClusterRole{
			ObjectMeta: kmetav1.ObjectMeta{
				Name:            name0,
				Labels:          labels0,
				OwnerReferences: ownerReferences0,
			},
			Rules: rules0,
		}
	)

	type args struct {
		a *krbacv1.ClusterRole
		b *krbacv1.ClusterRole
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
				a: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Labels:          labels0,
						OwnerReferences: ownerReferences0,
					},
					Rules: rules0,
				},
				b: &krbacv1.ClusterRole{
					ObjectMeta: kmetav1.ObjectMeta{
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
		ownerReferences0 = []kmetav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []kmetav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		subjects0 = []krbacv1.Subject{
			{
				Kind:      "kind-0",
				APIGroup:  "group-0",
				Name:      "name-0",
				Namespace: "namespace-0",
			},
		}
		subjects1 = []krbacv1.Subject{
			{
				Kind:      "kind-1",
				APIGroup:  "group-0",
				Name:      "name-0",
				Namespace: "namespace-0",
			},
		}

		roleRef0 = krbacv1.RoleRef{
			APIGroup: "group-0",
			Kind:     "kind-0",
			Name:     "name-0",
		}
		roleRef1 = krbacv1.RoleRef{
			APIGroup: "group-1",
			Kind:     "kind-0",
			Name:     "name-0",
		}
	)

	type args struct {
		a *krbacv1.ClusterRoleBinding
		b *krbacv1.ClusterRoleBinding
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ClusterRoleBindings are equal",
			args: args{
				a: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
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
				a: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						OwnerReferences: ownerReferences0,
					},
					Subjects: subjects0,
					RoleRef:  roleRef0,
				},
				b: &krbacv1.ClusterRoleBinding{
					ObjectMeta: kmetav1.ObjectMeta{
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
		ownerReferences0 = []kmetav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []kmetav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		ports0 = []kcorev1.ServicePort{
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
		ports1 = []kcorev1.ServicePort{
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
		a *kcorev1.Service
		b *kcorev1.Service
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Services are equal",
			args: args{
				a: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
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
				a: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name1,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
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
				a: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace1,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
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
				a: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences1,
					},
					Spec: kcorev1.ServiceSpec{
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
				a: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
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
				a: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
						Ports:    ports0,
						Selector: selector0,
					},
				},
				b: &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kcorev1.ServiceSpec{
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
		ownerReferences0 = []kmetav1.OwnerReference{
			{
				Name:               "name-0",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}
		ownerReferences1 = []kmetav1.OwnerReference{
			{
				Name:               "name-1",
				APIVersion:         "version-0",
				Kind:               "kind-0",
				UID:                "000000",
				Controller:         ptr.To(false),
				BlockOwnerDeletion: ptr.To(false),
			},
		}

		scaleTargetRef0 = kautoscalingv2.CrossVersionObjectReference{
			Name:       "name-0",
			Kind:       "kind-0",
			APIVersion: "version-0",
		}
		scaleTargetRef1 = kautoscalingv2.CrossVersionObjectReference{
			Name:       "name-1",
			Kind:       "kind-0",
			APIVersion: "version-0",
		}

		minReplicas0 = ptr.To(int32(1))
		minReplicas1 = ptr.To(int32(2))

		maxReplicas0 = int32(3)
		maxReplicas1 = int32(4)

		metrics0 = []kautoscalingv2.MetricSpec{
			{
				Type:              kautoscalingv2.MetricSourceType("type-0"),
				Object:            nil,
				Pods:              nil,
				Resource:          nil,
				ContainerResource: nil,
				External:          nil,
			},
		}
		metrics1 = []kautoscalingv2.MetricSpec{
			{
				Type:              kautoscalingv2.MetricSourceType("type-1"),
				Object:            nil,
				Pods:              nil,
				Resource:          nil,
				ContainerResource: nil,
				External:          nil,
			},
		}
	)

	type args struct {
		a *kautoscalingv2.HorizontalPodAutoscaler
		b *kautoscalingv2.HorizontalPodAutoscaler
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "HPAs are equal",
			args: args{
				a: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
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
				a: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name1,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
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
				a: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace1,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
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
				a: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences1,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
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
				a: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
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
				a: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
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
				a: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
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
				a: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: scaleTargetRef0,
						MinReplicas:    minReplicas0,
						MaxReplicas:    maxReplicas0,
						Metrics:        metrics0,
					},
				},
				b: &kautoscalingv2.HorizontalPodAutoscaler{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name0,
						Namespace:       namespace0,
						OwnerReferences: ownerReferences0,
					},
					Spec: kautoscalingv2.HorizontalPodAutoscalerSpec{
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
		e1 []kcorev1.EnvVar
		e2 []kcorev1.EnvVar
	}

	var11 := kcorev1.EnvVar{
		Name:  "var1",
		Value: "var1",
	}
	var12 := kcorev1.EnvVar{
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
				e1: []kcorev1.EnvVar{var11, var12},
				e2: []kcorev1.EnvVar{var11, var12},
			},
			want: true,
		},
		{
			name: "envs equal, different order",
			args: args{
				e1: []kcorev1.EnvVar{var11, var12},
				e2: []kcorev1.EnvVar{var12, var11},
			},
			want: true,
		},
		{
			name: "different length",
			args: args{
				e1: []kcorev1.EnvVar{var11, var11},
				e2: []kcorev1.EnvVar{var11},
			},
			want: false,
		},
		{
			name: "envs different",
			args: args{
				e1: []kcorev1.EnvVar{var11, var12},
				e2: []kcorev1.EnvVar{var11, var11},
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
	probe := &kcorev1.Probe{}

	type args struct {
		p1 *kcorev1.Probe
		p2 *kcorev1.Probe
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
				p1: &kcorev1.Probe{
					InitialDelaySeconds: 1,
				},
				p2: &kcorev1.Probe{
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
