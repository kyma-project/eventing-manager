package k8s

const (
	// ApplicationCrdName defines the CRD name for Application of application-connector module.
	ApplicationCrdName string = "applications.applicationconnector.kyma-project.io"
	// ApplicationKind defines the Kind name for Application of application-connector module.
	ApplicationKind string = "Application"
	// ApplicationAPIVersion defines the API version for Application of application-connector module.
	ApplicationAPIVersion string = "applicationconnector.kyma-project.io/v1alpha1"

	// APIRuleCrdName defines the CRD name for APIRule of kyma api-gateway module.
	APIRuleCrdName string = "apirules.gateway.kyma-project.io"
	// PeerAuthenticationCRDName is the name of the Istio peer authentication CRD.
	PeerAuthenticationCRDName = "peerauthentications.security.istio.io"
)
