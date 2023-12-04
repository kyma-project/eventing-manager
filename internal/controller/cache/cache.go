package cache

import (
	kapps "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	kcore "k8s.io/api/core/v1"
	krbac "k8s.io/api/rbac/v1"
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
	// TODO(marcobebway) filter by label "app.kubernetes.io/created-by=eventing-manager" when it is released
	instanceEventing := fromLabelSelector(label.SelectorInstanceEventing())
	options.ByObject = map[client.Object]cache.ByObject{
		&kapps.Deployment{}:                      instanceEventing,
		&kcore.ServiceAccount{}:                  instanceEventing,
		&krbac.ClusterRole{}:                     instanceEventing,
		&krbac.ClusterRoleBinding{}:              instanceEventing,
		&autoscalingv1.HorizontalPodAutoscaler{}: instanceEventing,
	}
	return options
}

func fromLabelSelector(selector labels.Selector) cache.ByObject {
	return cache.ByObject{Label: selector}
}
