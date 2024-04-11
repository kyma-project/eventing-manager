package sink

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

func TestSinkValidator(t *testing.T) {
	// given
	namespaceName := "test"
	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	ctx := context.Background()
	recorder := &record.FakeRecorder{}
	sinkValidator := NewValidator(fakeClient, recorder)

	testCases := []struct {
		name                  string
		givenSubscriptionSink string
		givenSvcNameToCreate  string
		wantErrString         string
	}{
		{
			name:                  "With invalid URL",
			givenSubscriptionSink: "http://invalid Sink",
			wantErrString:         "failed to parse subscription sink URL",
		},
		{
			name:                  "With no existing svc in the cluster",
			givenSubscriptionSink: "https://eventing-nats.test.svc.cluster.local:8080",
			wantErrString:         "Sink validation failed",
		},
		{
			name:                  "With no existing svc in the cluster, service has the wrong name",
			givenSubscriptionSink: "https://eventing-nats.test.svc.cluster.local:8080",
			givenSvcNameToCreate:  "test", // wrong name
			wantErrString:         "Sink validation failed",
		},
		{
			name:                  "With a valid sink",
			givenSubscriptionSink: "https://eventing-nats.test.svc.cluster.local:8080",
			givenSvcNameToCreate:  "eventing-nats",
			wantErrString:         "",
		},
	}

	for _, tC := range testCases {
		testCase := tC
		t.Run(testCase.name, func(t *testing.T) {
			// given
			sub := eventingtesting.NewSubscription(
				"foo", namespaceName,
				eventingtesting.WithConditions([]eventingv1alpha2.Condition{}),
				eventingtesting.WithStatus(true),
				eventingtesting.WithSink(testCase.givenSubscriptionSink),
			)

			// create the service if required for test
			if testCase.givenSvcNameToCreate != "" {
				svc := &kcorev1.Service{
					ObjectMeta: kmetav1.ObjectMeta{
						Name:      testCase.givenSvcNameToCreate,
						Namespace: namespaceName,
					},
				}

				err := fakeClient.Create(ctx, svc)
				require.NoError(t, err)
			}

			// when
			// call the defaultSinkValidator function
			err := sinkValidator.Validate(ctx, sub)

			// then
			// given error should match expected error
			if testCase.wantErrString == "" {
				require.NoError(t, err)
			} else {
				substringResult := strings.Contains(err.Error(), testCase.wantErrString)
				require.True(t, substringResult)
			}
		})
	}
}
