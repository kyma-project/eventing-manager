package validation_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/test"
	eventingMatchers "github.com/kyma-project/eventing-manager/test/matchers"
	"github.com/kyma-project/eventing-manager/test/utils/integration"
)

const projectRootDir = "../../../../../../"

const noError = ""

const (
	kind               = "kind"
	kindEventing       = "Eventing"
	apiVersion         = "apiVersion"
	apiVersionEventing = "operator.kyma-project.io/v1alpha1"
	metadata           = "metadata"
	name               = "name"
	namespace          = "namespace"

	spec                   = "spec"
	backend                = "backend"
	backendType            = "type"
	typeEventMesh          = "EventMesh"
	typeNats               = "NATS"
	config                 = "config"
	natsStreamStorageType  = "natsStreamStorageType"
	storageTypeFile        = "File"
	storageTypeMemory      = "Memory"
	natsStreamReplicas     = "natsStreamReplicas"
	streamReplicas         = 3
	natsStreamMaxSize      = "natsStreamMaxSize"
	maxSize                = "700Mi"
	natsMaxMsgsPerTopic    = "natsMaxMsgsPerTopic"
	msgsPerTopic           = 1000000
	eventTypePrefix        = "eventTypePrefix"
	eventMeshSecret        = "eventMeshSecret"
	domain                 = "domain"
	someSecret             = "namespace/name"
	wrongSecret            = "gibberish"
	publisher              = "publisher"
	replicas               = "replicas"
	max                    = "max"
	min                    = "min"
	resources              = "resources"
	limits                 = "limits"
	requests               = "requests"
	cpu                    = "cpu"
	memory                 = "memory"
	limitsCpuValue         = "500m"
	limitsMemoryValue      = "512Mi"
	requestsCpuValue       = "10m"
	requestsMemoryValue    = "256Mi"
	logging                = "logging"
	logLevel               = "logLevel"
	logLevelInfo           = "Info"
	logLevelWarn           = "Warn"
	logLevelError          = "Error"
	logLevelDebug          = "Debug"
	gibberish              = "Gibberish"
	defaultEventTypePrefix = "sap.kyma.custom"
)

var testEnvironment *integration.TestEnvironment

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = integration.NewTestEnvironment(integration.TestEnvironmentConfig{
		ProjectRootDir:            projectRootDir,
		CELValidationEnabled:      true,
		APIRuleCRDEnabled:         true,
		ApplicationRuleCRDEnabled: true,
		NATSCRDEnabled:            true,
		AllowedEventingCR:         nil,
	})
	if err != nil {
		panic(err)
	}

	// run tests
	code := m.Run()

	// tear down test env
	if err = testEnvironment.TearDown(); err != nil {
		panic(err)
	}

	os.Exit(code)
}

