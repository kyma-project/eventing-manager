package env

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
)

// E2EConfig represents the environment config for the end-to-end tests for eventing-manager.
type E2EConfig struct {
	BackendType           string `envconfig:"BACKEND_TYPE" default:"NATS"` // NATS or EventMesh
	ManagerImage          string `envconfig:"MANAGER_IMAGE" default:""`
	EventTypePrefix       string `envconfig:"EVENT_TYPE_PREFIX" default:"sap.kyma.custom"`
	EventMeshNamespace    string `envconfig:"EVENTMESH_NAMESPACE" default:"/default/sap.kyma/tunas-develop"`
	SubscriptionSinkImage string `envconfig:"SUBSCRIPTION_SINK_IMAGE" default:"ghcr.io/kyma-project/eventing-manager/e2e-tests-sink:sha-8e81aae"`
	SubscriptionSinkName  string `envconfig:"SUBSCRIPTION_SINK_Name" default:"test-sink"`
	SubscriptionSinkURL   string `envconfig:"SUBSCRIPTION_SINK_URL" default:""`
	TestNamespace         string `envconfig:"TEST_NAMESPACE" default:"eventing-tests"`
	PublisherURL          string `envconfig:"PUBLISHER_URL" default:"http://localhost:38081"`
	SinkPortForwardedURL  string `envconfig:"SINK_PORT_FORWARDED_URL" default:"http://localhost:38071"`
}

func (cfg E2EConfig) IsNATSBackend() bool {
	return operatorv1alpha1.BackendType(cfg.BackendType) == operatorv1alpha1.NatsBackendType
}

func (cfg E2EConfig) IsEventMeshBackend() bool {
	return operatorv1alpha1.BackendType(cfg.BackendType) == operatorv1alpha1.EventMeshBackendType
}

func GetE2EConfig() (*E2EConfig, error) {
	cfg := E2EConfig{}
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	// set subscription sink URL if its empty.
	if cfg.SubscriptionSinkURL == "" {
		cfg.SubscriptionSinkURL = fmt.Sprintf("http://%s.%s.svc.cluster.local", cfg.SubscriptionSinkName, cfg.TestNamespace)
	}
	return &cfg, nil
}
