package testenvironment

import (
	"context"
	"errors"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"strings"
	"time"

	"github.com/kyma-project/eventing-manager/hack/e2e/common"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/eventing"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
	pkghttp "github.com/kyma-project/eventing-manager/hack/e2e/common/http"
	"github.com/kyma-project/eventing-manager/hack/e2e/env"
	ecv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Interval      = 2 * time.Second
	SmallInterval = 200 * time.Millisecond
	Attempts      = 60
	FewAttempts   = 5
	ThreeAttempts = 3
)

// TestEnvironment provides mocked resources for integration tests.
type TestEnvironment struct {
	Context        context.Context
	Logger         *zap.Logger
	K8sClientset   *kubernetes.Clientset
	K8sClient      client.Client
	EventPublisher *eventing.Publisher
	SinkClient     *eventing.SinkClient
	TestConfigs    *env.E2EConfig
}

func NewTestEnvironment() *TestEnvironment {
	var err error
	logger, err := common.SetupLogger()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	testConfigs, err := env.GetE2EConfig()
	if err != nil {
		logger.Error(err.Error())
		panic(err)

	}
	logger.Info(fmt.Sprintf("##### NOTE: Tests will run w.r.t. backend: %s", testConfigs.BackendType))

	clientSet, k8sClient, err := common.GetK8sClients()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	return &TestEnvironment{
		Context:      context.TODO(),
		Logger:       logger,
		K8sClientset: clientSet,
		K8sClient:    k8sClient,
		TestConfigs:  testConfigs,
	}
}

func (te *TestEnvironment) CreateTestNamespace() error {
	return common.Retry(Attempts, Interval, func() error {
		// It's fine if the Namespace already exists.
		return client.IgnoreAlreadyExists(te.K8sClient.Create(te.Context, fixtures.Namespace(te.TestConfigs.TestNamespace)))
	})
}

func (te *TestEnvironment) DeleteTestNamespace() error {
	return common.Retry(FewAttempts, Interval, func() error {
		// It's fine if the Namespace already exists.
		return client.IgnoreAlreadyExists(te.K8sClient.Delete(te.Context, fixtures.Namespace(te.TestConfigs.TestNamespace)))
	})
}

func (te *TestEnvironment) InitEventPublisherClient() error {
	maxIdleConns := 10
	maxConnsPerHost := 10
	maxIdleConnsPerHost := 10
	idleConnTimeout := 1 * time.Minute
	t := pkghttp.NewTransport(maxIdleConns, maxConnsPerHost, maxIdleConnsPerHost, idleConnTimeout)
	clientHTTP := pkghttp.NewHttpClient(t.Clone())
	clientCE, err := pkghttp.NewCloudEventsClient(t.Clone())
	if err != nil {
		return err
	}
	te.EventPublisher = eventing.NewPublisher(context.Background(), *clientCE, clientHTTP, te.TestConfigs.PublisherURL, te.Logger)
	return nil
}

func (te *TestEnvironment) InitSinkClient() {
	maxIdleConns := 10
	maxConnsPerHost := 10
	maxIdleConnsPerHost := 10
	idleConnTimeout := 1 * time.Minute
	t := pkghttp.NewTransport(maxIdleConns, maxConnsPerHost, maxIdleConnsPerHost, idleConnTimeout)
	clientHTTP := pkghttp.NewHttpClient(t.Clone())
	te.SinkClient = eventing.NewSinkClient(context.Background(), clientHTTP, te.TestConfigs.SinkPortForwardedURL, te.Logger)
}

func (te *TestEnvironment) CreateAllSubscriptions() error {
	ctx := context.TODO()
	// create v1alpha1 subscriptions if not exists.
	err := te.CreateV1Alpha1Subscriptions(ctx, fixtures.V1Alpha1SubscriptionsToTest())
	if err != nil {
		return err
	}

	// create v1alpha2 subscriptions if not exists.
	return te.CreateV1Alpha2Subscriptions(ctx, fixtures.V1Alpha2SubscriptionsToTest())
}