// Test_Validate_CreateEventing creates an eventing CR with correct and purposefully incorrect values, and compares
// the error that was caused by this against a wantErrMsg to test the eventing CR validation rules.
func Test_Validate_CreateEventing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                      string
		givenUnstructuredEventing unstructured.Unstructured
		wantErrMsg                string
	}{
		{
			name: `validation of spec.publisher.replicas.min fails for values > max`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						publisher: map[string]any{
							replicas: map[string]any{
								min: 3,
								max: 2,
							},
						},
					},
				},
			},
			wantErrMsg: "min value must be smaller than the max value",
		},
		{
			name: `validation of spec.publisher.replicas.min passes for values = max`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						publisher: map[string]any{
							replicas: map[string]any{
								min: 2,
								max: 2,
							},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.publisher.replicas.min passes for values < max`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						publisher: map[string]any{
							replicas: map[string]any{
								min: 1,
								max: 2,
							},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backend.config.eventMeshSecret fails when empty if spec.type = EventMesh`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeEventMesh,
							config:      map[string]any{},
						},
					},
				},
			},
			wantErrMsg: "secret cannot be empty if EventMesh backend is used",
		},
		{
			name: `validation of spec.backend.config.eventMeshSecret fails if it does not match the format namespace/name`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeEventMesh,
							config: map[string]any{
								eventMeshSecret: wrongSecret,
							},
						},
					},
				},
			},
			wantErrMsg: "spec.backend.config.eventMeshSecret in body should match '^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$'",
		},
		{
			name: `validation of spec.backend.config.eventMeshSecret passes if it matches the format namespace/name`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeEventMesh,
							config: map[string]any{
								eventMeshSecret: someSecret,
							},
						},
					},
				},
			},
		},

		// validate the spec.backend.config.domain
		{
			name: `validation of spec.backend.config.domain passes if domain is nil`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							config: map[string]any{},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backend.config.domain passes if domain is empty`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							config: map[string]any{
								domain: "",
							},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backend.config.domain passes if domain is valid`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							config: map[string]any{
								domain: "domain.com",
							},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backend.config.domain passes if domain is valid`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							config: map[string]any{
								domain: "domain.part1.part2.part3.part4",
							},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backend.config.domain fails if domain is invalid`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							config: map[string]any{
								domain: " ",
							},
						},
					},
				},
			},
			wantErrMsg: `spec.backend.config.domain: Invalid value: " ": spec.backend.config.domain in body should match '^(?:([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*)?$'`,
		},
		{
			name: `validation of spec.backend.config.domain fails if domain is invalid`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							config: map[string]any{
								domain: "http://domain.com",
							},
						},
					},
				},
			},
			wantErrMsg: `spec.backend.config.domain: Invalid value: "http://domain.com": spec.backend.config.domain in body should match '^(?:([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*)?$'`,
		},
		{
			name: `validation of spec.backend.config.domain fails if domain is invalid`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							config: map[string]any{
								domain: "https://domain.com",
							},
						},
					},
				},
			},
			wantErrMsg: `spec.backend.config.domain: Invalid value: "https://domain.com": spec.backend.config.domain in body should match '^(?:([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*)?$'`,
		},
		{
			name: `validation of spec.backend.config.domain fails if domain is invalid`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							config: map[string]any{
								domain: "domain.com:8080",
							},
						},
					},
				},
			},
			wantErrMsg: `spec.backend.config.domain: Invalid value: "domain.com:8080": spec.backend.config.domain in body should match '^(?:([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*)?$'`,
		},
		{
			name: `validation of spec.backend.config.domain fails if domain is invalid`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							config: map[string]any{
								domain: "domain.com/endpoint",
							},
						},
					},
				},
			},
			wantErrMsg: `spec.backend.config.domain: Invalid value: "domain.com/endpoint": spec.backend.config.domain in body should match '^(?:([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*)?$'`,
		},

		{
			name: `validation of spec.backend.type fails for values other than NATS or EventMesh`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: gibberish,
						},
					},
				},
			},
			wantErrMsg: "backend type can only be set to NATS or EventMesh",
		},
		{
			name: `validation of spec.backend.type passes for value = NATS and spec.backend.config.eventMeshSecret passes when empty`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeNats,
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backend.type and spec.backend.config.eventMeshSecret passes for value = EventMesh`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeEventMesh,
							config: map[string]any{
								eventMeshSecret: someSecret,
							},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backend.config.natsStreamStorageType fails for values other than File or Memory`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeNats,
							config: map[string]any{
								natsStreamStorageType: gibberish,
							},
						},
					},
				},
			},
			wantErrMsg: "storage type can only be set to File or Memory",
		},
		{
			name: `validation of spec.backend.config.natsStreamStorageType passes for value = File`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeNats,
							config: map[string]any{
								natsStreamStorageType: storageTypeFile,
							},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backend.config.natsStreamStorageType passes for value = Memory`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeNats,
							config: map[string]any{
								natsStreamStorageType: storageTypeMemory,
							},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backend.config.eventTypePrefix fails for empty string`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeNats,
							config: map[string]any{
								eventTypePrefix: "",
							},
						},
					},
				},
			},
			wantErrMsg: "eventTypePrefix cannot be empty",
		},
		{
			name: `validation of spec.backend.config.eventTypePrefix should pass for valid value`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeNats,
							config: map[string]any{
								eventTypePrefix: "mock.test.prefix",
							},
						},
					},
				},
			},
		},
		{
			name: `validation of spec.logging fails for values other than Debug, Info, Warn or Error`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						logging: map[string]any{
							logLevel: gibberish,
						},
					},
				},
			},
			wantErrMsg: "logLevel can only be set to Debug, Info, Warn or Error",
		},
		{
			name: `validation of spec.logging passes for value = Debug`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						logging: map[string]any{
							logLevel: logLevelDebug,
						},
					},
				},
			},
		},
		{
			name: `validation of spec.logging passes for value = Info`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						logging: map[string]any{
							logLevel: logLevelInfo,
						},
					},
				},
			},
		},
		{
			name: `validation of spec.logging passes for value = Warn`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						logging: map[string]any{
							logLevel: logLevelWarn,
						},
					},
				},
			},
		},
		{
			name: `validation of spec.logging passes for value = Error`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						logging: map[string]any{
							logLevel: logLevelError,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnvironment.EnsureNamespaceCreation(t, tc.givenUnstructuredEventing.GetNamespace())

			// when
			err := testEnvironment.CreateUnstructuredK8sResource(&tc.givenUnstructuredEventing)

			// then
			if tc.wantErrMsg == noError {
				require.NoError(t, err, "Expected error message to be empty but got error instead."+
					" Check the validation rule of the eventing CR.")
			} else {
				require.Error(t, err, fmt.Sprintf("Expected the following error message: \n \" %s \" \n"+
					" but got no error. Check the validation rules of the eventing CR.", tc.wantErrMsg))

				require.Contains(t, err.Error(), tc.wantErrMsg, "Expected a specific error message"+
					" but messages do not match. Check the validation rules of the eventing CR.")
			}
		})
	}
}

