package sink

import (
	"context"
	"fmt"

	"golang.org/x/xerrors"
	kcorev1 "k8s.io/api/core/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	"github.com/kyma-project/eventing-manager/internal/controller/events"
	"github.com/kyma-project/eventing-manager/pkg/utils"
)

var ErrSinkValidationFailed = xerrors.New("Sink validation failed")

type Validator interface {
	Validate(ctx context.Context, subscription *v1alpha2.Subscription) error
}

// ValidatorFunc implements the Validator interface.
type ValidatorFunc func(context.Context, *v1alpha2.Subscription) error

func (vf ValidatorFunc) Validate(ctx context.Context, sub *v1alpha2.Subscription) error {
	return vf(ctx, sub)
}

type defaultSinkValidator struct {
	client   client.Client
	recorder record.EventRecorder
}

// Perform a compile-time check.
var _ Validator = &defaultSinkValidator{}

func NewValidator(client client.Client, recorder record.EventRecorder) Validator {
	return &defaultSinkValidator{client: client, recorder: recorder}
}

func (s defaultSinkValidator) Validate(ctx context.Context, subscription *v1alpha2.Subscription) error {
	var (
		svcNs, svcName string
	)

	if _, subDomains, err := utils.GetSinkData(subscription.Spec.Sink); err != nil {
		return err
	} else {
		svcNs = subDomains[1]
		svcName = subDomains[0]
	}

	if _, err := GetClusterLocalService(ctx, s.client, svcNs, svcName); err != nil {
		events.Warn(s.recorder, subscription, events.ReasonValidationFailed, "Sink does not correspond to a valid cluster local svc")
		return fmt.Errorf("%w: %w", ErrSinkValidationFailed, err)
	}

	return nil
}

func GetClusterLocalService(ctx context.Context, client client.Client, svcNs, svcName string) (*kcorev1.Service, error) {
	svcLookupKey := ktypes.NamespacedName{Name: svcName, Namespace: svcNs}
	svc := &kcorev1.Service{}
	if err := client.Get(ctx, svcLookupKey, svc); err != nil {
		return nil, err
	}
	return svc, nil
}
