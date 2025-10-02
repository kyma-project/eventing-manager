//go:generate go run github.com/vektra/mockery/v2 --name Backend
//go:generate go run github.com/vektra/mockery/v2 --srcpkg github.com/nats-io/nats.go --name JetStreamContext
package jetstream

import (
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nats-io/nats.go"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/pkg/backend/cleaner"
	backendmetrics "github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	backendutils "github.com/kyma-project/eventing-manager/pkg/backend/utils"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/logger"
)

const (
	separator = "/"
)

//go:generate go run github.com/vektra/mockery/v2 --name Backend
type Backend interface {
	// Initialize should initialize the communication layer with the messaging backend system
	Initialize(connCloseHandler backendutils.ConnClosedHandler) error

	// Shutdown should stop all clients.
	Shutdown()

	// SyncSubscription should synchronize the Kyma eventing subscription with the subscriber infrastructure of JetStream.
	SyncSubscription(subscription *eventingv1alpha2.Subscription) error

	// DeleteSubscription should delete the corresponding subscriber data of messaging backend
	DeleteSubscription(subscription *eventingv1alpha2.Subscription) error

	// DeleteSubscriptionsOnly should delete the JetStream subscriptions only.
	// The corresponding JetStream consumers of the subscriptions must not be deleted.
	DeleteSubscriptionsOnly(subscription *eventingv1alpha2.Subscription) error

	// GetJetStreamSubjects returns a list of subjects appended with stream name and source as prefix if needed
	GetJetStreamSubjects(source string, subjects []string, typeMatching eventingv1alpha2.TypeMatching) []string

	// DeleteInvalidConsumers deletes all JetStream consumers having no subscription types in subscription resources
	DeleteInvalidConsumers(subscriptions []eventingv1alpha2.Subscription) error

	// GetJetStreamContext returns the current JetStreamContext
	GetJetStreamContext() nats.JetStreamContext

	// GetConfig returns the backends Configuration
	GetConfig() env.NATSConfig
}

type JetStream struct {
	Config        env.NATSConfig
	Conn          *nats.Conn
	jsCtx         nats.JetStreamContext
	client        cloudevents.Client
	subscriptions map[SubscriptionSubjectIdentifier]Subscriber
	sinks         sync.Map
	// connClosedHandler gets called by the NATS server when Conn is closed and retry attempts are exhausted.
	connClosedHandler backendutils.ConnClosedHandler
	logger            *logger.Logger
	metricsCollector  *backendmetrics.Collector
	cleaner           cleaner.Cleaner
	subsConfig        env.DefaultSubscriptionConfig
}

func (js *JetStream) GetConfig() env.NATSConfig {
	return js.Config
}

type Subscriber interface {
	SubscriptionSubject() string
	ConsumerInfo() (*nats.ConsumerInfo, error)
	IsValid() bool
	Unsubscribe() error
	SetPendingLimits(msgLimit, byteLimit int) error
	PendingLimits() (int, int, error)
}

type Subscription struct {
	*nats.Subscription
}

// SubscriptionSubjectIdentifier is used to uniquely identify a Subscription subject.
// It should be used only with JetStream backend.
type SubscriptionSubjectIdentifier struct {
	consumerName, namespacedSubjectName string
}

type DefaultSubOpts []nats.SubOpt

//----------------------------------------
// JetStream Backend Test Types
//----------------------------------------

type jetStreamClient struct {
	nats.JetStreamContext
	natsConn *nats.Conn
}

func (js Subscription) SubscriptionSubject() string {
	return js.Subject
}
