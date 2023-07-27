package k8s

const (
	// PeerAuthenticationCrdName defines the CRD name for PeerAuthentication as in Istio.
	PeerAuthenticationCrdName string = "peerauthentications.security.istio.io"
	// PeerAuthenticationKind defines the Kind name for PeerAuthentication as in Istio.
	PeerAuthenticationKind string = "PeerAuthentication"
	// PeerAuthenticationAPIVersion defines the API version for PeerAuthentication as in Istio.
	PeerAuthenticationAPIVersion string = "security.istio.io/v1beta1"
)
