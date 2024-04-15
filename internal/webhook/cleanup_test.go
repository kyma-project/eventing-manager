package webhook

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_appendIfError(t *testing.T) {
	// given
	var errList []error

	// when
	errList = appendIfError(errList, nil)
	errList = appendIfError(errList, nil)
	// then
	require.Empty(t, errList)

	// when
	errList = appendIfError(errList, errors.New("error1")) //nolint:goerr113
	errList = appendIfError(errList, errors.New("error2")) //nolint:goerr113
	// then
	require.Len(t, errList, 2)
}
