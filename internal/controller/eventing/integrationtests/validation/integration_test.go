package validation_test

import (
	"github.com/kyma-project/eventing-manager/testutils"
	"github.com/kyma-project/eventing-manager/testutils/integration"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"testing"
)

const projectRootDir = "../../../../../"

const noError = ""

const (
	kind               = "kind"
	kindEventing       = "Eventing"
	apiVersion         = "apiVersion"
	apiVersionEventing = "operator.kyma-project.io/v1alpha1"
	metadata           = "metadata"
	name               = "name"
	namespace          = "namespace"

	spec                  = "spec"
	backends              = "backends"
	backendType           = "type"
	typeEventMesh         = "EventMesh"
	typeNats              = "NATS"
	config                = "config"
	natsStreamStorageType = "natsStreamStorageType"
	storageTypeFile       = "File"
	storageTypeMemory     = "Memory"
	eventMeshSecret       = "eventMeshSecret"
	someSecret            = "namespace/name"
	publisher             = "publisher"
	replicas              = "replicas"
	max                   = "max"
	min                   = "min"
	logging               = "logging"
	logLevel              = "logLevel"
	logLevelInfo          = "Info"
	logLevelWarn          = "Warn"
	logLevelError         = "Error"
	logLevelDebug         = "Debug"
	gibberish             = "Gibberish"
)

var testEnvironment *integration.TestEnvironment

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = integration.NewTestEnvironment(projectRootDir, true)
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
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
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
			name: `validation of spec.publisher.replicas.min passes for values <= max`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
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
			name: `validation of spec.backends.config.eventMeshSecret fails when empty if spec.type = EventMesh`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						backends: map[string]any{
							backendType: typeEventMesh,
							config:      map[string]any{},
						},
					},
				},
			},
			wantErrMsg: "secret cannot be empty if EventMesh backend is used",
		},
		{
			name: `validation of spec.backends.config.eventMeshSecret passes when set if spec.type = EventMesh`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						backends: map[string]any{
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
			name: `validation of spec.backends.config.eventMeshSecret passes when empty if spec.type = NATS`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						backends: map[string]any{
							backendType: typeNats,
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backends.type fails for values other than NATS or EventMesh`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						backends: map[string]any{
							backendType: gibberish,
						},
					},
				},
			},
			wantErrMsg: "storage type can only be set to NATS or EventMesh",
		},
		{
			name: `validation of spec.backends.type passes for value = NATS`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						backends: map[string]any{
							backendType: typeNats,
						},
					},
				},
			},
		},
		{
			name: `validation of spec.backends.type passes for value = EventMesh`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						backends: map[string]any{
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
			name: `validation of spec.backends.config.natsStreamStorageType fails for values other than File or Memory`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						backends: map[string]any{
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
			name: `validation of spec.backends.config.natsStreamStorageType passes for value = File`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						backends: map[string]any{
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
			name: `validation of spec.backends.config.natsStreamStorageType passes for value = Memory`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						backends: map[string]any{
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
			name: `validation of spec.logging fails for values other than Debug, Info, Warn or Error`,
			givenUnstructuredEventing: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindEventing,
					apiVersion: apiVersionEventing,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
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
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
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
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
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
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
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
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
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
					" Check the validation rule of the eventing CR where the above error is returned.")
			} else {
				if err != nil {
					require.Contains(t, err.Error(), tc.wantErrMsg, "Expected a specific error message."+
						" but message does not match. Check the validation rules of the eventing CR.")
				} else {
					require.Error(t, err, "Expected the following error message: \""+tc.wantErrMsg+
						".\" but got no error. Check the validation rules of the eventing CR.")
				}

			}
		})
	}
}
