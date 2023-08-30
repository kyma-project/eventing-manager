package eventing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/kyma/components/eventing-controller/pkg/deployment"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/object"
	"k8s.io/apimachinery/pkg/types"
)

func (r *Reconciler) startEventMeshSubscriptionController(ctx context.Context, eventing *v1alpha1.Eventing) error {
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

	// Set environment with secrets for BEB subscription controller
	err = setUpEnvironmentForEventMesh(secretForPublisher)
	if err != nil {
		return fmt.Errorf("failed to setup environment variables for EventMesh controller: %v", err)
	}

	// TODO: start subscription controller

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
