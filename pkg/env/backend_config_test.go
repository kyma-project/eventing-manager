package env

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_GetBackendConfig(t *testing.T) {
	g := NewGomegaWithT(t)
	envs := map[string]string{
		// optional
		"PUBLISHER_REQUEST_TIMEOUT": "10s",
	}

	for k, v := range envs {
		t.Setenv(k, v)
	}
	backendConfig := GetBackendConfig()
	// Ensure optional variables can be set
	g.Expect(backendConfig.PublisherConfig.RequestTimeout).To(Equal(envs["PUBLISHER_REQUEST_TIMEOUT"]))
}
