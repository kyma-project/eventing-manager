package cloudevent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/pkg/cloudevent"
)

func Test_NewHTTP(t *testing.T) {
	// SuT: ClientFactory
	// UoW: NewHTTP()
	// test kind: value

	cf := cloudevent.ClientFactory{}
	client, err := cf.NewHTTP()
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NotNil(t, client.Client)
}
