package eventing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	kcorev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	"github.com/kyma-project/eventing-manager/internal/label"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	submgrmanager "github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/manager"
	"github.com/kyma-project/eventing-manager/pkg/utils"
)

const (
	secretKeyClientID     = "client_id"
	secretKeyClientSecret = "client_secret"
	secretKeyTokenURL     = "token_url"
	secretKeyCertsURL     = "certs_url"
)

var (
	ErrEMSecretMessagingMissing = errors.New("messaging is missing from EM secret")
	ErrEMSecretNamespaceMissing = errors.New("namespace is missing from EM secret")
)

type oauth2Credentials struct {
	clientID     []byte
	clientSecret []byte
	tokenURL     []byte
	certsURL     []byte
}

const EventMeshSecretMissingMessage = "The specified EventMesh secret is not found. Please provide an existing secret."

func (r *Reconciler) reconcileEventMeshSubManager(ctx context.Context, eventing *v1alpha1.Eventing,
	eventMeshSecret *kcorev1.Secret, log *zap.SugaredLogger,
) error {
	// gets oauth2ClientID and secret and stops the EventMesh subscription manager if changed
	err := r.syncOauth2ClientIDAndSecret(ctx, eventing)
	if err != nil {
		return fmt.Errorf("failed to sync OAuth secret: %w", err)
	}

	// CreateOrUpdate deployment for publisher proxy secret
	secretForPublisher, err := r.SyncPublisherProxySecret(ctx, eventMeshSecret)
	if err != nil {
		return fmt.Errorf("failed to sync Publisher Proxy secret: %w", err)
	}

	// Set environment with secrets for EventMesh subscription controller
	err = setUpEnvironmentForEventMesh(secretForPublisher, eventing)
	if err != nil {
		return fmt.Errorf("failed to setup environment variables for EventMesh controller: %w", err)
	}

	// Read the cluster domain from the Eventing CR, or
	// read it from the configmap managed by gardener
	domain := eventing.Spec.Backend.Config.Domain
	if utils.IsEmpty(domain) {
		r.namedLogger().Infof(
			`Domain is not configured in the Eventing CR, reading it from the ConfigMap %s/%s`,
			shootInfoConfigMapNamespace, shootInfoConfigMapName,
		)
		domain, err = r.readDomainFromConfigMap(ctx)
		if err != nil || utils.IsEmpty(domain) {
			return domainMissingError(err)
		}
	}
	r.namedLogger().Infof(`Domain is %s`, domain)

	// get the subscription config
	defaultSubsConfig := r.getDefaultSubscriptionConfig()
	// get the subManager parameters
	eventMeshSubMgrParams := r.getEventMeshSubManagerParams()
	// get the hash of current config
	specHash, err := getEventMeshBackendConfigHash(
		eventing.Spec.Backend.Config.EventMeshSecret,
		eventing.Spec.Backend.Config.EventTypePrefix,
		domain,
	)
	if err != nil {
		return err
	}

	// update the config if hashes differ
	if eventing.Status.BackendConfigHash != specHash {
		// stop the subsManager without cleanup
		if err := r.stopEventMeshSubManager(false, r.namedLogger()); err != nil {
			return err
		}
	}

	if r.eventMeshSubManager == nil {
		// create instance of EventMesh subscription manager
		eventMeshSubManager, err := r.subManagerFactory.NewEventMeshManager(domain)
		if err != nil {
			return err
		}

		// init it
		if err = eventMeshSubManager.Init(r.ctrlManager); err != nil {
			return err
		}

		r.namedLogger().Info("EventMesh subscription-manager initialized")
		// save instance only when init is successful.
		r.eventMeshSubManager = eventMeshSubManager
	}

	if r.isEventMeshSubManagerStarted {
		r.namedLogger().Info("EventMesh subscription-manager is already started")
		return nil
	}

	err = r.startEventMeshSubManager(defaultSubsConfig, eventMeshSubMgrParams)
	if err != nil {
		return err
	}

	// update the hash of the current config only once subManager is started
	eventing.Status.BackendConfigHash = specHash
	r.namedLogger().Info(fmt.Sprintf("EventMesh subscription-manager has been updated, new hash: %d", specHash))

	return nil
}

func (r *Reconciler) getEventMeshSubManagerParams() submgrmanager.Params {
	return submgrmanager.Params{
		submgrmanager.ParamNameClientID:     r.oauth2credentials.clientID,
		submgrmanager.ParamNameClientSecret: r.oauth2credentials.clientSecret,
		submgrmanager.ParamNameTokenURL:     r.oauth2credentials.tokenURL,
		submgrmanager.ParamNameCertsURL:     r.oauth2credentials.certsURL,
	}
}

