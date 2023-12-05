package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kyma-project/eventing-manager/pkg/logger"
)

func Test_Build(t *testing.T) {
	kymaLogger, err := logger.New("json", "warn")
	assert.NoError(t, err)
	assert.NotNil(t, kymaLogger)
}
