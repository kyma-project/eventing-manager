package validator

import (
	"context"
	"fmt"

	pkgerrors "github.com/pkg/errors"

	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
)

var ErrSubscriptionValidationFailed = pkgerrors.New("Subscription validation failed")

//go:generate go run github.com/vektra/mockery/v2 --name=SubscriptionValidator --outpkg=mocks
type SubscriptionValidator interface {
	Validate(ctx context.Context, subscription eventingv1alpha2.Subscription) error
}

type subscriptionValidator struct {
	sinkValidator SinkValidator
}

// Perform a compile-time check.
var _ SubscriptionValidator = &subscriptionValidator{}

func NewSubscriptionValidator(sinkValidator SinkValidator) SubscriptionValidator {
	return &subscriptionValidator{sinkValidator: sinkValidator}
}

func (sv *subscriptionValidator) Validate(ctx context.Context, subscription eventingv1alpha2.Subscription) error {
	if errs := validateSpec(subscription); len(errs) > 0 {
		return fmt.Errorf("%w: %w", ErrSubscriptionValidationFailed, errs.ToAggregate())
	}
	if err := sv.sinkValidator.Validate(ctx, subscription.Spec.Sink); err != nil {
		return fmt.Errorf("%w: %w", ErrSubscriptionValidationFailed, err)
	}
	return nil
}

// SubscriptionValidatorFunc implements the SinkValidator interface.
type SubscriptionValidatorFunc func(ctx context.Context, subscription eventingv1alpha2.Subscription) error

// Perform a compile-time check.
var _ SubscriptionValidator = SubscriptionValidatorFunc(func(_ context.Context, _ eventingv1alpha2.Subscription) error { return nil })

func (svf SubscriptionValidatorFunc) Validate(ctx context.Context, subscription eventingv1alpha2.Subscription) error {
	return svf(ctx, subscription)
}
