package peerauthentication

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/utils/ptr"

	"github.com/pkg/errors"
	istiosecv1beta1 "istio.io/api/security/v1beta1"
	istiotypes "istio.io/api/type/v1beta1"
	istio "istio.io/client-go/pkg/apis/security/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/eventing-manager/pkg/k8s"
)

func SyncPeerAuthentications(ctx context.Context, kubeClient k8s.Client, log *zap.SugaredLogger) error {
	// Only attempt to create PAs if the corresponding CRD exists on the cluster.
	crdExists, err := kubeClient.PeerAuthenticationCRDExists(ctx)
	if err != nil {
		return errors.Wrap(err, "error while fetching PeerAuthentication CRD")
	}

	if !crdExists {
		log.Infof("PeerAuthentication CRD not found! Skipping creation of PeerAuthentication...")
		return nil
	}

	log.Infof("PeerAuthentication CRD found!")
	// Get the eventing Deployment for the OwnerReference.
	deploy, err := kubeClient.GetDeploymentDynamic(ctx, "eventing-manager", "kyma-system")
	if err != nil {
		return errors.Wrap(err, "error while fetching eventing Deployment")
	}
	if deploy == nil {
		return errors.New("eventing-manager deployment not found")
	}
	// create PeerAuthentications.
	for _, pa := range []*istio.PeerAuthentication{
		EventingManagerMetrics(deploy.Namespace, ownerReferences(*deploy)),
		EventPublisherProxyMetrics(deploy.Namespace, ownerReferences(*deploy)),
	} {
		if err = kubeClient.PatchApplyPeerAuthentication(ctx, pa); err != nil {
			return errors.Wrap(err, "failed to patchApply PeerAuthentication")
		}
		log.Infof("patch applied PeerAuthentication: %s in Namespace: %s", pa.Name, pa.Namespace)
	}
	return nil
}

// EventPublisherProxyMetrics returns the PeerAuthentication for the Event-Publisher-Proxy metrics endpoint.
func EventPublisherProxyMetrics(namespace string, ref []kmeta.OwnerReference) *istio.PeerAuthentication {
	return &istio.PeerAuthentication{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      "eventing-publisher-proxy-metrics",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    "eventing-publisher-proxy",
				"app.kubernetes.io/version": "0.1.0",
			},
			OwnerReferences: ref,
		},
		TypeMeta: typeMeta(),
		Spec: istiosecv1beta1.PeerAuthentication{
			Selector: &istiotypes.WorkloadSelector{MatchLabels: map[string]string{
				"app.kubernetes.io/name": "eventing-publisher-proxy",
			}},
			PortLevelMtls: map[uint32]*istiosecv1beta1.PeerAuthentication_MutualTLS{
				9090: {Mode: istiosecv1beta1.PeerAuthentication_MutualTLS_PERMISSIVE},
			},
		},
	}
}

// EventingManagerMetrics returns the PeerAuthentication for the Eventing-Manager metrics endpoint.
func EventingManagerMetrics(namespace string, ref []kmeta.OwnerReference) *istio.PeerAuthentication {
	return &istio.PeerAuthentication{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      "eventing-manager-metrics",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":     "controller",
				"app.kubernetes.io/instance": "eventing",
			},
			OwnerReferences: ref,
		},
		TypeMeta: typeMeta(),
		Spec: istiosecv1beta1.PeerAuthentication{
			Selector: &istiotypes.WorkloadSelector{MatchLabels: map[string]string{
				"app.kubernetes.io/name":     "eventing-manager",
				"app.kubernetes.io/instance": "eventing-manager",
			}},
			PortLevelMtls: map[uint32]*istiosecv1beta1.PeerAuthentication_MutualTLS{
				8080: {Mode: istiosecv1beta1.PeerAuthentication_MutualTLS_PERMISSIVE},
			},
		},
	}
}

func typeMeta() kmeta.TypeMeta {
	return kmeta.TypeMeta{
		Kind:       "PeerAuthentication",
		APIVersion: "security.istio.io/v1beta1",
	}
}

func ownerReferences(deploy kapps.Deployment) []kmeta.OwnerReference {
	return []kmeta.OwnerReference{
		{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			Name:               deploy.Name,
			UID:                deploy.UID,
			Controller:         ptr.To(true),
			BlockOwnerDeletion: ptr.To(false),
		},
	}
}
