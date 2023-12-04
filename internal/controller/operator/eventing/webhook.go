package eventing

import (
	"bytes"
	"context"

	emerrors "github.com/kyma-project/eventing-manager/pkg/errors"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TLSCertField = "tls.crt"
)

var (
	errObjectNotFound = errors.New("object not found")
	errInvalidObject  = errors.New("invalid object")
)

// reconcileWebhooksWithCABundle injects the CABundle into mutating and validating webhooks.
func (r *Reconciler) reconcileWebhooksWithCABundle(ctx context.Context) error {
	// get the secret containing the certificate
	secretKey := client.ObjectKey{
		Namespace: r.backendConfig.Namespace,
		Name:      r.backendConfig.WebhookSecretName,
	}
	certificateSecret, err := r.kubeClient.GetSecret(ctx, secretKey.String())
	if err != nil {
		return emerrors.MakeError(errObjectNotFound, err)
	}

	// get the mutating WH config.
	mutatingWH, err := r.kubeClient.GetMutatingWebHookConfiguration(ctx, r.backendConfig.MutatingWebhookName)
	if err != nil {
		return emerrors.MakeError(errObjectNotFound, err)
	}

	// get the validation WH config.
	validatingWH, err := r.kubeClient.GetValidatingWebHookConfiguration(ctx, r.backendConfig.ValidatingWebhookName)
	if err != nil {
		return emerrors.MakeError(errObjectNotFound, err)
	}

	// check that the mutating and validating WH config are valid
	if len(mutatingWH.Webhooks) == 0 {
		return emerrors.MakeError(errInvalidObject,
			errors.Errorf("mutatingWH %s does not have associated webhooks",
				r.backendConfig.MutatingWebhookName))
	}
	if len(validatingWH.Webhooks) == 0 {
		return emerrors.MakeError(errInvalidObject,
			errors.Errorf("validatingWH %s does not have associated webhooks",
				r.backendConfig.ValidatingWebhookName))
	}

	// check if the CABundle present is valid
	if !(mutatingWH.Webhooks[0].ClientConfig.CABundle != nil &&
		bytes.Equal(mutatingWH.Webhooks[0].ClientConfig.CABundle, certificateSecret.Data[TLSCertField])) {
		// update the ClientConfig for mutating WH config
		mutatingWH.Webhooks[0].ClientConfig.CABundle = certificateSecret.Data[TLSCertField]
		// update the mutating WH on k8s.
		if err = r.Client.Update(ctx, mutatingWH); err != nil {
			return errors.Wrap(err, "while updating mutatingWH with caBundle")
		}
	}

	if !(validatingWH.Webhooks[0].ClientConfig.CABundle != nil &&
		bytes.Equal(validatingWH.Webhooks[0].ClientConfig.CABundle, certificateSecret.Data[TLSCertField])) {
		// update the ClientConfig for validating WH config
		validatingWH.Webhooks[0].ClientConfig.CABundle = certificateSecret.Data[TLSCertField]
		// update the validating WH on k8s.
		if err = r.Client.Update(ctx, validatingWH); err != nil {
			return errors.Wrap(err, "while updating validatingWH with caBundle")
		}
	}

	return nil
}
