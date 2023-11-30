package label

import (
	"k8s.io/apimachinery/pkg/labels"
)

const (
	KeyComponent = "app.kubernetes.io/component"
	KeyCreatedBy = "app.kubernetes.io/created-by"
	KeyInstance  = "app.kubernetes.io/instance"
	KeyManagedBy = "app.kubernetes.io/managed-by"
	KeyName      = "app.kubernetes.io/name"
	KeyPartOf    = "app.kubernetes.io/part-of"
	KeyBackend   = "eventing.kyma-project.io/backend"
	KeyDashboard = "kyma-project.io/dashboard"

	ValueEventingPublisherProxy = "eventing-publisher-proxy"
	ValueEventingManager        = "eventing-manager"
	ValueEventing               = "eventing"
)

func SelectorCreatedByEventingManager() labels.Selector {
	return labels.SelectorFromSet(map[string]string{KeyCreatedBy: ValueEventingManager})
}