func (te *TestEnvironment) DeleteAllSubscriptions() error {
	// delete v1alpha1 subscriptions if not exists.
	for _, subToTest := range fixtures.V1Alpha1SubscriptionsToTest() {
		if err := te.DeleteSubscriptionFromK8s(subToTest.Name, te.TestConfigs.TestNamespace); err != nil {
			return err
		}
	}

	// delete v1alpha2 subscriptions if not exists.
	for _, subToTest := range fixtures.V1Alpha2SubscriptionsToTest() {
		if err := te.DeleteSubscriptionFromK8s(subToTest.Name, te.TestConfigs.TestNamespace); err != nil {
			return err
		}
	}
	return nil
}

func (te *TestEnvironment) WaitForAllSubscriptions() error {
	ctx := context.TODO()
	// wait for v1alpha1 subscriptions to get ready.
	err := te.WaitForSubscriptions(ctx, fixtures.V1Alpha1SubscriptionsToTest())
	if err != nil {
		return err
	}

	// wait for v1alpha2 subscriptions to get ready
	return te.WaitForSubscriptions(ctx, fixtures.V1Alpha1SubscriptionsToTest())
}

func (te *TestEnvironment) CreateV1Alpha1Subscriptions(ctx context.Context, subList []eventing.TestSubscriptionInfo) error {
	for _, subInfo := range subList {
		err := common.Retry(FewAttempts, SmallInterval, func() error {
			newSub := subInfo.ToSubscriptionV1Alpha1(te.TestConfigs.SubscriptionSinkURL, te.TestConfigs.TestNamespace)
			return client.IgnoreAlreadyExists(te.K8sClient.Create(ctx, newSub))
		})
		// return error if all retries are exhausted.
		if err != nil {
			return err
		}
	}
	return nil
}

func (te *TestEnvironment) CreateV1Alpha2Subscriptions(ctx context.Context, subList []eventing.TestSubscriptionInfo) error {
	for _, subInfo := range subList {
		err := common.Retry(FewAttempts, SmallInterval, func() error {
			newSub := subInfo.ToSubscriptionV1Alpha2(te.TestConfigs.SubscriptionSinkURL, te.TestConfigs.TestNamespace)
			return client.IgnoreAlreadyExists(te.K8sClient.Create(ctx, newSub))
		})
		// return error if all retries are exhausted.
		if err != nil {
			return err
		}
	}
	return nil
}

func (te *TestEnvironment) WaitForSubscriptions(ctx context.Context, subsToTest []eventing.TestSubscriptionInfo) error {
	for _, subToTest := range subsToTest {
		return te.WaitForSubscription(ctx, subToTest)
	}
	return nil
}

func (te *TestEnvironment) WaitForSubscription(ctx context.Context, subsToTest eventing.TestSubscriptionInfo) error {
	return common.Retry(Attempts, Interval, func() error {
		// get subscription from cluster.
		gotSub := ecv1alpha2.Subscription{}
		err := te.K8sClient.Get(ctx, k8stypes.NamespacedName{
			Name:      subsToTest.Name,
			Namespace: te.TestConfigs.TestNamespace,
		}, &gotSub)
		if err != nil {
			te.Logger.Debug(fmt.Sprintf("failed to check readiness; failed to fetch subscription: %s "+
				"in namespace: %s", subsToTest.Name, te.TestConfigs.TestNamespace))
			return err
		}

		// check if subscription is reconciled by correct backend.
		if !te.IsSubscriptionReconcileByBackend(gotSub, te.TestConfigs.BackendType) {
			errMsg := fmt.Sprintf("waiting subscription: %s "+
				"in namespace: %s to get recocniled by backend: %s", subsToTest.Name, te.TestConfigs.TestNamespace,
				te.TestConfigs.BackendType)
			te.Logger.Debug(errMsg)
			return errors.New(errMsg)
		}

		// check if subscription is ready.
		if !gotSub.Status.Ready {
			errMsg := fmt.Sprintf("waiting subscription: %s "+
				"in namespace: %s to get ready", subsToTest.Name, te.TestConfigs.TestNamespace)
			te.Logger.Debug(errMsg)
			return errors.New(errMsg)
		}
		return nil
	})
}

func (te *TestEnvironment) IsSubscriptionReconcileByBackend(sub ecv1alpha2.Subscription, activeBackend string) bool {
	condition := sub.Status.FindCondition(ecv1alpha2.ConditionSubscriptionActive)
	if condition == nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(condition.Reason)), strings.ToLower(activeBackend))
}

