package validator

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

func Test_validateSpec(t *testing.T) {
	t.Parallel()

	const (
		subName             = "sub"
		subNamespace        = "test"
		maxInFlightMessages = "10"
		sink                = "https://eventing-nats.test.svc.cluster.local:8080"
	)

	type TestCase struct {
		name     string
		givenSub *eventingv1alpha2.Subscription
		wantErr  field.ErrorList
	}

	testCases := []TestCase{
		{
			name: "A valid subscription should not have errors",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithWebhookAuthForEventMesh(),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: nil,
		},
		{
			name: "empty source and TypeMatching Standard should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(sourcePath,
				"", emptyErrDetail)},
		},
		{
			name: "valid source and TypeMatching Standard should not return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: nil,
		},
		{
			name: "empty source and TypeMatching Exact should not return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingExact(),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: nil,
		},
		{
			name: "invalid URI reference as source should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource("s%ourc%e"),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(sourcePath,
				"s%ourc%e", invalidURIErrDetail)},
		},
		{
			name: "nil types field should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(typesPath,
				"", emptyErrDetail)},
		},
		{
			name: "empty types field should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{}),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(typesPath,
				"", emptyErrDetail)},
		},
		{
			name: "duplicate types should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{
					eventingtesting.OrderCreatedV1Event,
					eventingtesting.OrderCreatedV1Event,
				}),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(typesPath,
				"order.created.v1", duplicateTypesErrDetail)},
		},
		{
			name: "empty event type should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{eventingtesting.OrderCreatedV1Event, ""}),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(typesPath,
				"", lengthErrDetail)},
		},
		{
			name: "lower than min segments should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{"order"}),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(typesPath,
				"order", minSegmentErrDetail)},
		},
		{
			name: "invalid prefix should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{validPrefix}),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{
				makeInvalidFieldError(typesPath, "sap.kyma.custom", invalidPrefixErrDetail),
			},
		},
		{
			name: "invalid prefix with exact should not return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingExact(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{validPrefix}),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: nil,
		},
		{
			name: "invalid maxInFlight value should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages("invalid"),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(configPath,
				"invalid", stringIntErrDetail)},
		},
		{
			name: "invalid QoS value should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithInvalidProtocolSettingsQos(),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(configPath,
				"AT_INVALID_ONCE", invalidQosErrDetail)},
		},
		{
			name: "invalid webhook auth type value should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithInvalidWebhookAuthType(),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(configPath,
				"abcd", invalidAuthTypeErrDetail)},
		},
		{
			name: "invalid webhook grant type value should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithInvalidWebhookAuthGrantType(),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(configPath,
				"invalid", invalidGrantTypeErrDetail)},
		},
		{
			name: "missing sink should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(sinkPath,
				"", emptyErrDetail)},
		},
		{
			name: "sink with invalid scheme should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink(subNamespace),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(sinkPath,
				"test", missingSchemeErrDetail)},
		},
		{
			name: "sink with invalid URL should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink("http://invalid Sink"),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(sinkPath,
				"http://invalid Sink", "failed to parse subscription sink URL: "+
					"parse \"http://invalid Sink\": invalid character \" \" in host name")},
		},
		{
			name: "sink with invalid suffix should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink("https://svc2.test.local"),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(sinkPath,
				"https://svc2.test.local", suffixMissingErrDetail)},
		},
		{
			name: "sink with invalid suffix and port should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink("https://svc2.test.local:8080"),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(sinkPath,
				"https://svc2.test.local:8080", suffixMissingErrDetail)},
		},
		{
			name: "sink with invalid number of subdomains should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink("https://svc.cluster.local:8080"),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(sinkPath,
				"https://svc.cluster.local:8080", subDomainsErrDetail+"svc.cluster.local")},
		},
		{
			name: "sink with different namespace should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(maxInFlightMessages),
				eventingtesting.WithSink("https://eventing-nats.kyma-system.svc.cluster.local"),
			),
			wantErr: field.ErrorList{makeInvalidFieldError(namespacePath,
				"https://eventing-nats.kyma-system.svc.cluster.local", namespaceMismatchErrDetail+"kyma-system")},
		},
		{
			name: "multiple errors should be reported if exists",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages("invalid"),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{
				makeInvalidFieldError(sourcePath,
					"", emptyErrDetail),
				makeInvalidFieldError(configPath,
					"invalid", stringIntErrDetail),
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateSpec(*tc.givenSub)
			require.Equal(t, tc.wantErr, err)
		})
	}
}

func Test_isInvalidCE(t *testing.T) {
	t.Parallel()
	type TestCase struct {
		name          string
		givenSource   string
		givenType     string
		wantIsInvalid bool
	}

	testCases := []TestCase{
		{
			name:          "invalid URI Path source should be invalid",
			givenSource:   "app%%type",
			givenType:     "order.created.v1",
			wantIsInvalid: true,
		},
		{
			name:          "valid URI Path source should not be invalid",
			givenSource:   "t..e--s__t!!a@@**p##p&&",
			givenType:     "",
			wantIsInvalid: false,
		},
		{
			name:          "should ignore check if the source is empty",
			givenSource:   "",
			givenType:     "",
			wantIsInvalid: false,
		},
		{
			name:          "invalid type should be invalid",
			givenSource:   "source",
			givenType:     " ", // space is an invalid type for cloud event
			wantIsInvalid: true,
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotIsInvalid := isInvalidCE(tc.givenSource, tc.givenType)
			require.Equal(t, tc.wantIsInvalid, gotIsInvalid)
		})
	}
}
