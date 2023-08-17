package eventing

import (
	"context"
	"github.com/kyma-project/kyma/components/eventing-controller/options"
	"testing"

	"github.com/kyma-project/kyma/components/eventing-controller/logger"

	ctrlmocks "github.com/kyma-project/eventing-manager/internal/controller/eventing/mocks"

	"k8s.io/apimachinery/pkg/runtime"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	managermocks "github.com/kyma-project/eventing-manager/pkg/eventing/mocks"
	k8smocks "github.com/kyma-project/eventing-manager/pkg/k8s/mocks"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockedUnitTestEnvironment provides mocked resources for unit tests.
type MockedUnitTestEnvironment struct {
	Context         context.Context
	Client          client.Client
	kubeClient      *k8smocks.Client
	eventingManager *managermocks.Manager
	ctrlManager     *ctrlmocks.Manager
	Reconciler      *Reconciler
	Logger          *logger.Logger
	Recorder        *record.FakeRecorder
}

func NewMockedUnitTestEnvironment(t *testing.T, objs ...client.Object) *MockedUnitTestEnvironment {
	// setup context
	ctx := context.Background()

	// setup logger
	ctrLogger, err := logger.New("json", "info")
	require.NoError(t, err)

	// setup fake client for k8s
	newScheme := runtime.NewScheme()
	err = natsv1alpha1.AddToScheme(newScheme)
	require.NoError(t, err)
	err = eventingv1alpha1.AddToScheme(newScheme)
	require.NoError(t, err)
	fakeClientBuilder := fake.NewClientBuilder().WithScheme(newScheme)
	// TODO: once controller-runtime version is upgraded to >=0.15.x, use the following.
	//fakeClient := fakeClientBuilder.WithObjects(objs...).WithStatusSubresource(objs...).Build()
	fakeClient := fakeClientBuilder.WithObjects(objs...).Build()
	recorder := &record.FakeRecorder{}

	// setup custom mocks
	kubeClient := new(k8smocks.Client)
	eventingManager := new(managermocks.Manager)
	mockManager := new(ctrlmocks.Manager)

	opts := options.New()

	// setup reconciler
	reconciler := NewReconciler(
		fakeClient,
		kubeClient,
		newScheme,
		ctrLogger,
		recorder,
		eventingManager,
		nil,
		opts,
	)
	reconciler.ctrlManager = mockManager

	return &MockedUnitTestEnvironment{
		Context:         ctx,
		Client:          fakeClient,
		kubeClient:      kubeClient,
		Reconciler:      reconciler,
		Logger:          ctrLogger,
		Recorder:        recorder,
		eventingManager: eventingManager,
		ctrlManager:     mockManager,
	}
}

func (testEnv *MockedUnitTestEnvironment) GetEventing(name, namespace string) (eventingv1alpha1.Eventing, error) {
	var evnt eventingv1alpha1.Eventing
	err := testEnv.Client.Get(testEnv.Context, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &evnt)
	return evnt, err
}
