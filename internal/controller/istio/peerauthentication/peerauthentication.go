package peerauthentication

import (
	istiosecv1beta1 "istio.io/api/security/v1beta1"
	istiotypes "istio.io/api/type/v1beta1"
	istio "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventPublisherProxyMetrics returns the PeerAuthentication for the Event-Publisher-Proxy metrics endpoint.
func EventPublisherProxyMetrics(namespace string) *istio.PeerAuthentication {
	return &istio.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eventing-publisher-proxy-metrics",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":    "eventing-publisher-proxy",
				"app.kubernetes.io/version": "0.1.0",
			},
			OwnerReferences: nil, // todo
		},
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
func EventingManagerMetrics(namespace string) *istio.PeerAuthentication {
	return &istio.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eventing-manager-metrics",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":     "controller",
				"app.kubernetes.io/instance": "eventing",
			},
			OwnerReferences: nil, // todo
		},
		Spec: istiosecv1beta1.PeerAuthentication{
			Selector: &istiotypes.WorkloadSelector{MatchLabels: map[string]string{
				"app.kubernetes.io/name": "eventing-publisher-proxy",
			}},
			PortLevelMtls: map[uint32]*istiosecv1beta1.PeerAuthentication_MutualTLS{
				8080: {Mode: istiosecv1beta1.PeerAuthentication_MutualTLS_PERMISSIVE},
			},
		},
	}
}
