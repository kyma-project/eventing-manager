package validator

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSinkValidator(t *testing.T) {
	// given
	const (
		namespaceName = "test-namespace"
		serviceName   = "test-service"
	)

	ctx := context.Background()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	validator := NewSinkValidator(fakeClient)

	tests := []struct {
		name         string
		givenSink    string
		givenService *kcorev1.Service
		wantErr      error
	}{
		{
			name:         "With empty sink",
			givenSink:    "",
			givenService: nil,
			wantErr:      ErrSinkValidationFailed,
		},
		{
			name:         "With invalid sink URL",
			givenSink:    "[:invalid:url:]",
			givenService: nil,
			wantErr:      ErrSinkValidationFailed,
		},
		{
			name:         "With invalid sink format",
			givenSink:    "http://insuffecient",
			givenService: nil,
			wantErr:      ErrSinkValidationFailed,
		},
		{
			name:         "With non-existing service",
			givenSink:    fmt.Sprintf("https://%s.%s", serviceName, namespaceName),
			givenService: nil,
			wantErr:      ErrSinkValidationFailed,
		},
		{
			name:      "With existing service",
			givenSink: fmt.Sprintf("https://%s.%s", serviceName, namespaceName),
			givenService: &kcorev1.Service{
				ObjectMeta: kmetav1.ObjectMeta{Name: serviceName, Namespace: namespaceName},
			},
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create service if needed
			if test.givenService != nil {
				require.NoError(t, fakeClient.Create(ctx, test.givenService))
			}

			// when
			gotErr := validator.Validate(ctx, test.givenSink)

			// then
			if test.wantErr == nil {
				require.NoError(t, gotErr)
			} else {
				require.ErrorIs(t, gotErr, ErrSinkValidationFailed)
			}
		})
	}
}
