package webhook

import (
	"context"

	kadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	kbatchv1 "k8s.io/api/batch/v1"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace                      = "kyma-system"
	service                        = "eventing-manager-webhook-service"
	cronjob                        = "eventing-manager-cert-handler"
	job                            = "eventing-manager-cert-handler"
	mutatingWebhookConfiguration   = "subscription-mutating-webhook-configuration"
	validatingWebhookConfiguration = "subscription-validating-webhook-configuration"
)

// CleanupResources removes the mutating and validating webhook resources.
func CleanupResources(ctx context.Context, client kctrlclient.Client) []error {
	const capacity = 5
	errList := make([]error, 0, capacity)
	errList = appendIfError(errList, deleteService(ctx, client, namespace, service))
	errList = appendIfError(errList, deleteCronJob(ctx, client, namespace, cronjob))
	errList = appendIfError(errList, deleteJob(ctx, client, namespace, job))
	errList = appendIfError(errList, deleteMutatingWebhookConfiguration(ctx, client, namespace, mutatingWebhookConfiguration))
	errList = appendIfError(errList, deleteValidatingWebhookConfiguration(ctx, client, namespace, validatingWebhookConfiguration))
	return errList
}

func deleteService(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return kctrlclient.IgnoreNotFound(
		client.Delete(
			ctx,
			&kcorev1.Service{
				ObjectMeta: kmetav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
			},
			kctrlclient.PropagationPolicy(kmetav1.DeletePropagationBackground),
		),
	)
}

func deleteCronJob(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return kctrlclient.IgnoreNotFound(
		client.Delete(
			ctx,
			&kbatchv1.CronJob{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			},
			kctrlclient.PropagationPolicy(kmetav1.DeletePropagationBackground),
		),
	)
}

func deleteJob(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return kctrlclient.IgnoreNotFound(
		client.Delete(
			ctx,
			&kbatchv1.Job{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			},
			kctrlclient.PropagationPolicy(kmetav1.DeletePropagationBackground),
		),
	)
}

func deleteMutatingWebhookConfiguration(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return kctrlclient.IgnoreNotFound(
		client.Delete(
			ctx,
			&kadmissionregistrationv1.MutatingWebhookConfiguration{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			},
			kctrlclient.PropagationPolicy(kmetav1.DeletePropagationBackground),
		),
	)
}

func deleteValidatingWebhookConfiguration(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return kctrlclient.IgnoreNotFound(
		client.Delete(
			ctx,
			&kadmissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: kmetav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			},
			kctrlclient.PropagationPolicy(kmetav1.DeletePropagationBackground),
		),
	)
}

func appendIfError(errList []error, err error) []error {
	if err != nil {
		return append(errList, err)
	}
	return errList
}
