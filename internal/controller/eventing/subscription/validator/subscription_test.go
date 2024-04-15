package validator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

func TestValidate(t *testing.T) {
	ctx := context.TODO()

	const (
		subName             = "sub"
		subNamespace        = "test"
		maxInFlightMessages = "10"
		sink                = "https://eventing-nats.test.svc.cluster.local:8080"
	)

	happySinkValidator := SinkValidatorFunc(func(ctx context.Context, sinkURI string) error { return nil })
	unhappySinkValidator := SinkValidatorFunc(func(ctx context.Context, sinkURI string) error { return ErrSinkValidationFailed })

	tests := []struct {
		name               string
		givenSubscription  *eventingv1alpha2.Subscription
		givenSinkValidator SinkValidator
		wantErr            error
	}{
		{
			name: "simulate no validation errors",
			givenSubscription: eventingtesting.NewSubscription(
				subName,
				subNamespace,
				eventingtesting.WithTypeMatchingExact(),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			givenSinkValidator: happySinkValidator,
			wantErr:            nil,
		},
		{
			name: "simulate spec validation error",
			givenSubscription: eventingtesting.NewSubscription(
				subName,
				subNamespace,
				eventingtesting.WithTypeMatchingExact(),
				eventingtesting.WithTypes([]string{}), // Trigger validation error.
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			givenSinkValidator: happySinkValidator,
			wantErr:            ErrSubscriptionValidationFailed,
		},
		{
			name: "simulate sink validation error",
			givenSubscription: eventingtesting.NewSubscription(
				subName,
				subNamespace,
				eventingtesting.WithTypeMatchingExact(),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			givenSinkValidator: unhappySinkValidator, // Trigger sink validation error.
			wantErr:            ErrSinkValidationFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := &subscriptionValidator{sinkValidator: test.givenSinkValidator}
			gotErr := validator.Validate(ctx, *test.givenSubscription)
			if test.wantErr == nil {
				require.NoError(t, gotErr)
			} else {
				require.ErrorIs(t, gotErr, test.wantErr)
			}
		})
	}
}
