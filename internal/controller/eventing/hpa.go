package eventing

import (
	"context"
	"fmt"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// createOrUpdateHorizontalPodAutoscaler creates or updates the HPA for the given deployment.
func (r *Reconciler) createOrUpdateHPA(ctx context.Context, deployment *v1.Deployment, eventing *eventingv1alpha1.Eventing, cpuUtilization, memoryUtilization int32) error {
	// try to get the existing horizontal pod autoscaler object
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, hpa)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to get horizontal pod autoscaler: %v", err)
	}
	min := int32(eventing.Spec.Publisher.Min)
	max := int32(eventing.Spec.Publisher.Max)
	hpa = createHorizontalPodAutoscaler(deployment, min, max, cpuUtilization, memoryUtilization)
	if err := controllerutil.SetControllerReference(eventing, hpa, r.Scheme()); err != nil {
		return err
	}
	// if the horizontal pod autoscaler object does not exist, create it
	if errors.IsNotFound(err) {
		// create a new horizontal pod autoscaler object
		err = r.Client.Create(ctx, hpa)
		if err != nil {
			return fmt.Errorf("failed to create horizontal pod autoscaler: %v", err)
		}
		return nil
	}

	// if the horizontal pod autoscaler object exists, update it
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err = r.Client.Update(ctx, hpa)
		if err != nil {
			return fmt.Errorf("failed to update horizontal pod autoscaler: %v", err)
		}
		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("failed to update horizontal pod autoscaler: %v", retryErr)
	}

	return nil
}

func createHorizontalPodAutoscaler(deployment *v1.Deployment, min int32, max int32, cpuUtilization, memoryUtilization int32) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       deployment.Name,
				APIVersion: "apps/v1",
			},
			MinReplicas: &min,
			MaxReplicas: max,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "cpu",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpuUtilization,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "memory",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &memoryUtilization,
						},
					},
				},
			},
		},
	}
}
