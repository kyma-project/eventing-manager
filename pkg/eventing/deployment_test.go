package eventing

import (
	"fmt"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"

	"github.com/kyma-project/eventing-manager/api/operator.kyma-project.io/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/test"
	testutils "github.com/kyma-project/eventing-manager/test/utils"
)

const (
	natsURL         = "eventing-nats.kyma-system.svc.cluster.local"
	eventTypePrefix = "test.prefix"
)

func TestNewDeployment(t *testing.T) {
	publisherConfig := env.PublisherConfig{
		Image:           "testImage",
		ImagePullPolicy: "Always",
		AppLogFormat:    "json",
	}
	testCases := []struct {
		name                  string
		givenPublisherName    string
		givenBackendType      v1alpha1.BackendType
		wantBackendAssertions func(t *testing.T, publisherName string, deployment appsv1.Deployment)
	}{
		{
			name:                  "NATS should be set properly after calling the constructor",
			givenPublisherName:    "test-name",
			givenBackendType:      v1alpha1.NatsBackendType,
			wantBackendAssertions: natsBackendAssertions,
		},
		{
			name:                  "EventMesh should be set properly after calling the constructor",
			givenPublisherName:    "test-name",
			givenBackendType:      v1alpha1.EventMeshBackendType,
			wantBackendAssertions: eventMeshBackendAssertions,
		},
	}

	publisherName := fmt.Sprintf("%s-%s", "test-name", publisherProxySuffix)
	publisherNamespace := "test-namespace"
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var deployment *appsv1.Deployment
			var natsConfig env.NATSConfig

			switch tc.givenBackendType {
			case "NATS":
				natsConfig = env.NATSConfig{
					JSStreamName: "kyma",
					URL:          natsURL,
				}
				deployment = newNATSPublisherDeployment(testutils.NewEventingCR(
					testutils.WithEventingCRName(tc.givenPublisherName),
					testutils.WithEventingCRNamespace(publisherNamespace),
					testutils.WithEventingEventTypePrefix(eventTypePrefix),
				), natsConfig, publisherConfig)
			case "EventMesh":
				deployment = newEventMeshPublisherDeployment(testutils.NewEventingCR(
					testutils.WithEventingCRName(tc.givenPublisherName),
					testutils.WithEventingCRNamespace(publisherNamespace),
					testutils.WithEventMeshBackend("test-namespace/test-name"),
				), publisherConfig)
			default:
				t.Errorf("Invalid backend!")
			}

			// the tight backenType should be set
			assert.Equal(t, deployment.ObjectMeta.Labels[BackendLabelKey], string(getECBackendType(tc.givenBackendType)))
			assert.Equal(t, deployment.ObjectMeta.Labels[AppLabelKey], publisherName)

			// check the container properties were set properly
			container := findPublisherContainer(publisherName, *deployment)
			assert.NotNil(t, container)

			assert.Equal(t, fmt.Sprint(container.Name), publisherName)
			assert.Equal(t, fmt.Sprint(container.Image), publisherConfig.Image)
			assert.Equal(t, fmt.Sprint(container.ImagePullPolicy), publisherConfig.ImagePullPolicy)

			tc.wantBackendAssertions(t, publisherName, *deployment)
		})
	}
}

func Test_NewDeploymentSecurityContext(t *testing.T) {
	// given
	config := env.GetBackendConfig()
	givenEventing := testutils.NewEventingCR(
		testutils.WithEventingCRName("tets-deployment"),
		testutils.WithEventingCRNamespace("test-namespace"),
	)
	deployment := newDeployment(givenEventing, config.PublisherConfig,
		WithContainers(config.PublisherConfig, givenEventing),
	)

	// when
	podSecurityContext := deployment.Spec.Template.Spec.SecurityContext
	containerSecurityContext := deployment.Spec.Template.Spec.Containers[0].SecurityContext

	// then
	assert.Equal(t, getPodSecurityContext(), podSecurityContext)
	assert.Equal(t, getContainerSecurityContext(), containerSecurityContext)
}

