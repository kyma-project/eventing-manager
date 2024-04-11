package v1alpha2_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

const (
	subName      = "sub"
	subNamespace = "test"
	sink         = "https://eventing-nats.test.svc.cluster.local:8080"
)

func Test_validateSubscription(t *testing.T) {
	t.Parallel()
	type TestCase struct {
		name     string
		givenSub *v1alpha2.Subscription
		wantErr  field.ErrorList
	}

	testCases := []TestCase{
		{
			name: "A valid subscription should not have errors",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithWebhookAuthForEventMesh(),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: nil,
		},
		{
			name: "empty source and TypeMatching Standard should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.SourcePath,
				"", v1alpha2.EmptyErrDetail)},
		},
		{
			name: "valid source and TypeMatching Standard should not return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: nil,
		},
		{
			name: "empty source and TypeMatching Exact should not return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingExact(),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
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
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.SourcePath,
				"s%ourc%e", v1alpha2.InvalidURIErrDetail)},
		},
		{
			name: "nil types field should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.TypesPath,
				"", v1alpha2.EmptyErrDetail)},
		},
		{
			name: "empty types field should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{}),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.TypesPath,
				"", v1alpha2.EmptyErrDetail)},
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
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.TypesPath,
				"order.created.v1", v1alpha2.DuplicateTypesErrDetail)},
		},
		{
			name: "empty event type should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{eventingtesting.OrderCreatedV1Event, ""}),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.TypesPath,
				"", v1alpha2.LengthErrDetail)},
		},
		{
			name: "lower than min segments should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{"order"}),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.TypesPath,
				"order", v1alpha2.MinSegmentErrDetail)},
		},
		{
			name: "invalid prefix should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{v1alpha2.InvalidPrefix}),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.TypesPath,
				"sap.kyma.custom", v1alpha2.InvalidPrefixErrDetail)},
		},
		{
			name: "invalid prefix with exact should not return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingExact(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithTypes([]string{v1alpha2.InvalidPrefix}),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
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
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.ConfigPath,
				"invalid", v1alpha2.StringIntErrDetail)},
		},
		{
			name: "invalid QoS value should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithInvalidProtocolSettingsQos(),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.ConfigPath,
				"AT_INVALID_ONCE", v1alpha2.InvalidQosErrDetail)},
		},
		{
			name: "invalid webhook auth type value should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithInvalidWebhookAuthType(),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.ConfigPath,
				"abcd", v1alpha2.InvalidAuthTypeErrDetail)},
		},
		{
			name: "invalid webhook grant type value should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithInvalidWebhookAuthGrantType(),
				eventingtesting.WithSink(sink),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.ConfigPath,
				"invalid", v1alpha2.InvalidGrantTypeErrDetail)},
		},
		{
			name: "missing sink should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.SinkPath,
				"", v1alpha2.EmptyErrDetail)},
		},
		{
			name: "sink with invalid scheme should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink(subNamespace),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.SinkPath,
				"test", v1alpha2.MissingSchemeErrDetail)},
		},
		{
			name: "sink with invalid URL should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink("http://invalid Sink"),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.SinkPath,
				"http://invalid Sink", "failed to parse subscription sink URL: "+
					"parse \"http://invalid Sink\": invalid character \" \" in host name")},
		},
		{
			name: "sink with invalid suffix should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink("https://svc2.test.local"),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.SinkPath,
				"https://svc2.test.local", v1alpha2.SuffixMissingErrDetail)},
		},
		{
			name: "sink with invalid suffix and port should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink("https://svc2.test.local:8080"),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.SinkPath,
				"https://svc2.test.local:8080", v1alpha2.SuffixMissingErrDetail)},
		},
		{
			name: "sink with invalid number of subdomains should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink("https://svc.cluster.local:8080"),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.SinkPath,
				"https://svc.cluster.local:8080", v1alpha2.SubDomainsErrDetail+"svc.cluster.local")},
		},
		{
			name: "sink with different namespace should return error",
			givenSub: eventingtesting.NewSubscription(subName, subNamespace,
				eventingtesting.WithTypeMatchingStandard(),
				eventingtesting.WithSource(eventingtesting.EventSourceClean),
				eventingtesting.WithEventType(eventingtesting.OrderCreatedV1Event),
				eventingtesting.WithMaxInFlightMessages(v1alpha2.DefaultMaxInFlightMessages),
				eventingtesting.WithSink("https://eventing-nats.kyma-system.svc.cluster.local"),
			),
			wantErr: field.ErrorList{v1alpha2.MakeInvalidFieldError(v1alpha2.NSPath,
				"https://eventing-nats.kyma-system.svc.cluster.local", v1alpha2.NSMismatchErrDetail+"kyma-system")},
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
				v1alpha2.MakeInvalidFieldError(v1alpha2.SourcePath,
					"", v1alpha2.EmptyErrDetail),
				v1alpha2.MakeInvalidFieldError(v1alpha2.ConfigPath,
					"invalid", v1alpha2.StringIntErrDetail),
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.givenSub.ValidateSpec()
			require.Equal(t, tc.wantErr, err)
		})
	}
}

func Test_IsInvalidCESource(t *testing.T) {
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
			gotIsInvalid := v1alpha2.IsInvalidCE(tc.givenSource, tc.givenType)
			require.Equal(t, tc.wantIsInvalid, gotIsInvalid)
		})
	}
}
