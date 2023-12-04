package object

import (
	apigatewayv1beta1 "github.com/kyma-incubator/api-gateway/api/v1beta1"
)

// Option is a functional option for API objects builders.
type Option func(*apigatewayv1beta1.APIRule)
