package eventing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	ecsubscriptionmanager "github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/object"
	"k8s.io/apimachinery/pkg/types"
)

const (
	secretKeyClientID     = "client_id"
	secretKeyClientSecret = "client_secret"
	secretKeyTokenURL     = "token_url"
	secretKeyCertsURL     = "certs_url"
)

type oauth2Credentials struct {
	clientID     []byte
	clientSecret []byte
	tokenURL     []byte
	certsURL     []byte
}

func (r *Reconciler) reconcileEventMeshSubManager(ctx context.Context, eventing *v1alpha1.Eventing, log *zap.SugaredLogger) error {
	// gets oauth2ClientID and secret and stops the EventMesh subscription manager if changed
	err := r.syncOauth2ClientIDAndSecret(ctx, eventing.Namespace)
	if err != nil {
		// TODO: update status
		//backendStatus.SetPublisherReadyCondition(false, eventingv1alpha1.ConditionReasonOauth2ClientSyncFailed, err.Error())
		//if updateErr := r.syncBackendStatus(ctx, backendStatus, nil); updateErr != nil {
		//	return ctrl.Result{}, errors.Wrapf(err, "failed to update status while syncing oauth2Client")
		//}
		return err
	}

	// retrieve secret to authenticate with EventMesh
	eventMeshSecret, err := r.kubeClient.GetSecret(ctx, eventing.Spec.Backend.Config.EventMeshSecret)
	if err != nil {
		return err
	}
	// CreateOrUpdate deployment for publisher proxy secret
	secretForPublisher, err := r.SyncPublisherProxySecret(ctx, eventMeshSecret)
	if err != nil {
		return err
	}

	// Set environment with secrets for EventMesh subscription controller
	err = setUpEnvironmentForEventMesh(secretForPublisher)
	if err != nil {
		return fmt.Errorf("failed to setup environment variables for EventMesh controller: %v", err)
	}

	if r.eventMeshSubManager == nil {
		// create instance of EventMesh subscription manager
		eventMeshSubManager, err := r.subManagerFactory.NewEventMeshManager(*eventing)
		if err != nil {
			return err
		}

		// init it
		if err = eventMeshSubManager.Init(r.ctrlManager); err != nil {
			return err
		}

		log.Info("EventMesh subscription-manager initialized")
		// save instance only when init is successful.
		r.eventMeshSubManager = eventMeshSubManager
	}

	if r.isEventMeshSubManagerStarted {
		log.Info("EventMesh subscription-manager is already started")
		return nil
	}

	defaultSubsConfig := r.eventingManager.GetBackendConfig().
		DefaultSubscriptionConfig.ToECENVDefaultSubscriptionConfig()
	eventMeshSubMgrParams := ecsubscriptionmanager.Params{
		ecsubscriptionmanager.ParamNameClientID:     r.credentials.clientID,
		ecsubscriptionmanager.ParamNameClientSecret: r.credentials.clientSecret,
		ecsubscriptionmanager.ParamNameTokenURL:     r.credentials.tokenURL,
		ecsubscriptionmanager.ParamNameCertsURL:     r.credentials.certsURL,
	}
	if err = r.eventMeshSubManager.Start(defaultSubsConfig, eventMeshSubMgrParams); err != nil {
		return err
	}
	log.Info("EventMesh subscription-manager started")
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

func (r *Reconciler) SyncPublisherProxySecret(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
	secretNamespacedName := types.NamespacedName{
		Namespace: secret.Namespace,
		Name:      deployment.PublisherName,
	}
	currentSecret := new(corev1.Secret)

	desiredSecret, err := getSecretForPublisher(secret)
	if err != nil {
		return nil, fmt.Errorf("invalid secret for Event Publisher: %v", err)
	}
	err = r.Get(ctx, secretNamespacedName, currentSecret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Create secret
			r.namedLogger().Debug("Creating secret for BEB publisher")
			err := r.Create(ctx, desiredSecret)
			if err != nil {
				return nil, fmt.Errorf("create secret for Event Publisher failed: %v", err)
			}
			return desiredSecret, nil
		}
		return nil, fmt.Errorf("failed to get Event Publisher secret failed: %v", err)
	}

	if object.Semantic.DeepEqual(currentSecret, desiredSecret) {
		r.namedLogger().Debug("No need to update secret for BEB Event Publisher")
		return currentSecret, nil
	}

	// Update secret
	desiredSecret.ResourceVersion = currentSecret.ResourceVersion
	if err := r.Update(ctx, desiredSecret); err != nil {
		return nil, fmt.Errorf("failed to update Event Publisher secret: %v", err)
	}

	return desiredSecret, nil
}

