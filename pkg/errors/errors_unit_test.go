//nolint:err113 // no need to wrap errors in the test for the error package
package errors_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	emerrors "github.com/kyma-project/eventing-manager/pkg/errors"
)

var errInvalidStorageType = emerrors.NewArgumentError("invalid stream storage type: %q")

func ExampleMakeError() {
	actualError := errors.New("failed to connect to NATS")
	underlyingError := errors.New("some error coming from the NATS package")
	fmt.Println(emerrors.MakeError(actualError, underlyingError))
	// Output: failed to connect to NATS: some error coming from the NATS package
}

func ExampleArgumentError() {
	e := errInvalidStorageType
	e = e.WithArg("some storage type")

	fmt.Println(e.Error())
	// Output: invalid stream storage type: "some storage type"
}

func Test_ArgumentError_Is(t *testing.T) {
	t.Parallel()

	const givenStorageType = "some storage type"

	testCases := []struct {
		name       string
		givenError func() error
		wantIsTrue bool
	}{
		{
			name: "with argument",
			givenError: func() error {
				return errInvalidStorageType.WithArg(givenStorageType)
			},
			wantIsTrue: true,
		},
		{
			name: "with argument and wrapped",
			givenError: func() error {
				e := errInvalidStorageType.WithArg(givenStorageType)
				return fmt.Errorf("%w: %w", errors.New("new error"), e)
			},
			wantIsTrue: true,
		},
		{
			name: "without argument",
			givenError: func() error {
				return errInvalidStorageType
			},
			wantIsTrue: true,
		},
		{
			name: "completely different error",
			givenError: func() error {
				return errors.New("unknown error")
			},
			wantIsTrue: false,
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// when
			ok := errors.Is(testcase.givenError(), errInvalidStorageType)

			// then
			assert.Equal(t, testcase.wantIsTrue, ok)
		})
	}
}
