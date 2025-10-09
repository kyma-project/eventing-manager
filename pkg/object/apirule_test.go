package object

import (
	"net/http"
	"reflect"
	"testing"

	apigatewayv2 "github.com/kyma-project/api-gateway/apis/gateway/v2"
	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
)

func TestStringsToMethods(t *testing.T) {
	// arrange
	givenStrings := []string{
		http.MethodGet,
		http.MethodPut,
		http.MethodHead,
		http.MethodPost,
		http.MethodPatch,
		http.MethodConnect,
		http.MethodTrace,
		http.MethodOptions,
		http.MethodDelete,
	}

	// act
	methods := StringsToMethods(givenStrings)
	actualStrings := apigatewayv2.ConvertHttpMethodsToStrings(methods)

	// assert
	if !reflect.DeepEqual(givenStrings, actualStrings) {
		t.Fatalf("slices of strings are not. wanted: %v but got %v", givenStrings, actualStrings)
	}
}

func TestApplyExistingAPIRuleAttributes(t *testing.T) {
	// given
	const (
		name            = "name-0"
		generateName    = "0123"
		resourceVersion = "4567"
	)

	var (
		h      = apigatewayv2.Host("some.host")
		hosts  = []*apigatewayv2.Host{&h}
		status = apigatewayv2.APIRuleStatus{
			LastProcessedTime: *ptr.To(kmetav1.Time{}),
		}
	)

	type args struct {
		givenSrc *apigatewayv2.APIRule
		givenDst *apigatewayv2.APIRule
		wantDst  *apigatewayv2.APIRule
	}
	testCases := []struct {
		name string
		args args
	}{
		{
			name: "ApiRule attributes are applied from src to dst",
			args: args{
				givenSrc: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name,
						GenerateName:    generateName,
						ResourceVersion: resourceVersion,
					},
					Spec:   apigatewayv2.APIRuleSpec{Hosts: hosts},
					Status: status,
				},
				givenDst: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name,
						GenerateName:    generateName,
						ResourceVersion: resourceVersion,
					},
					Spec:   apigatewayv2.APIRuleSpec{Hosts: hosts},
					Status: status,
				},
				wantDst: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:            name,
						GenerateName:    "",
						ResourceVersion: resourceVersion,
					},
					Spec:   apigatewayv2.APIRuleSpec{Hosts: hosts},
					Status: status,
				},
			},
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			ApplyExistingAPIRuleAttributes(testcase.args.givenSrc, testcase.args.givenDst)

			// then
			require.Equal(t, testcase.args.wantDst.Name, testcase.args.givenDst.Name)
			require.Equal(t, testcase.args.wantDst.GenerateName, testcase.args.givenDst.GenerateName)
			require.Equal(t, testcase.args.wantDst.ResourceVersion, testcase.args.givenDst.ResourceVersion)
			require.Equal(t, testcase.args.wantDst.Spec, testcase.args.givenDst.Spec)
			require.Equal(t, testcase.args.wantDst.Status, testcase.args.givenDst.Status)
		})
	}
}

