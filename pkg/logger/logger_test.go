package logger_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/pkg/logger"
)

func Test_Build(t *testing.T) {
	kymaLogger, err := logger.New("json", "warn")
	require.NoError(t, err)
	require.NotNil(t, kymaLogger)
}
