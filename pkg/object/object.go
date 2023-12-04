package object

import (
	apigateway "github.com/kyma-incubator/api-gateway/api/v1beta1"
)

// Option is a functional option for API objects builders.
type Option func(*apigateway.APIRule)