func TestGetService(t *testing.T) {
	// given
	const (
		name       = "name-0"
		port       = uint32(9080)
		isExternal = true
	)

	type args struct {
		svcName string
		port    uint32
	}
	tests := []struct {
		name string
		args args
		want apigatewayv2.Service
	}{
		{
			name: "get service with the given properties",
			args: args{
				svcName: name,
				port:    port,
			},
			want: apigatewayv2.Service{
				Name:       ptr.To(name),
				Port:       ptr.To(port),
				IsExternal: ptr.To(isExternal),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			if got := GetService(tt.args.svcName, tt.args.port); !reflect.DeepEqual(got, tt.want) {
				// then
				t.Errorf("GetService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewAPIRule(t *testing.T) {
	// given
	const (
		namespace  = "namespace-0"
		namePrefix = "name-0"
	)

	type args struct {
		ns         string
		namePrefix string
		opts       []Option
	}
	tests := []struct {
		name string
		args args
		want *apigatewayv2.APIRule
	}{
		{
			name: "get APIRule with the given properties",
			args: args{
				ns:         namespace,
				namePrefix: namePrefix,
				opts:       nil,
			},
			want: &apigatewayv2.APIRule{
				TypeMeta: kmetav1.TypeMeta{},
				ObjectMeta: kmetav1.ObjectMeta{
					Namespace:    namespace,
					GenerateName: namePrefix,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			if got := NewAPIRule(tt.args.ns, tt.args.namePrefix, tt.args.opts...); !reflect.DeepEqual(got, tt.want) {
				// then
				t.Errorf("NewAPIRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveDuplicateValues(t *testing.T) {
	// given
	type args struct {
		values []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "list without duplicates",
			args: args{
				values: []string{
					"1", "2", "3",
				},
			},
			want: []string{
				"1", "2", "3",
			},
		},
		{
			name: "list with duplicates",
			args: args{
				values: []string{
					"1", "2", "3",
					"3", "2", "1",
				},
			},
			want: []string{
				"1", "2", "3",
			},
		},
		{
			name: "empty list",
			args: args{
				values: []string{},
			},
			want: []string{},
		},
		{
			name: "nil list",
			args: args{
				values: nil,
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			if got := RemoveDuplicateValues(tt.args.values); !reflect.DeepEqual(got, tt.want) {
				// then
				t.Errorf("RemoveDuplicateValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithGateway(t *testing.T) {
	// given
	const (
		gateway = "some.gateway"
	)

	type args struct {
		givenGateway string
		givenObject  *apigatewayv2.APIRule
	}
	testCases := []struct {
		name       string
		args       args
		wantObject *apigatewayv2.APIRule
	}{
		{
			name: "apply gateway to object",
			args: args{
				givenGateway: gateway,
				givenObject:  &apigatewayv2.APIRule{},
			},
			wantObject: &apigatewayv2.APIRule{
				Spec: apigatewayv2.APIRuleSpec{
					Gateway: ptr.To(gateway),
				},
			},
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			WithGateway(testcase.args.givenGateway)(testcase.args.givenObject)

			// then
			require.Equal(t, testcase.wantObject.Spec.Gateway, testcase.args.givenObject.Spec.Gateway)
		})
	}
}

func TestWithLabels(t *testing.T) {
	// given
	type args struct {
		givenLabels map[string]string
		givenObject *apigatewayv2.APIRule
	}
	testCases := []struct {
		name       string
		args       args
		wantObject *apigatewayv2.APIRule
	}{
		{
			name: "object with nil labels",
			args: args{
				givenLabels: map[string]string{
					"key-0": "val-0",
				},
				givenObject: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						Labels: nil,
					},
				},
			},
			wantObject: &apigatewayv2.APIRule{
				ObjectMeta: kmetav1.ObjectMeta{
					Labels: map[string]string{
						"key-0": "val-0",
					},
				},
			},
		},
		{
			name: "object with empty labels",
			args: args{
				givenLabels: map[string]string{
					"key-0": "val-0",
				},
				givenObject: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						Labels: map[string]string{},
					},
				},
			},
			wantObject: &apigatewayv2.APIRule{
				ObjectMeta: kmetav1.ObjectMeta{
					Labels: map[string]string{
						"key-0": "val-0",
					},
				},
			},
		},
		{
			name: "object with labels",
			args: args{
				givenLabels: map[string]string{
					"key-0": "val-0",
				},
				givenObject: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						Labels: map[string]string{
							"key-1": "val-1",
							"key-2": "val-2",
						},
					},
				},
			},
			wantObject: &apigatewayv2.APIRule{
				ObjectMeta: kmetav1.ObjectMeta{
					Labels: map[string]string{
						"key-0": "val-0",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			WithLabels(testcase.args.givenLabels)(testcase.args.givenObject)

			// then
			require.Equal(t, testcase.wantObject.GetLabels(), testcase.args.givenObject.GetLabels())
		})
	}
}

func TestWithOwnerReference(t *testing.T) {
	// given
	const (
		kind0 = "kind-0"
		kind1 = "kind-1"
		kind2 = "kind-2"

		apiVersion0 = "version-0"
		apiVersion1 = "version-1"
		apiVersion2 = "version-2"

		name0 = "name-0"
		name1 = "name-1"
		name2 = "name-2"

		uid0 = "000000"
		uid1 = "111111"
		uid2 = "222222"

		blockOwnerDeletion = true
	)

	var (
		sub0 = eventingv1alpha2.Subscription{
			TypeMeta:   kmetav1.TypeMeta{Kind: kind0, APIVersion: apiVersion0},
			ObjectMeta: kmetav1.ObjectMeta{Name: name0, UID: uid0},
		}
		sub1 = eventingv1alpha2.Subscription{
			TypeMeta:   kmetav1.TypeMeta{Kind: kind1, APIVersion: apiVersion1},
			ObjectMeta: kmetav1.ObjectMeta{Name: name1, UID: uid1},
		}
	)

	type args struct {
		givenSubs   []eventingv1alpha2.Subscription
		givenObject *apigatewayv2.APIRule
	}
	tests := []struct {
		name       string
		args       args
		wantObject *apigatewayv2.APIRule
	}{
		{
			name: "nil Subscriptions",
			args: args{
				givenSubs: nil,
				givenObject: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						OwnerReferences: nil,
					},
				},
			},
			wantObject: &apigatewayv2.APIRule{
				ObjectMeta: kmetav1.ObjectMeta{
					OwnerReferences: []kmetav1.OwnerReference{},
				},
			},
		},
		{
			name: "empty Subscriptions",
			args: args{
				givenSubs: []eventingv1alpha2.Subscription{},
				givenObject: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						OwnerReferences: nil,
					},
				},
			},
			wantObject: &apigatewayv2.APIRule{
				ObjectMeta: kmetav1.ObjectMeta{
					OwnerReferences: []kmetav1.OwnerReference{},
				},
			},
		},
		{
			name: "object with nil OwnerReferences",
			args: args{
				givenSubs: []eventingv1alpha2.Subscription{
					sub0,
					sub1,
				},
				givenObject: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						OwnerReferences: nil,
					},
				},
			},
			wantObject: &apigatewayv2.APIRule{
				ObjectMeta: kmetav1.ObjectMeta{
					OwnerReferences: []kmetav1.OwnerReference{
						{
							APIVersion:         apiVersion0,
							Kind:               kind0,
							Name:               name0,
							UID:                uid0,
							BlockOwnerDeletion: ptr.To(blockOwnerDeletion),
						},
						{
							APIVersion:         apiVersion1,
							Kind:               kind1,
							Name:               name1,
							UID:                uid1,
							BlockOwnerDeletion: ptr.To(blockOwnerDeletion),
						},
					},
				},
			},
		},
		{
			name: "object with empty OwnerReferences",
			args: args{
				givenSubs: []eventingv1alpha2.Subscription{
					sub0,
					sub1,
				},
				givenObject: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						OwnerReferences: []kmetav1.OwnerReference{},
					},
				},
			},
			wantObject: &apigatewayv2.APIRule{
				ObjectMeta: kmetav1.ObjectMeta{
					OwnerReferences: []kmetav1.OwnerReference{
						{
							APIVersion:         apiVersion0,
							Kind:               kind0,
							Name:               name0,
							UID:                uid0,
							BlockOwnerDeletion: ptr.To(blockOwnerDeletion),
						},
						{
							APIVersion:         apiVersion1,
							Kind:               kind1,
							Name:               name1,
							UID:                uid1,
							BlockOwnerDeletion: ptr.To(blockOwnerDeletion),
						},
					},
				},
			},
		},
		{
			name: "object with OwnerReferences",
			args: args{
				givenSubs: []eventingv1alpha2.Subscription{
					sub0,
					sub1,
				},
				givenObject: &apigatewayv2.APIRule{
					ObjectMeta: kmetav1.ObjectMeta{
						OwnerReferences: []kmetav1.OwnerReference{
							{
								APIVersion:         apiVersion2,
								Kind:               kind2,
								Name:               name2,
								UID:                uid2,
								BlockOwnerDeletion: ptr.To(blockOwnerDeletion),
							},
						},
					},
				},
			},
			wantObject: &apigatewayv2.APIRule{
				ObjectMeta: kmetav1.ObjectMeta{
					OwnerReferences: []kmetav1.OwnerReference{
						{
							APIVersion:         apiVersion0,
							Kind:               kind0,
							Name:               name0,
							UID:                uid0,
							BlockOwnerDeletion: ptr.To(blockOwnerDeletion),
						},
						{
							APIVersion:         apiVersion1,
							Kind:               kind1,
							Name:               name1,
							UID:                uid1,
							BlockOwnerDeletion: ptr.To(blockOwnerDeletion),
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			WithOwnerReference(testcase.args.givenSubs)(testcase.args.givenObject)
			gotOwnerReferences := testcase.args.givenObject.GetOwnerReferences()

			// then
			require.Equal(t, testcase.wantObject.GetOwnerReferences(), gotOwnerReferences)
		})
	}
}

func TestWithRules(t *testing.T) {
	// given
	const (
		endpoint0 = "/endpoint0"
		endpoint1 = "/endpoint1"

		sink0 = "https://sink0.com" + endpoint0
		sink1 = "https://sink1.com" + endpoint1

		certsURL  = "some.url/oauth2/certs"
		issuerURL = "some.url"
		name      = "name-0"
		port      = uint32(9999)
		external  = true
	)

	var (
		sub0 = eventingv1alpha2.Subscription{
			Spec: eventingv1alpha2.SubscriptionSpec{Sink: sink0},
		}
		sub1 = eventingv1alpha2.Subscription{
			Spec: eventingv1alpha2.SubscriptionSpec{Sink: sink1},
		}

		methods = []string{
			http.MethodGet,
		}
	)

	type args struct {
		givenCertsURL string
		givenSubs     []eventingv1alpha2.Subscription
		givenSvc      apigatewayv2.Service
		givenMethods  []string
		givenObject   *apigatewayv2.APIRule
	}
	tests := []struct {
		name       string
		args       args
		wantObject *apigatewayv2.APIRule
	}{
		{
			name: "apply properties to object",
			args: args{
				givenCertsURL: certsURL,
				givenSubs: []eventingv1alpha2.Subscription{
					sub0,
					sub1,
				},
				givenSvc: apigatewayv2.Service{
					Name:       ptr.To(name),
					Port:       ptr.To(port),
					IsExternal: ptr.To(external),
				},
				givenMethods: methods,
				givenObject:  &apigatewayv2.APIRule{},
			},
			wantObject: &apigatewayv2.APIRule{
				Spec: apigatewayv2.APIRuleSpec{
					Rules: []apigatewayv2.Rule{
						{
							Path: endpoint0,
							Service: &apigatewayv2.Service{
								Name:       ptr.To(name),
								Port:       ptr.To(port),
								IsExternal: ptr.To(external),
							},
							Methods: StringsToMethods(methods),
							Jwt: &apigatewayv2.JwtConfig{
								Authentications: []*apigatewayv2.JwtAuthentication{
									{
										JwksUri: certsURL,
										Issuer:  issuerURL,
									},
								},
							},
						},
						{
							Path: endpoint1,
							Service: &apigatewayv2.Service{
								Name:       ptr.To(name),
								Port:       ptr.To(port),
								IsExternal: ptr.To(external),
							},
							Methods: StringsToMethods(methods),
							Jwt: &apigatewayv2.JwtConfig{
								Authentications: []*apigatewayv2.JwtAuthentication{
									{
										JwksUri: certsURL,
										Issuer:  issuerURL,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			WithRules(testcase.args.givenCertsURL, testcase.args.givenSubs, testcase.args.givenSvc, testcase.args.givenMethods...)(testcase.args.givenObject)

			// then
			require.Equal(t, testcase.wantObject.Spec.Rules, testcase.args.givenObject.Spec.Rules)
		})
	}
}

func TestWithService(t *testing.T) {
	// given
	host := apigatewayv2.Host("host0")
	const (
		name     = "name-0"
		port     = uint32(9999)
		external = true
	)

	type args struct {
		givenHosts   []*apigatewayv2.Host
		givenSvcName string
		givenPort    uint32
		givenObject  *apigatewayv2.APIRule
	}
	tests := []struct {
		name       string
		args       args
		wantObject *apigatewayv2.APIRule
	}{
		{
			name: "apply properties to object",
			args: args{
				givenHosts:   []*apigatewayv2.Host{&host},
				givenSvcName: name,
				givenPort:    port,
				givenObject:  &apigatewayv2.APIRule{},
			},
			wantObject: &apigatewayv2.APIRule{
				Spec: apigatewayv2.APIRuleSpec{
					Hosts: []*apigatewayv2.Host{&host},
					Service: &apigatewayv2.Service{
						Name:       ptr.To(name),
						Port:       ptr.To(port),
						IsExternal: ptr.To(external),
					},
				},
			},
		},
	}
	for _, tc := range tests {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			WithService(testcase.args.givenHosts, testcase.args.givenSvcName, testcase.args.givenPort)(testcase.args.givenObject)

			// then
			require.Equal(t, testcase.wantObject, testcase.args.givenObject)
		})
	}
}
