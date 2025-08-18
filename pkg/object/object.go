package object

import (
	apigatewayv2 "github.com/kyma-project/api-gateway/apis/gateway/v2"
	istiosecurityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
)

// Option is a functional option for API objects builders.
type Option func(*apigatewayv2.APIRule)
type AuthorizationPolicyOption func(*istiosecurityv1beta1.AuthorizationPolicy)