// Test_Validate_CreateEventing creates an eventing CR with correct and purposefully incorrect values, and compares
// the error that was caused by this against a wantErrMsg to test the eventing CR validation rules.
func Test_Validate_Defaulting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                      string
		givenUnstructuredEventing unstructured.Unstructured
		wantMatches               gomegatypes.GomegaMatcher
	}{
		{
			name: "defaulting with bare minimum eventing",
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
				},
			},
			wantMatches: gomega.And(
				eventingMatchers.HaveBackendTypeNats(defaultBackendConfig()),
				eventingMatchers.HavePublisher(defaultPublisher()),
				eventingMatchers.HavePublisherResources(defaultPublisherResources()),
				eventingMatchers.HaveLogging(defaultLogging()),
			),
		},
		{
			name: "defaulting with an empty spec",
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{},
				},
			},
			wantMatches: gomega.And(
				eventingMatchers.HaveBackendTypeNats(defaultBackendConfig()),
				eventingMatchers.HavePublisher(defaultPublisher()),
				eventingMatchers.HavePublisherResources(defaultPublisherResources()),
				eventingMatchers.HaveLogging(defaultLogging()),
			),
		},
		{
			name: "defaulting with an empty spec.backend.config",
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeNats,
							config:      map[string]any{
								// empty, to be filled by defaulting
							},
							publisher: map[string]any{
								replicas: map[string]any{
									min: 2,
									max: 2,
								},
								resources: map[string]any{
									limits: map[string]any{
										cpu:    limitsCpuValue,
										memory: limitsMemoryValue,
									},
									requests: map[string]any{
										cpu:    requestsCpuValue,
										memory: requestsMemoryValue,
									},
								},
							},
							logging: map[string]any{
								logLevel: logLevelInfo,
							},
						},
					},
				},
			},
			wantMatches: gomega.And(
				eventingMatchers.HaveBackendTypeNats(defaultBackendConfig()),
			),
		},
		{
			name: "defaulting with an empty spec.publisher",
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeNats,
							config: map[string]any{
								natsStreamStorageType: storageTypeFile,
								natsStreamReplicas:    streamReplicas,
								natsStreamMaxSize:     maxSize,
								natsMaxMsgsPerTopic:   msgsPerTopic,
							},
							publisher: map[string]any{
								// empty, to be filled by defaulting
							},
							logging: map[string]any{
								logLevel: logLevelInfo,
							},
						},
					},
				},
			},
			wantMatches: gomega.And(
				eventingMatchers.HavePublisher(defaultPublisher()),
				eventingMatchers.HavePublisherResources(defaultPublisherResources()),
			),
		},
		{
			name: "defaulting with an empty spec.logging",
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      test.GetRandK8sName(7),
						namespace: test.GetRandK8sName(7),
					},
					spec: map[string]any{
						backend: map[string]any{
							backendType: typeNats,
							config: map[string]any{
								natsStreamStorageType: storageTypeFile,
								natsStreamReplicas:    streamReplicas,
								natsStreamMaxSize:     maxSize,
								natsMaxMsgsPerTopic:   msgsPerTopic,
							},
							publisher: map[string]any{
								replicas: map[string]any{
									min: 2,
									max: 2,
								},
								resources: map[string]any{
									limits: map[string]any{
										cpu:    limitsCpuValue,
										memory: limitsMemoryValue,
									},
									requests: map[string]any{
										cpu:    requestsCpuValue,
										memory: requestsMemoryValue,
									},
								},
							},
							logging: map[string]any{
								// empty, to be filled by defaulting
							},
						},
					},
				},
			},
			wantMatches: gomega.And(
				eventingMatchers.HaveLogging(defaultLogging()),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewGomegaWithT(t)

			// given
			testEnvironment.EnsureNamespaceCreation(t, tc.givenUnstructuredEventing.GetNamespace())

			// when
			testEnvironment.EnsureK8sUnStructResourceCreated(t, &tc.givenUnstructuredEventing)

			// then
			testEnvironment.GetEventingAssert(g, &v1alpha1.Eventing{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      tc.givenUnstructuredEventing.GetName(),
					Namespace: tc.givenUnstructuredEventing.GetNamespace(),
				},
			}).Should(tc.wantMatches)
		})
	}
}

func defaultBackendConfig() v1alpha1.BackendConfig {
	return v1alpha1.BackendConfig{
		NATSStreamStorageType: storageTypeFile,
		NATSStreamReplicas:    3,
		NATSStreamMaxSize:     resource.MustParse("700Mi"),
		NATSMaxMsgsPerTopic:   1000000,
		EventTypePrefix:       defaultEventTypePrefix,
	}
}

func defaultPublisher() v1alpha1.Publisher {
	return v1alpha1.Publisher{
		Replicas: v1alpha1.Replicas{
			Min: 2,
			Max: 2,
		},
	}
}

func defaultPublisherResources() kcorev1.ResourceRequirements {
	return kcorev1.ResourceRequirements{
		Limits: kcorev1.ResourceList{
			"cpu":    resource.MustParse("500m"),
			"memory": resource.MustParse("512Mi"),
		},
		Requests: kcorev1.ResourceList{
			"cpu":    resource.MustParse("40m"),
			"memory": resource.MustParse("256Mi"),
		},
	}
}

func defaultLogging() v1alpha1.Logging {
	return v1alpha1.Logging{LogLevel: logLevelInfo}
}
