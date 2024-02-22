package label

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
)

func TestSelectorCreatedByEventingManager(t *testing.T) {
	// given
	tests := []struct {
		name string
		want labels.Selector
	}{
		{
			name: "should return the correct selector",
			want: labels.SelectorFromSet(
				map[string]string{
					"app.kubernetes.io/created-by": "eventing-manager",
				},
			),
		},
	}
	for _, tc := range tests {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			got := SelectorCreatedByEventingManager()

			// then
			require.Equal(t, testcase.want, got)
		})
	}
}
