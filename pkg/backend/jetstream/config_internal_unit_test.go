package jetstream

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/pkg/env"
)

func TestUnitValidate_For_Errors(t *testing.T) {
	testCases := []struct {
		name        string
		givenConfig env.NATSConfig
		wantError   error
	}{
		{
			name:        "ErrorEmptyStream",
			givenConfig: env.NATSConfig{JSStreamName: ""},
			wantError:   ErrEmptyStreamName,
		},
		{
			name:        "ErrorStreamToLong",
			givenConfig: env.NATSConfig{JSStreamName: fixtureStreamNameTooLong()},
			wantError:   ErrStreamNameTooLong,
		},
		{
			name: "ErrorStorageType",
			givenConfig: env.NATSConfig{
				JSStreamName:        "not-empty",
				JSStreamStorageType: "invalid-storage-type",
			},
			wantError: ErrInvalidStorageType.WithArg("invalid-storage-type"),
		},
		{
			name: "ErrorRetentionPolicy",
			givenConfig: env.NATSConfig{
				JSStreamName:            "not-empty",
				JSStreamStorageType:     StorageTypeMemory,
				JSStreamRetentionPolicy: "invalid-retention-policy",
			},
			wantError: ErrInvalidRetentionPolicy.WithArg("invalid-retention-policy"),
		},
		{
			name: "ErrorDiscardPolicy",
			givenConfig: env.NATSConfig{
				JSStreamName:            "not-empty",
				JSStreamStorageType:     StorageTypeMemory,
				JSStreamRetentionPolicy: RetentionPolicyInterest,
				JSStreamDiscardPolicy:   "invalid-discard-policy",
			},
			wantError: ErrInvalidDiscardPolicy.WithArg("invalid-discard-policy"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.givenConfig)
			require.ErrorIs(t, err, tc.wantError)
		})
	}
}

func fixtureStreamNameTooLong() string {
	b := strings.Builder{}
	for i := 0; i < (jsMaxStreamNameLength + 1); i++ {
		b.WriteString("a")
	}
	streamName := b.String()
	return streamName
}
