package eventing

import (
	"context"
	"errors"
	"fmt"
)

const (
	shootInfoConfigMapName      = "shoot-info"
	shootInfoConfigMapNamespace = "kube-system"
	shootInfoConfigMapKeyDomain = "domain"
	domainMissingMessageFormat  = `%w. domain must be configured in either the Eventing` +
		` CustomResource under "Spec.Backend.Config.Domain" or in the ConfigMap "%s/%s" under "data.%s"`
	domainMissingMessageFormatWithError = domainMissingMessageFormat + `: %w`
)

var ErrDomainConfigMissing = errors.New("domain configuration missing")

func (r *Reconciler) readDomainFromConfigMap(ctx context.Context) (string, error) {
	cm, err := r.kubeClient.GetConfigMap(ctx, shootInfoConfigMapName, shootInfoConfigMapNamespace)
	if err != nil {
		return "", err
	}
	return cm.Data[shootInfoConfigMapKeyDomain], nil
}

func domainMissingError(err error) error {
	if err != nil {
		return fmt.Errorf(
			domainMissingMessageFormatWithError, ErrDomainConfigMissing,
			shootInfoConfigMapNamespace, shootInfoConfigMapName, shootInfoConfigMapKeyDomain, err,
		)
	}
	return fmt.Errorf(
		domainMissingMessageFormat, ErrDomainConfigMissing,
		shootInfoConfigMapNamespace, shootInfoConfigMapName, shootInfoConfigMapKeyDomain,
	)
}
