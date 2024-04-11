package webhook

import (
	"context"

	kadmissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	kbatchv1 "k8s.io/api/batch/v1"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CleanupResources removes the mutating and validating webhook resources.
func CleanupResources(ctx context.Context, client kctrlclient.Client) []error {
	const (
		namespace                      = "kyma-system"
		service                        = "eventing-manager-webhook-service"
		cronjob                        = "eventing-manager-cert-handler"
		job                            = "eventing-manager-cert-handler"
		mutatingWebhookConfiguration   = "subscription-mutating-webhook-configuration"
		validatingWebhookConfiguration = "subscription-validating-webhook-configuration"
	)
	return []error{
		deleteService(ctx, client, namespace, service),
		deleteCronJob(ctx, client, namespace, cronjob),
		deleteJob(ctx, client, namespace, job),
		deleteMutatingWebhookConfiguration(ctx, client, namespace, mutatingWebhookConfiguration),
		deleteValidatingWebhookConfiguration(ctx, client, namespace, validatingWebhookConfiguration),
	}
}

func deleteService(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return client.Delete(ctx, &kcorev1.Service{
		ObjectMeta: kmetav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	})
}

func deleteCronJob(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return client.Delete(ctx, &kbatchv1.CronJob{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func deleteJob(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return client.Delete(ctx, &kbatchv1.Job{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func deleteMutatingWebhookConfiguration(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return client.Delete(ctx, &kadmissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func deleteValidatingWebhookConfiguration(ctx context.Context, client kctrlclient.Client, namespace, name string) error {
	return client.Delete(ctx, &kadmissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}