func (r *Reconciler) startEventMeshSubManager(defaultSubsConfig env.DefaultSubscriptionConfig,
	eventMeshSubMgrParams submgrmanager.Params,
) error {
	if err := r.eventMeshSubManager.Start(defaultSubsConfig, eventMeshSubMgrParams); err != nil {
		return err
	}

	r.namedLogger().Info("EventMesh subscription-manager started")
	// update flag so it does not try to start the manager again
	r.isEventMeshSubManagerStarted = true
	return nil
}

func (r *Reconciler) stopEventMeshSubManager(runCleanup bool, log *zap.SugaredLogger) error {
	log.Debug("stopping EventMesh subscription-manager")
	if r.eventMeshSubManager == nil || !r.isEventMeshSubManagerStarted {
		log.Info("EventMesh subscription-manager is already stopped!")
		return nil
	}

	// stop the subscription manager.
	if err := r.eventMeshSubManager.Stop(runCleanup); err != nil {
		return err
	}

	log.Info("EventMesh subscription-manager stopped!")
	// update flags so it does not try to stop the manager again.
	r.isEventMeshSubManagerStarted = false
	r.eventMeshSubManager = nil

	return nil
}

func (r *Reconciler) SyncPublisherProxySecret(ctx context.Context, secret *kcorev1.Secret) (*kcorev1.Secret, error) {
	desiredSecret, err := getSecretForPublisher(secret)
	if err != nil {
		return nil, fmt.Errorf("invalid secret for Event Publisher: %w", err)
	}

	err = r.kubeClient.PatchApply(ctx, desiredSecret)
	if err != nil {
		return nil, err
	}
	return desiredSecret, nil
}

func (r *Reconciler) syncOauth2ClientIDAndSecret(ctx context.Context, eventing *v1alpha1.Eventing) error {
	credentials, err := r.getOAuth2ClientCredentials(ctx, eventing.Namespace)
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}
	oauth2CredentialsNotFound := kerrors.IsNotFound(err)
	oauth2CredentialsChanged := false
	if err == nil && r.isOauth2CredentialsInitialized() {
		oauth2CredentialsChanged = !bytes.Equal(r.oauth2credentials.clientID, credentials.clientID) ||
			!bytes.Equal(r.oauth2credentials.clientSecret, credentials.clientSecret) ||
			!bytes.Equal(r.oauth2credentials.tokenURL, credentials.tokenURL) ||
			!bytes.Equal(r.oauth2credentials.certsURL, credentials.certsURL)
	}
	if oauth2CredentialsNotFound || oauth2CredentialsChanged {
		// Stop the controller and mark all subs as not ready
		message := "Stopping the EventMesh subscription manager due to change in OAuth2 oauth2credentials"
		r.namedLogger().Info(message)
		if stopErr := r.stopEventMeshSubManager(true, r.namedLogger()); stopErr != nil {
			return stopErr
		}
		// update eventing status to reflect that the EventMesh sub manager is not ready
		if updateErr := r.syncStatusWithSubscriptionManagerFailedCondition(ctx, eventing,
			errors.New(message), r.namedLogger()); updateErr != nil {
			return updateErr
		}
	}
	if oauth2CredentialsNotFound {
		return err
	}
	if oauth2CredentialsChanged || !r.isOauth2CredentialsInitialized() {
		r.oauth2credentials.clientID = credentials.clientID
		r.oauth2credentials.clientSecret = credentials.clientSecret
		r.oauth2credentials.tokenURL = credentials.tokenURL
		r.oauth2credentials.certsURL = credentials.certsURL
	}
	return nil
}

func (r *Reconciler) getOAuth2ClientCredentials(ctx context.Context, secretNamespace string) (*oauth2Credentials, error) {
	var err error
	var exists bool
	var clientID, clientSecret, tokenURL, certsURL []byte

	oauth2Secret := new(kcorev1.Secret)
	oauth2SecretNamespacedName := types.NamespacedName{
		Namespace: secretNamespace,
		Name:      r.backendConfig.EventingWebhookAuthSecretName,
	}

	r.namedLogger().Infof("Reading secret %s", oauth2SecretNamespacedName.String())

	if getErr := r.Get(ctx, oauth2SecretNamespacedName, oauth2Secret); getErr != nil {
		return nil, getErr
	}

	if clientID, exists = oauth2Secret.Data[secretKeyClientID]; !exists {
		err = errors.Errorf("key '%s' not found in secret %s",
			secretKeyClientID, oauth2SecretNamespacedName.String())
		return nil, err
	}

	if clientSecret, exists = oauth2Secret.Data[secretKeyClientSecret]; !exists {
		err = errors.Errorf("key '%s' not found in secret %s",
			secretKeyClientSecret, oauth2SecretNamespacedName.String())
		return nil, err
	}

	if tokenURL, exists = oauth2Secret.Data[secretKeyTokenURL]; !exists {
		err = errors.Errorf("key '%s' not found in secret %s",
			secretKeyTokenURL, oauth2SecretNamespacedName.String())
		return nil, err
	}

	if certsURL, exists = oauth2Secret.Data[secretKeyCertsURL]; !exists {
		err = errors.Errorf("key '%s' not found in secret %s",
			secretKeyCertsURL, oauth2SecretNamespacedName.String())
		return nil, err
	}

	credentials := oauth2Credentials{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		certsURL:     certsURL,
	}

	return &credentials, nil
}

