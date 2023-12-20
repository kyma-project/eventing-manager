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
	instanceEventing := fromLabelSelector(label.SelectorInstanceEventing())
	options.ByObject = map[client.Object]cache.ByObject{
		&kappsv1.Deployment{}:                     instanceEventing,
		&kcorev1.ServiceAccount{}:                 instanceEventing,
		&krbacv1.ClusterRole{}:                    instanceEventing,
		&krbacv1.ClusterRoleBinding{}:             instanceEventing,
		&kautoscalingv1.HorizontalPodAutoscaler{}: instanceEventing,
	}
	return options
}

func fromLabelSelector(selector labels.Selector) cache.ByObject {
	return cache.ByObject{Label: selector}
}
