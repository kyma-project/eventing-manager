package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClearConditions(t *testing.T) {
	t.Parallel()

	// given
	givenEventingStatus := &EventingStatus{
		Conditions: []kmetav1.Condition{
			{
				Type: "NATS",
			},
			{
				Type: "EventMesh",
			},
		},
	}
	require.NotEmpty(t, givenEventingStatus.Conditions)

	// when
	givenEventingStatus.ClearConditions()

	// then
	require.Empty(t, givenEventingStatus.Conditions)
}
