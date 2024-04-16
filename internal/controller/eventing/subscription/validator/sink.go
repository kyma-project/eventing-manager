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

type sinkValidator interface {
	validate(ctx context.Context, sink string) error
}

type sinkServiceValidator struct {
	client client.Client
}

// Perform a compile-time check.
var _ sinkValidator = &sinkServiceValidator{}

func newSinkValidator(client client.Client) sinkValidator {
	return &sinkServiceValidator{client: client}
}

func (sv sinkServiceValidator) validate(ctx context.Context, sink string) error {
	_, subDomains, err := utils.GetSinkData(sink)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSinkValidationFailed, err)
	}

	const minSubDomains = 2
	if len(subDomains) < minSubDomains {
		return fmt.Errorf("%w: sink format should contain the service name.namespace", ErrSinkValidationFailed)
	}

	namespace, name := subDomains[1], subDomains[0]
	if !sv.serviceExists(ctx, namespace, name) {
		return fmt.Errorf("%w: service %s.%s not found in the cluster", ErrSinkValidationFailed, name, namespace)
	}

	return nil
}

func (sv sinkServiceValidator) serviceExists(ctx context.Context, namespace, name string) bool {
	return sv.client.Get(ctx, ktypes.NamespacedName{Namespace: namespace, Name: name}, &kcorev1.Service{}) == nil
}

// sinkValidatorFunc implements the sinkValidator interface.
type sinkValidatorFunc func(ctx context.Context, sink string) error

// Perform a compile-time check.
var _ sinkValidator = sinkValidatorFunc(func(_ context.Context, _ string) error { return nil })

func (svf sinkValidatorFunc) validate(ctx context.Context, sink string) error {
	return svf(ctx, sink)
}