func (r *Reconciler) isOauth2CredentialsInitialized() bool {
	return len(r.oauth2credentials.clientID) > 0 &&
		len(r.oauth2credentials.clientSecret) > 0 &&
		len(r.oauth2credentials.tokenURL) > 0 &&
		len(r.oauth2credentials.certsURL) > 0
}

func newSecret(name, namespace string) *kcorev1.Secret {
	return &kcorev1.Secret{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: kcorev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func getSecretForPublisher(eventMeshSecret *kcorev1.Secret) (*kcorev1.Secret, error) {
	secret := newSecret(eventing.PublisherName, eventMeshSecret.Namespace)

	secret.Labels = map[string]string{
		label.KeyName: label.ValueEventingPublisherProxy,
	}

	if _, ok := eventMeshSecret.Data["messaging"]; !ok {
		return nil, ErrEMSecretMessagingMissing
	}
	messagingBytes := eventMeshSecret.Data["messaging"]

	if _, ok := eventMeshSecret.Data["namespace"]; !ok {
		return nil, ErrEMSecretNamespaceMissing
	}
	namespaceBytes := eventMeshSecret.Data["namespace"]

	var messages []Message
	err := json.Unmarshal(messagingBytes, &messages)
	if err != nil {
		return nil, err
	}

	for _, m := range messages {
		if m.Broker.BrokerType == "saprestmgw" {
			if len(m.OA2.ClientID) == 0 {
				return nil, errors.New("client ID is missing")
			}
			if len(m.OA2.ClientSecret) == 0 {
				return nil, errors.New("client secret is missing")
			}
			if len(m.OA2.TokenEndpoint) == 0 {
				return nil, errors.New("tokenendpoint is missing")
			}
			if len(m.OA2.GrantType) == 0 {
				return nil, errors.New("granttype is missing")
			}
			if len(m.URI) == 0 {
				return nil, errors.New("publish URL is missing")
			}

			secret.StringData = getSecretStringData(m.OA2.ClientID, m.OA2.ClientSecret, m.OA2.TokenEndpoint, m.OA2.GrantType, m.URI, string(namespaceBytes))
			break
		}
	}

	return secret, nil
}

func getSecretStringData(clientID, clientSecret, tokenEndpoint, grantType, publishURL, namespace string) map[string]string {
	return map[string]string{
		eventing.PublisherSecretClientIDKey:      clientID,
		eventing.PublisherSecretClientSecretKey:  clientSecret,
		eventing.PublisherSecretTokenEndpointKey: fmt.Sprintf(TokenEndpointFormat, tokenEndpoint, grantType),
		eventing.PublisherSecretEMSURLKey:        fmt.Sprintf("%s%s", publishURL, EventMeshPublishEndpointForPublisher),
		PublisherSecretEMSHostKey:                publishURL,
		eventing.PublisherSecretBEBNamespaceKey:  namespace,
	}
}

func setUpEnvironmentForEventMesh(secret *kcorev1.Secret, eventingCR *v1alpha1.Eventing) error {
	err := os.Setenv("BEB_API_URL", fmt.Sprintf("%s%s", string(secret.Data[PublisherSecretEMSHostKey]), EventMeshPublishEndpointForSubscriber))
	if err != nil {
		return fmt.Errorf("set BEB_API_URL env var failed: %w", err)
	}

	err = os.Setenv("CLIENT_ID", string(secret.Data[eventing.PublisherSecretClientIDKey]))
	if err != nil {
		return fmt.Errorf("set CLIENT_ID env var failed: %w", err)
	}

	err = os.Setenv("CLIENT_SECRET", string(secret.Data[eventing.PublisherSecretClientSecretKey]))
	if err != nil {
		return fmt.Errorf("set CLIENT_SECRET env var failed: %w", err)
	}

	err = os.Setenv("TOKEN_ENDPOINT", string(secret.Data[eventing.PublisherSecretTokenEndpointKey]))
	if err != nil {
		return fmt.Errorf("set TOKEN_ENDPOINT env var failed: %w", err)
	}

	err = os.Setenv("BEB_NAMESPACE", fmt.Sprintf("%s%s", NamespacePrefix, string(secret.Data[eventing.PublisherSecretBEBNamespaceKey])))
	if err != nil {
		return fmt.Errorf("set BEB_NAMESPACE env var failed: %w", err)
	}

	if err := os.Setenv("EVENT_TYPE_PREFIX", eventingCR.Spec.Backend.Config.EventTypePrefix); err != nil {
		return fmt.Errorf("set EVENT_TYPE_PREFIX env var failed: %w", err)
	}

	return nil
}
