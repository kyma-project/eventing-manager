package eventing

import (
	"bytes"
	"context"

	pkgerrors "github.com/kyma-project/kyma/components/eventing-controller/pkg/errors"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
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
	var certificateSecret corev1.Secret
	secretKey := client.ObjectKey{
		Namespace: r.backendConfig.Namespace,
		Name:      r.backendConfig.WebhookSecretName,
	}
	if err := r.Client.Get(ctx, secretKey, &certificateSecret); err != nil {
		return pkgerrors.MakeError(errObjectNotFound, err)
	}

	// get the mutating and validation WH config
	mutatingWH, validatingWH, err := r.getMutatingAndValidatingWebHookConfig(ctx)
	if err != nil {
		return err
	}

	// check that the mutating and validation WH config are valid
	if len(mutatingWH.Webhooks) == 0 {
		return pkgerrors.MakeError(errInvalidObject,
			errors.Errorf("mutatingWH %s does not have associated webhooks",
				r.backendConfig.MutatingWebhookName))
	}
	if len(validatingWH.Webhooks) == 0 {
		return pkgerrors.MakeError(errInvalidObject,
			errors.Errorf("validatingWH %s does not have associated webhooks",
				r.backendConfig.ValidatingWebhookName))
	}

	// check if the CABundle present is valid
	if !(mutatingWH.Webhooks[0].ClientConfig.CABundle != nil &&
		bytes.Equal(mutatingWH.Webhooks[0].ClientConfig.CABundle, certificateSecret.Data[TLSCertField])) {
		// update the ClientConfig for mutating WH config
		mutatingWH.Webhooks[0].ClientConfig.CABundle = certificateSecret.Data[TLSCertField]
		err = r.Client.Update(ctx, mutatingWH)
		if err != nil {
			return errors.Wrap(err, "while updating mutatingWH with caBundle")
		}
	}

	if !(validatingWH.Webhooks[0].ClientConfig.CABundle != nil &&
		bytes.Equal(validatingWH.Webhooks[0].ClientConfig.CABundle, certificateSecret.Data[TLSCertField])) {
		// update the ClientConfig for validating WH config
		validatingWH.Webhooks[0].ClientConfig.CABundle = certificateSecret.Data[TLSCertField]
		err = r.Client.Update(ctx, validatingWH)
		if err != nil {
			return errors.Wrap(err, "while updating validatingWH with caBundle")
		}
	}

	return nil
}

func (r *Reconciler) getMutatingAndValidatingWebHookConfig(ctx context.Context) (
	*admissionv1.MutatingWebhookConfiguration, *admissionv1.ValidatingWebhookConfiguration, error) {
	var mutatingWH admissionv1.MutatingWebhookConfiguration
	mutatingWHKey := client.ObjectKey{
		Name: r.backendConfig.MutatingWebhookName,
	}
	if err := r.Client.Get(ctx, mutatingWHKey, &mutatingWH); err != nil {
		return nil, nil, pkgerrors.MakeError(errObjectNotFound, err)
	}
	var validatingWH admissionv1.ValidatingWebhookConfiguration
	validatingWHKey := client.ObjectKey{
		Name: r.backendConfig.ValidatingWebhookName,
	}
	if err := r.Client.Get(ctx, validatingWHKey, &validatingWH); err != nil {
		return nil, nil, pkgerrors.MakeError(errObjectNotFound, err)
	}
	return &mutatingWH, &validatingWH, nil
}
