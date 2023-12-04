package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClearConditions(t *testing.T) {
	t.Parallel()

	// given
	givenEventingStatus := &EventingStatus{
		Conditions: []kmeta.Condition{
			{
				Type: "NATS",
			},
			{
				Type: "EventMesh",
			},
		},
	}
	require.NotEqual(t, 0, len(givenEventingStatus.Conditions))

	// when
	givenEventingStatus.ClearConditions()

	// then
	require.Len(t, givenEventingStatus.Conditions, 0)
}
