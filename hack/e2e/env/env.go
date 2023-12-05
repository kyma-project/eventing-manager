package env

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"

	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
)

// E2EConfig represents the environment config for the end-to-end tests for eventing-manager.
type E2EConfig struct {
	BackendType           string `default:"NATS"                                                             envconfig:"BACKEND_TYPE"` // NATS or EventMesh
	ManagerImage          string `default:""                                                                 envconfig:"MANAGER_IMAGE"`
	EventTypePrefix       string `default:"sap.kyma.custom"                                                  envconfig:"EVENT_TYPE_PREFIX"`
	EventMeshNamespace    string `default:"/default/sap.kyma/tunas-develop"                                  envconfig:"EVENTMESH_NAMESPACE"`
	SubscriptionSinkImage string `default:"ghcr.io/kyma-project/eventing-manager/e2e-tests-sink:sha-8e81aae" envconfig:"SUBSCRIPTION_SINK_IMAGE"`
	SubscriptionSinkName  string `default:"test-sink"                                                        envconfig:"SUBSCRIPTION_SINK_Name"`
	SubscriptionSinkURL   string `default:""                                                                 envconfig:"SUBSCRIPTION_SINK_URL"`
	TestNamespace         string `default:"eventing-tests"                                                   envconfig:"TEST_NAMESPACE"`
	PublisherURL          string `default:"http://localhost:38081"                                           envconfig:"PUBLISHER_URL"`
	SinkPortForwardedURL  string `default:"http://localhost:38071"                                           envconfig:"SINK_PORT_FORWARDED_URL"`
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