func (te *TestEnvironment) SetupSink() error {
	if err := te.CreateSinkDeployment(te.TestConfigs.SubscriptionSinkName, te.TestConfigs.TestNamespace,
		te.TestConfigs.SubscriptionSinkImage); err != nil {
		return err
	}

	if err := te.CreateSinkService(te.TestConfigs.SubscriptionSinkName, te.TestConfigs.TestNamespace); err != nil {
		return err
	}

	return te.WaitForDeploymentReady(te.TestConfigs.SubscriptionSinkName, te.TestConfigs.TestNamespace,
		te.TestConfigs.SubscriptionSinkImage)
}

func (te *TestEnvironment) DeleteSinkResources() error {
	if err := te.DeleteDeployment(te.TestConfigs.SubscriptionSinkName, te.TestConfigs.TestNamespace); err != nil {
		return err
	}
	return te.DeleteService(te.TestConfigs.SubscriptionSinkName, te.TestConfigs.TestNamespace)
}

func (te *TestEnvironment) CreateSinkDeployment(name, namespace, image string) error {
	return common.Retry(FewAttempts, Interval, func() error {
		return te.K8sClient.Patch(te.Context, fixtures.NewSinkDeployment(name, namespace, image),
			client.Apply,
			&client.PatchOptions{
				Force:        pointer.Bool(true),
				FieldManager: fixtures.FieldManager,
			})
	})
}

func (te *TestEnvironment) CreateSinkService(name, namespace string) error {
	return common.Retry(FewAttempts, Interval, func() error {
		return te.K8sClient.Patch(te.Context, fixtures.NewSinkService(name, namespace),
			client.Apply,
			&client.PatchOptions{
				Force:        pointer.Bool(true),
				FieldManager: fixtures.FieldManager,
			})
	})
}

func (te *TestEnvironment) DeleteDeployment(name, namespace string) error {
	return common.Retry(FewAttempts, Interval, func() error {
		return te.K8sClient.Delete(te.Context, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		})
	})
}

func (te *TestEnvironment) DeleteService(name, namespace string) error {
	return common.Retry(FewAttempts, Interval, func() error {
		return te.K8sClient.Delete(te.Context, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		})
	})
}

func (te *TestEnvironment) GetDeploymentFromK8s(name, namespace string) (*appsv1.Deployment, error) {
	return te.K8sClientset.AppsV1().Deployments(namespace).Get(te.Context, name, metav1.GetOptions{})
}

func (te *TestEnvironment) WaitForDeploymentReady(name, namespace, image string) error {
	// RetryGet the Eventing Manager and test status.
	return common.Retry(Attempts, Interval, func() error {
		te.Logger.Debug(fmt.Sprintf("waiting for deployment: %s to get ready with image: %s", name, image))
		// Get the deployment from the cluster.
		gotDeployment, err := common.RetryGet(FewAttempts, SmallInterval, func() (*appsv1.Deployment, error) {
			return te.GetDeploymentFromK8s(name, namespace)
		})
		if err != nil {
			return err
		}

		// if image is provided, then check if the deployment has correct image.
		if image != "" && gotDeployment.Spec.Template.Spec.Containers[0].Image != image {
			err = fmt.Errorf("expected deployment (%s) image to be: %s, but found: %s", name, image,
				gotDeployment.Spec.Template.Spec.Containers[0].Image,
			)
			te.Logger.Debug(err.Error())
			return err
		}

		// check if the deployment is ready.
		if *gotDeployment.Spec.Replicas != gotDeployment.Status.UpdatedReplicas ||
			*gotDeployment.Spec.Replicas != gotDeployment.Status.ReadyReplicas ||
			*gotDeployment.Spec.Replicas != gotDeployment.Status.AvailableReplicas {
			err = fmt.Errorf("waiting for deployment: %s to get ready", name)
			te.Logger.Debug(err.Error())
			return err
		}

		// Everything is fine.
		te.Logger.Debug(fmt.Sprintf("deployment: %s is ready with image: %s", name, image))
		return nil
	})
}

