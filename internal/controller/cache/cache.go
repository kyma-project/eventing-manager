package cache

import (
	kappsv1 "k8s.io/api/apps/v1"
	kautoscalingv1 "k8s.io/api/autoscaling/v1"
	kcorev1 "k8s.io/api/core/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/eventing-manager/internal/label"
)

func New(config *rest.Config, options cache.Options) (cache.Cache, error) {
	return cache.New(config, applySelectors(options))
}

// applySelectors applies label selectors to runtime objects created by the EventingManager.
func applySelectors(options cache.Options) cache.Options {
	createdByEventingManager := fromLabelSelector(label.SelectorCreatedByEventingManager())
	options.ByObject = map[client.Object]cache.ByObject{
		&kappsv1.Deployment{}:                     createdByEventingManager,
		&kcorev1.ServiceAccount{}:                 createdByEventingManager,
		&krbacv1.ClusterRole{}:                    createdByEventingManager,
		&krbacv1.ClusterRoleBinding{}:             createdByEventingManager,
		&kautoscalingv1.HorizontalPodAutoscaler{}: createdByEventingManager,
	}
	return options
}

func fromLabelSelector(selector labels.Selector) cache.ByObject {
	return cache.ByObject{Label: selector}
}
