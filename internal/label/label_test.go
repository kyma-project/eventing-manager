package label

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
)

func TestSelectorInstanceEventing(t *testing.T) {
	// given
	tests := []struct {
		name string
		want labels.Selector
	}{
		{
			name: "should return the correct selector",
			want: labels.SelectorFromSet(
				map[string]string{
					"app.kubernetes.io/instance": "eventing",
				},
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got := SelectorInstanceEventing()

			// then
			require.Equal(t, tt.want, got)
		})
	}
}
