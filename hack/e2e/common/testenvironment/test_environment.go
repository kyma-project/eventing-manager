package testenvironment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	istio "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"

	ecv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/eventing-manager/hack/e2e/common"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/eventing"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
	pkghttp "github.com/kyma-project/eventing-manager/hack/e2e/common/http"
	"github.com/kyma-project/eventing-manager/hack/e2e/env"
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
	Context          context.Context
	Logger           *zap.Logger
	K8sClientset     *kubernetes.Clientset
	K8sClient        client.Client
	K8sDynamicClient *dynamic.DynamicClient
	EventPublisher   *eventing.Publisher
	SinkClient       *eventing.SinkClient
	TestConfigs      *env.E2EConfig
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

	clientSet, k8sClient, k8sDynamicClient, err := common.GetK8sClients()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	return &TestEnvironment{
		Context:          context.TODO(),
		Logger:           logger,
		K8sClientset:     clientSet,
		K8sClient:        k8sClient,
		K8sDynamicClient: k8sDynamicClient,
		TestConfigs:      testConfigs,
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
		// It's fine if the Namespace does not exist.
		return client.IgnoreNotFound(te.K8sClient.Delete(te.Context, fixtures.Namespace(te.TestConfigs.TestNamespace)))
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
	// Create v1alpha1 subscriptions if not exists.
	err := te.CreateV1Alpha1Subscriptions(ctx, fixtures.V1Alpha1SubscriptionsToTest())
	if err != nil {
		return err
	}

	// Create v1alpha2 subscriptions if not exists.
	return te.CreateV1Alpha2Subscriptions(ctx, fixtures.V1Alpha2SubscriptionsToTest())
}

func (te *TestEnvironment) DeleteAllSubscriptions() error {
	// delete v1alpha1 subscriptions if not exists.
	for _, subToTest := range fixtures.V1Alpha1SubscriptionsToTest() {
		if err := te.DeleteSubscriptionFromK8s(subToTest.Name, te.TestConfigs.TestNamespace); err != nil {
			return err
		}
	}

	// Delete v1alpha2 subscriptions if not exists.
	for _, subToTest := range fixtures.V1Alpha2SubscriptionsToTest() {
		if err := te.DeleteSubscriptionFromK8s(subToTest.Name, te.TestConfigs.TestNamespace); err != nil {
			return err
		}
	}
	return nil
}

func (te *TestEnvironment) WaitForAllSubscriptions() error {
	ctx := context.TODO()
	// Wait for v1alpha1 subscriptions to get ready.
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
		// Return error if all retries are exhausted.
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
		return client.IgnoreNotFound(err)
	}
	return client.IgnoreNotFound(te.DeleteService(te.TestConfigs.SubscriptionSinkName, te.TestConfigs.TestNamespace))
}

func (te *TestEnvironment) CreateSinkDeployment(name, namespace, image string) error {
	return common.Retry(FewAttempts, Interval, func() error {
		return te.K8sClient.Patch(te.Context, fixtures.NewSinkDeployment(name, namespace, image),
			client.Apply,
			&client.PatchOptions{
				Force:        ptr.To(true),
				FieldManager: fixtures.FieldManager,
			})
	})
}