func Test_GetNATSEnvVars(t *testing.T) {
	testCases := []struct {
		name            string
		givenEnvs       map[string]string
		givenNATSConfig env.NATSConfig
		givenEventing   *v1alpha1.Eventing
		wantEnvs        []v1.EnvVar
	}{
		{
			name: "JS envs should stay empty",
			givenEnvs: map[string]string{
				"PUBLISHER_REQUEST_TIMEOUT": "10s",
			},
			givenEventing: testutils.NewEventingCR(),
			wantEnvs: []v1.EnvVar{
				{Name: "BACKEND", Value: "nats"},
				{Name: "PORT", Value: "8080"},
				{Name: "NATS_URL", Value: ""},
				{Name: "REQUEST_TIMEOUT", Value: "10s"},
				{Name: "LEGACY_NAMESPACE", Value: "kyma"},
				{Name: "EVENT_TYPE_PREFIX", Value: ""},
				{Name: "APPLICATION_CRD_ENABLED", Value: "false"},
				{Name: "JS_STREAM_NAME", Value: ""},
			},
		},
		{
			name: "Test the REQUEST_TIMEOUT and non-empty NatsConfig",
			givenEnvs: map[string]string{
				"PUBLISHER_REQUEST_TIMEOUT": "10s",
			},
			givenNATSConfig: env.NATSConfig{
				JSStreamName: "sap",
				URL:          "test-url",
			},
			givenEventing: testutils.NewEventingCR(),
			wantEnvs: []v1.EnvVar{
				{Name: "BACKEND", Value: "nats"},
				{Name: "PORT", Value: "8080"},
				{Name: "NATS_URL", Value: "test-url"},
				{Name: "REQUEST_TIMEOUT", Value: "10s"},
				{Name: "LEGACY_NAMESPACE", Value: "kyma"},
				{Name: "EVENT_TYPE_PREFIX", Value: ""},
				{Name: "APPLICATION_CRD_ENABLED", Value: "false"},
				{Name: "JS_STREAM_NAME", Value: "sap"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.givenEnvs {
				t.Setenv(k, v)
			}
			backendConfig := env.GetBackendConfig()
			envVars := getNATSEnvVars(tc.givenNATSConfig, backendConfig.PublisherConfig, tc.givenEventing)

			// ensure the right envs were set
			require.Equal(t, tc.wantEnvs, envVars)
		})
	}
}
func Test_GetLogEnvVars(t *testing.T) {
	testCases := []struct {
		name          string
		givenEventing *v1alpha1.Eventing
		wantEnvs      []v1.EnvVar
	}{
		{
			name: "APP_LOG_FORMAT should be text and APP_LOG_LEVEL should become the default info value",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingLogLevel("Info"),
			),
			wantEnvs: []v1.EnvVar{
				{Name: "APP_LOG_FORMAT", Value: "json"},
				{Name: "APP_LOG_LEVEL", Value: "info"},
			},
		},
		{
			name: "APP_LOG_FORMAT should become default json and APP_LOG_LEVEL should be warning",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingLogLevel("Warn"),
			),
			wantEnvs: []v1.EnvVar{
				{Name: "APP_LOG_FORMAT", Value: "json"},
				{Name: "APP_LOG_LEVEL", Value: "warn"},
			},
		},
		{
			name: "APP_LOG_FORMAT should be testFormat and APP_LOG_LEVEL should be error",
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventingLogLevel("Error"),
			),
			wantEnvs: []v1.EnvVar{
				{Name: "APP_LOG_FORMAT", Value: "json"},
				{Name: "APP_LOG_LEVEL", Value: "error"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			backendConfig := env.GetBackendConfig()
			envVars := getLogEnvVars(backendConfig.PublisherConfig, tc.givenEventing)

			// ensure the right envs were set
			require.Equal(t, tc.wantEnvs, envVars)
		})
	}
}

