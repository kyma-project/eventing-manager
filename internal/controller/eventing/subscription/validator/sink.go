package validator

import (
	"context"
	"fmt"

	pkgerrors "github.com/pkg/errors"
	kcorev1 "k8s.io/api/core/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/eventing-manager/pkg/utils"
)

var ErrSinkValidationFailed = pkgerrors.New("Sink validation failed")

type SinkValidator interface {
	Validate(ctx context.Context, sink string) error
}

type sinkValidator struct {
	client client.Client
}

// Perform a compile-time check.
var _ SinkValidator = &sinkValidator{}

func NewSinkValidator(client client.Client) SinkValidator {
	return &sinkValidator{client: client}
}

func (sv sinkValidator) Validate(ctx context.Context, sink string) error {
	_, subDomains, err := utils.GetSinkData(sink)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSinkValidationFailed, err)
	}

	if len(subDomains) < 2 {
		return fmt.Errorf("%w: sink format should contain the service name.namespace", ErrSinkValidationFailed)
	}

	namespace, name := subDomains[1], subDomains[0]
	if !sv.serviceExists(ctx, namespace, name) {
		return fmt.Errorf("%w: service %s.%s not found in the cluster", ErrSinkValidationFailed, name, namespace)
	}

	return nil
}

func (sv sinkValidator) serviceExists(ctx context.Context, namespace, name string) bool {
	return sv.client.Get(ctx, ktypes.NamespacedName{Namespace: namespace, Name: name}, &kcorev1.Service{}) == nil
}

// SinkValidatorFunc implements the SinkValidator interface.
type SinkValidatorFunc func(ctx context.Context, sink string) error

// Perform a compile-time check.
var _ SinkValidator = SinkValidatorFunc(func(_ context.Context, _ string) error { return nil })

func (svf SinkValidatorFunc) Validate(ctx context.Context, sink string) error {
	return svf(ctx, sink)
}
