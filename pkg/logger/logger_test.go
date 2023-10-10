package logger_test

import (
	"testing"

	"github.com/kyma-project/eventing-manager/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func Test_Build(t *testing.T) {
	kymaLogger, err := logger.New("json", "warn")
	assert.NoError(t, err)
	assert.NotNil(t, kymaLogger)
}