func Test_GetEventMeshEnvVars(t *testing.T) {
	testCases := []struct {
		name          string
		givenEnvs     map[string]string
		givenEventing *v1alpha1.Eventing
		wantEnvs      map[string]string
	}{
		{
			name: "REQUEST_TIMEOUT is not set, the default value should be taken",
			givenEnvs: map[string]string{
				"PUBLISHER_REQUESTS_CPU": "64m",
			},
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventMeshBackend("test-namespace/test-name"),
			),
			wantEnvs: map[string]string{
				"REQUEST_TIMEOUT": "5s", // default value
			},
		},
		{
			name: "REQUEST_TIMEOUT should be set",
			givenEnvs: map[string]string{
				"PUBLISHER_REQUEST_TIMEOUT": "10s",
			},
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventMeshBackend("test-namespace/test-name"),
				testutils.WithEventingEventTypePrefix(eventTypePrefix),
			),
			wantEnvs: map[string]string{
				"EVENT_TYPE_PREFIX": eventTypePrefix,
				"REQUEST_TIMEOUT":   "10s",
			},
		},
		{
			name:      "APPLICATION_CRD_ENABLED should be set",
			givenEnvs: map[string]string{},
			givenEventing: testutils.NewEventingCR(
				testutils.WithEventMeshBackend("test-namespace/test-name"),
				testutils.WithEventingEventTypePrefix(eventTypePrefix),
			),
			wantEnvs: map[string]string{
				"APPLICATION_CRD_ENABLED": "false",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.givenEnvs {
				t.Setenv(k, v)
			}
			backendConfig := env.GetBackendConfig()
			envVars := getEventMeshEnvVars("test-name", backendConfig.PublisherConfig, tc.givenEventing)

			// ensure the right envs were set
			for index, val := range tc.wantEnvs {
				gotEnv := test.FindEnvVar(envVars, index)
				assert.NotNil(t, gotEnv)
				assert.Equal(t, val, gotEnv.Value)
			}
		})
	}
}

// natsBackendAssertions checks that the NATS-specific data was set in the NewNATSPublisherDeployment.
func natsBackendAssertions(t *testing.T, publisherName string, deployment appsv1.Deployment) {
	container := findPublisherContainer(publisherName, deployment)
	assert.NotNil(t, container)

	streamName := test.FindEnvVar(container.Env, "JS_STREAM_NAME")
	assert.Equal(t, streamName.Value, "kyma")
	url := test.FindEnvVar(container.Env, "NATS_URL")
	assert.Equal(t, url.Value, natsURL)
	eventTypePrefixEnv := test.FindEnvVar(container.Env, "EVENT_TYPE_PREFIX")
	assert.Equal(t, eventTypePrefixEnv.Value, eventTypePrefix)

	// check the affinity was set
	affinityLabels := deployment.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchLabels
	for _, val := range affinityLabels {
		assert.Equal(t, val, publisherName)
	}
}

// eventMeshBackendAssertions checks that the eventmesh-specific data was set in the NewEventMeshPublisherDeployment.
func eventMeshBackendAssertions(t *testing.T, publisherName string, deployment appsv1.Deployment) {
	container := findPublisherContainer(publisherName, deployment)
	assert.NotNil(t, container)

	// check eventmesh-specific env variables
	eventMeshNamespace := test.FindEnvVar(container.Env, "BEB_NAMESPACE")
	assert.Equal(t, eventMeshNamespace.Value, fmt.Sprintf("%s$(BEB_NAMESPACE_VALUE)", eventMeshNamespacePrefix))

	// check the affinity is empty
	assert.Empty(t, deployment.Spec.Template.Spec.Affinity)
}

// findPublisherContainer gets the publisher proxy container by its name.
func findPublisherContainer(publisherName string, deployment appsv1.Deployment) v1.Container {
	var container v1.Container
	for _, c := range deployment.Spec.Template.Spec.Containers {
		if strings.EqualFold(c.Name, publisherName) {
			container = c
		}
	}
	return container
}