func (te *TestEnvironment) CreateSinkService(name, namespace string) error {
	return common.Retry(FewAttempts, Interval, func() error {
		return te.K8sClient.Patch(te.Context, fixtures.NewSinkService(name, namespace),
			client.Apply,
			&client.PatchOptions{
				Force:        ptr.To(true),
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

func (te *TestEnvironment) TestDeliveryOfLegacyEvent(eventSource, eventType string, subCRVersion fixtures.SubscriptionCRVersion) error {
	// define the event
	var eventId, legacyEventSource, legacyEventType, payload string
	if subCRVersion == fixtures.V1Alpha1SubscriptionCRVersion {
		eventId, legacyEventSource, legacyEventType, payload = eventing.NewLegacyEventForV1Alpha1(eventType, te.TestConfigs.EventTypePrefix)
	} else {
		eventId, legacyEventSource, legacyEventType, payload = eventing.NewLegacyEvent(eventSource, eventType)
	}

	// publish the event
	if err := te.EventPublisher.SendLegacyEventWithRetries(legacyEventSource, legacyEventType, payload, FewAttempts, Interval); err != nil {
		te.Logger.Debug(err.Error())
		return err
	}

	// verify if the event was received by the sink.
	te.Logger.Debug(fmt.Sprintf("Verifying if LegacyEvent (ID: %s) was received by the sink", eventId))
	return te.VerifyLegacyEventReceivedBySink(eventId, eventType, eventSource, payload)
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
	return te.VerifyCloudEventReceivedBySink(*ceEvent)
}

func (te *TestEnvironment) VerifyLegacyEventReceivedBySink(eventId, eventType, eventSource, payload string) error {
	// publisher-proxy converts LegacyEvent to CloudEvent, so the sink should have received a CloudEvent.
	// extract data from payload of legacy event.
	result := make(map[string]interface{})
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return err
	}
	data := result["data"]

	// define the expected CloudEvent.
	expectedCEEvent := cloudevents.NewEvent()
	expectedCEEvent.SetID(eventId)
	expectedCEEvent.SetType(eventType)
	expectedCEEvent.SetSource(eventSource)
	if err := expectedCEEvent.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return err
	}

	// verify if the event was received.
	return te.VerifyCloudEventReceivedBySink(expectedCEEvent)
}

func (te *TestEnvironment) VerifyCloudEventReceivedBySink(expectedEvent cloudevents.Event) error {
	// define the event
	gotSinkEvent, err := te.SinkClient.GetEventFromSinkWithRetries(expectedEvent.ID(), Attempts, Interval)
	if err != nil {
		te.Logger.Debug(err.Error())
		return err
	}

	// verify if the event was received by the sink.
	te.Logger.Debug(fmt.Sprintf("Got event (ID: %s) from sink, checking if the payload is correct", gotSinkEvent.ID()))
	return te.CompareCloudEvents(expectedEvent, gotSinkEvent.Event)
}

func (te *TestEnvironment) CompareCloudEvents(expectedEvent cloudevents.Event, gotEvent cloudevents.Event) error {
	var resultError error
	// check if its a valid CloudEvent.
	if err := gotEvent.Validate(); err != nil {
		msg := fmt.Sprintf("expected valid cloud event, but got invalid cloud event. Error: %s", err.Error())
		resultError = fixtures.AppendMsgToError(resultError, msg)
	}

	if expectedEvent.ID() != gotEvent.ID() {
		msg := fmt.Sprintf("expected event ID: %s, got event ID: %s", expectedEvent.ID(), gotEvent.ID())
		resultError = fixtures.AppendMsgToError(resultError, msg)
	}

	if string(expectedEvent.Data()) != string(gotEvent.Data()) {
		msg := fmt.Sprintf("expected event data: %s, got event data: %s",
			string(expectedEvent.Data()), string(gotEvent.Data()))
		resultError = fixtures.AppendMsgToError(resultError, msg)
	}

	// if it is a v1alpha1 Subscription event, then we do not check further.
	if strings.HasPrefix(gotEvent.Type(), te.TestConfigs.EventTypePrefix) {
		return resultError
	}

	// check in detail further the source and type.
	if expectedEvent.Source() != gotEvent.Source() {
		msg := fmt.Sprintf("expected event Source: %s, got event Source: %s", expectedEvent.Source(), gotEvent.Source())
		resultError = fixtures.AppendMsgToError(resultError, msg)
	}

	if expectedEvent.Type() != gotEvent.Type() {
		msg := fmt.Sprintf("expected event Type: %s, got event Type: %s", expectedEvent.Type(), gotEvent.Type())
		resultError = fixtures.AppendMsgToError(resultError, msg)
	}

	originalType, ok := gotEvent.Extensions()[fixtures.EventOriginalTypeHeader]
	if !ok {
		msg := fmt.Sprintf("expected event to have header: %s, but its missing", fixtures.EventOriginalTypeHeader)
		resultError = fixtures.AppendMsgToError(resultError, msg)
	}
	if expectedEvent.Type() != originalType {
		msg := fmt.Sprintf("expected originaltype header to have value: %s, but got: %s", expectedEvent.Type(), originalType)
		resultError = fixtures.AppendMsgToError(resultError, msg)
	}

	return resultError
}

func (te *TestEnvironment) SetupEventingCR() error {
	return common.Retry(Attempts, Interval, func() error {
		ctx := context.TODO()
		eventingCR := fixtures.EventingCR(eventingv1alpha1.BackendType(te.TestConfigs.BackendType))
		errEvnt := te.K8sClient.Create(ctx, eventingCR)
		if k8serrors.IsAlreadyExists(errEvnt) {
			gotEventingCR, getErr := te.GetEventingCRFromK8s(eventingCR.Name, eventingCR.Namespace)
			if getErr != nil {
				return getErr
			}

			// If Backend type is changed then update the CR.
			if gotEventingCR.Spec.Backend.Type != eventingCR.Spec.Backend.Type {
				eventingCR.ObjectMeta = gotEventingCR.ObjectMeta
				if errEvnt = te.K8sClient.Update(ctx, eventingCR); errEvnt != nil {
					return errEvnt
				}
			} else {
				te.Logger.Warn(
					"error while creating Eventing CR, resource already exist; test will continue with existing CR",
				)
			}
			return nil
		}
		return errEvnt
	})
}

func (te *TestEnvironment) DeleteEventingCR() error {
	return common.Retry(Attempts, Interval, func() error {
		return client.IgnoreNotFound(te.K8sClient.Delete(te.Context,
			fixtures.EventingCR(eventingv1alpha1.BackendType(te.TestConfigs.BackendType))))
	})
}

func (te *TestEnvironment) GetEventingCRFromK8s(name, namespace string) (*eventingv1alpha1.Eventing, error) {
	var eventingCR eventingv1alpha1.Eventing
	err := te.K8sClient.Get(te.Context, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &eventingCR)
	return &eventingCR, err
}

func (te *TestEnvironment) GetDeployment(name, namespace string) (*appsv1.Deployment, error) {
	return te.K8sClientset.AppsV1().Deployments(namespace).Get(te.Context, name, metav1.GetOptions{})
}

func (te *TestEnvironment) WaitForEventingCRReady() error {
	// RetryGet the Eventing CR and test status.
	return common.Retry(Attempts, Interval, func() error {
		te.Logger.Debug(fmt.Sprintf("waiting for Eventing CR to get ready. "+
			"CR name: %s, namespace: %s", fixtures.CRName, fixtures.NamespaceName))

		// Get the Eventing CR from the cluster.
		gotEventingCR, err := common.RetryGet(Attempts, Interval, func() (*eventingv1alpha1.Eventing, error) {
			return te.GetEventingCRFromK8s(fixtures.CRName, fixtures.NamespaceName)
		})
		if err != nil {
			return err
		}

		if gotEventingCR.Spec.Backend.Type != gotEventingCR.Status.ActiveBackend {
			err := fmt.Errorf("waiting for Eventing CR to switch backend")
			te.Logger.Debug(err.Error())
			return err
		}

		if gotEventingCR.Status.State != eventingv1alpha1.StateReady {
			err := fmt.Errorf("waiting for Eventing CR to get ready state")
			te.Logger.Debug(err.Error())
			return err
		}

		// Everything is fine.
		te.Logger.Debug(fmt.Sprintf("Eventing CR is ready. "+
			"CR name: %s, namespace: %s", fixtures.CRName, fixtures.NamespaceName))
		return nil
	})
}

func (env *TestEnvironment) GetPeerAuthenticationFromK8s(name, namespace string) (*istio.PeerAuthentication, error) {
	result, err := env.K8sDynamicClient.Resource(fixtures.PeerAuthenticationGVR()).Namespace(
		namespace).Get(env.Context, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// convert from unstructured to structured.
	pa := &istio.PeerAuthentication{}
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(result.Object, pa); err != nil {
		return nil, err
	}
	return pa, nil
}