func (r *Reconciler) syncOauth2ClientIDAndSecret(ctx context.Context, secretNamespace string) error {
	credentials, err := r.getOAuth2ClientCredentials(ctx, secretNamespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	oauth2CredentialsNotFound := k8serrors.IsNotFound(err)
	oauth2CredentialsChanged := false
	if err == nil && r.isOauth2CredentialsInitialized() {
		oauth2CredentialsChanged = !bytes.Equal(r.credentials.clientID, credentials.clientID) ||
			!bytes.Equal(r.credentials.clientSecret, credentials.clientSecret) ||
			!bytes.Equal(r.credentials.tokenURL, credentials.tokenURL)
	}
	if oauth2CredentialsNotFound || oauth2CredentialsChanged {
		// Stop the controller and mark all subs as not ready
		message := "Stopping the BEB subscription manager due to change in OAuth2 credentials"
		r.namedLogger().Info(message)
		if err := r.eventMeshSubManager.Stop(false); err != nil {
			return err
		}
		r.isEventMeshSubManagerStarted = false
		// update eventing backend status to reflect that the controller is not ready
		// TODO: update status
		//backendStatus.SetSubscriptionControllerReadyCondition(false, eventingv1alpha1.ConditionReasonSubscriptionControllerNotReady, message)
		//if updateErr := r.syncBackendStatus(ctx, backendStatus, nil); updateErr != nil {
		//	return errors.Wrapf(err, "update status after stopping BEB controller failed")
		//}
	}
	if oauth2CredentialsNotFound {
		return err
	}
	if oauth2CredentialsChanged || !r.isOauth2CredentialsInitialized() {
		r.credentials.clientID = credentials.clientID
		r.credentials.clientSecret = credentials.clientSecret
		r.credentials.tokenURL = credentials.tokenURL
		r.credentials.certsURL = credentials.certsURL
	}
	return nil
}

func (r *Reconciler) getOAuth2ClientCredentials(ctx context.Context, secretNamespace string) (*oauth2Credentials, error) {
	var err error
	var exists bool
	var clientID, clientSecret, tokenURL, certsURL []byte

	oauth2Secret := new(corev1.Secret)
	oauth2SecretNamespacedName := types.NamespacedName{
		Namespace: secretNamespace,
		Name:      r.cfg.EventingWebhookAuthSecretName,
	}

	r.namedLogger().Infof("Reading secret %s", oauth2SecretNamespacedName.String())

	if getErr := r.Get(ctx, oauth2SecretNamespacedName, oauth2Secret); getErr != nil {
		err = fmt.Errorf("get secret failed namespace:%s name:%s: %v",
			oauth2SecretNamespacedName.Namespace, oauth2SecretNamespacedName.Name, getErr)
		return nil, err
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
	return len(r.credentials.clientID) > 0 &&
		len(r.credentials.clientSecret) > 0 &&
		len(r.credentials.tokenURL) > 0 &&
		len(r.credentials.certsURL) > 0
}

func newSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func getSecretForPublisher(eventMeshSecret *corev1.Secret) (*corev1.Secret, error) {
	secret := newSecret(deployment.PublisherName, eventMeshSecret.Namespace)

	secret.Labels = map[string]string{
		deployment.AppLabelKey: AppLabelValue,
	}

	if _, ok := eventMeshSecret.Data["messaging"]; !ok {
		return nil, errors.New("message is missing from BEB secret")
	}
	messagingBytes := eventMeshSecret.Data["messaging"]

	if _, ok := eventMeshSecret.Data["namespace"]; !ok {
		return nil, errors.New("namespace is missing from BEB secret")
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
		deployment.PublisherSecretClientIDKey:      clientID,
		deployment.PublisherSecretClientSecretKey:  clientSecret,
		deployment.PublisherSecretTokenEndpointKey: fmt.Sprintf(TokenEndpointFormat, tokenEndpoint, grantType),
		deployment.PublisherSecretEMSURLKey:        fmt.Sprintf("%s%s", publishURL, EventMeshPublishEndpointForPublisher),
		PublisherSecretEMSHostKey:                  publishURL,
		deployment.PublisherSecretBEBNamespaceKey:  namespace,
	}
}

func setUpEnvironmentForEventMesh(secret *corev1.Secret) error {
	err := os.Setenv("BEB_API_URL", fmt.Sprintf("%s%s", string(secret.Data[PublisherSecretEMSHostKey]), EventMeshPublishEndpointForSubscriber))
	if err != nil {
		return fmt.Errorf("set BEB_API_URL env var failed: %v", err)
	}

	err = os.Setenv("CLIENT_ID", string(secret.Data[deployment.PublisherSecretClientIDKey]))
	if err != nil {
		return fmt.Errorf("set CLIENT_ID env var failed: %v", err)
	}

	err = os.Setenv("CLIENT_SECRET", string(secret.Data[deployment.PublisherSecretClientSecretKey]))
	if err != nil {
		return fmt.Errorf("set CLIENT_SECRET env var failed: %v", err)
	}

	err = os.Setenv("TOKEN_ENDPOINT", string(secret.Data[deployment.PublisherSecretTokenEndpointKey]))
	if err != nil {
		return fmt.Errorf("set TOKEN_ENDPOINT env var failed: %v", err)
	}

	err = os.Setenv("BEB_NAMESPACE", fmt.Sprintf("%s%s", NamespacePrefix, string(secret.Data[deployment.PublisherSecretBEBNamespaceKey])))
	if err != nil {
		return fmt.Errorf("set BEB_NAMESPACE env var failed: %v", err)
	}

	return nil
}