func (te *TestEnvironment) DeleteSubscriptionFromK8s(name, namespace string) error {
	// define subscription to delete.
	sub := &ecv1alpha2.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	// delete with retries.
	return common.Retry(FewAttempts, Interval, func() error {
		// delete subscription from cluster.
		err := te.K8sClient.Delete(te.Context, sub)
		if err != nil && !k8serrors.IsNotFound(err) {
			te.Logger.Debug(fmt.Sprintf("failed to delete subscription: %s "+
				"in namespace: %s", name, te.TestConfigs.TestNamespace))
			return err
		}
		return nil
	})
}

func (te *TestEnvironment) TestDeliveryOfLegacyEventForSubV1Alpha1(eventType string) error {
	// define the event
	eventID, eventSource, legacyEventType, payload := eventing.NewLegacyEventForV1Alpha1(eventType, te.TestConfigs.EventTypePrefix)

	// publish the event
	if err := te.EventPublisher.SendLegacyEventWithRetries(eventSource, legacyEventType, payload, FewAttempts, Interval); err != nil {
		te.Logger.Debug(err.Error())
		return err
	}

	// verify if the event was received by the sink.
	te.Logger.Debug(eventID)
	// TODO: implement me!

	return nil
}

func (te *TestEnvironment) TestDeliveryOfLegacyEvent(eventSource, eventType string) error {
	// define the event
	eventID, eventSource, legacyEventType, payload := eventing.NewLegacyEvent(eventSource, eventType)

	// publish the event
	if err := te.EventPublisher.SendLegacyEventWithRetries(eventSource, legacyEventType, payload, FewAttempts, Interval); err != nil {
		te.Logger.Debug(err.Error())
		return err
	}

	// verify if the event was received by the sink.
	te.Logger.Debug(eventID)
	// TODO: implement me!

	return nil
}

func (te *TestEnvironment) TestDeliveryOfCloudEvent(eventSource, eventType string, encoding binding.Encoding) error {
	// define the event
	ceEvent, err := eventing.NewCloudEvent(eventSource, eventType, encoding)
	if err != nil {
		return err
	}

	// publish the event
	if err := te.EventPublisher.SendCloudEventWithRetries(ceEvent, encoding, FewAttempts, Interval); err != nil {
		te.Logger.Debug(err.Error())
		return err
	}

	// verify if the event was received by the sink.
	te.Logger.Debug(fmt.Sprintf("Verifying if CloudEvent (ID: %s) was received by the sink", ceEvent.ID()))
	//return te.VerifyCloudEventReceivedBySink(*ceEvent)
	return nil
}

func (te *TestEnvironment) VerifyCloudEventReceivedBySink(expectedEvent cloudevents.Event) error {
	// define the event
	gotSinkEvent, err := te.SinkClient.GetEventFromSinkWithRetries(expectedEvent.ID(), Attempts, Interval)
	if err != nil {
		te.Logger.Debug(err.Error())
		return err
	}

	// verify if the event was received by the sink.
	te.Logger.Debug(fmt.Sprintf("Got event (ID: %s) from sink, checking if the encoding and payload is correct", gotSinkEvent.ID()))
	return te.CompareCloudEvents(expectedEvent, gotSinkEvent.Event)
}

func (te *TestEnvironment) CompareCloudEvents(expectedEvent cloudevents.Event, gotEvent cloudevents.Event) error {
	if expectedEvent.ID() != gotEvent.ID() {
		return fmt.Errorf("expected event ID: %s, got event ID: %s", expectedEvent.ID(), gotEvent.ID())
	}

	if expectedEvent.Source() != gotEvent.Source() {
		return fmt.Errorf("expected event Source: %s, got event Source: %s", expectedEvent.Source(), gotEvent.Source())
	}

	if expectedEvent.Type() != gotEvent.Type() {
		return fmt.Errorf("expected event Type: %s, got event Type: %s", expectedEvent.Type(), gotEvent.Type())
	}

	if string(expectedEvent.Data()) != string(gotEvent.Data()) {
		return fmt.Errorf("expected event Type: %s, got event Type: %s", expectedEvent.Type(), gotEvent.Type())
	}

	if err := gotEvent.Validate(); err != nil {
		return fmt.Errorf("expected valid cloud event, but got invalid cloud event. Error: %s", err.Error())
	}

	return nil
}