package errors_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	controllererrors "github.com/kyma-project/eventing-manager/internal/controller/errors"
)

var ErrGeneric = errors.New("some error")

func Test_NewSkippable(t *testing.T) {
	testCases := []struct {
		error error
	}{
		{error: controllererrors.NewSkippable(nil)},
		{error: controllererrors.NewSkippable(controllererrors.NewSkippable(nil))},
		{error: controllererrors.NewSkippable(ErrGeneric)},
		{error: controllererrors.NewSkippable(controllererrors.NewSkippable(ErrGeneric))},
	}

	for _, tc := range testCases {
		skippableErr := controllererrors.NewSkippable(tc.error)
		if skippableErr == nil {
			t.Errorf("test NewSkippable retuned nil error")
			continue
		}
		require.ErrorIs(t, skippableErr, tc.error)
	}
}

func Test_IsSkippable(t *testing.T) {
	testCases := []struct {
		name          string
		givenError    error
		wantSkippable bool
	}{
		{
			name:          "nil error, should be skipped",
			givenError:    nil,
			wantSkippable: true,
		},
		{
			name:          "skippable error, should be skipped",
			givenError:    controllererrors.NewSkippable(ErrGeneric),
			wantSkippable: true,
		},
		{
			name:          "not-skippable error, should not be skipped",
			givenError:    ErrGeneric,
			wantSkippable: false,
		},
		{
			name:          "not-skippable error which wraps a skippable error, should not be skipped",
			givenError:    fmt.Errorf("some error %w", controllererrors.NewSkippable(ErrGeneric)),
			wantSkippable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if gotSkippable := controllererrors.IsSkippable(tc.givenError); tc.wantSkippable != gotSkippable {
				t.Errorf("test skippable failed, want: %v but got: %v", tc.wantSkippable, gotSkippable)
			}
		})
	}
}
