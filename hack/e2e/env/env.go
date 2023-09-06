package env

import (
	"github.com/kelseyhightower/envconfig"
	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
)

// E2EConfig represents the environment config for the end-to-end tests for eventing-manager.
type E2EConfig struct {
	EventTypePrefix string `envconfig:"EVENT_TYPE_PREFIX" default:"sap.kyma.custom"`
	BackendType     string `envconfig:"BACKEND_TYPE" default:"NATS"` // NATS or EventMesh
	ManagerImage    string `envconfig:"MANAGER_IMAGE" default:""`
}

func (cfg E2EConfig) IsNATSBackend() bool {
	return eventingv1alpha1.BackendType(cfg.BackendType) == eventingv1alpha1.NatsBackendType
}

func (cfg E2EConfig) IsEventMeshBackend() bool {
	return eventingv1alpha1.BackendType(cfg.BackendType) == eventingv1alpha1.EventMeshBackendType
}

func GetE2EConfig() (E2EConfig, error) {
	cfg := E2EConfig{}
	if err := envconfig.Process("", &cfg); err != nil {
		return E2EConfig{}, err
	}
	return cfg, nil
}