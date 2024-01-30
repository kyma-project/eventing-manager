package jetstream_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/pkg/backend/jetstream"
	"github.com/kyma-project/eventing-manager/pkg/env"
	eventingtesting "github.com/kyma-project/eventing-manager/testing"
)

// Test_ConnectionBuilder_Build tests that the connection object is created and correctly configured.
func Test_ConnectionBuilder_Build(t *testing.T) {
	// SuT: ConnectionBuilder
	// UoW: Build()
	// test kind: state verification

	// given: a NATS server.
	natsServer := startManagedNATSServer(t)

	config := env.NATSConfig{URL: natsServer.ClientURL()}
	cb := jetstream.NewConnectionBuilder(config)

	// when: connection establishment should work.
	connection, err := cb.Build()

	// then: connection is configured correctly.
	require.NoError(t, err)
	require.NotNil(t, connection)

	// get the actual connection for state testing
	natsConn, ok := connection.(*nats.Conn)
	require.True(t, ok)

	// ensure the options are set
	assert.Equal(t, "Kyma Controller", natsConn.Opts.Name)
	assert.Equal(t, natsConn.Opts.MaxReconnect, config.MaxReconnects)
	assert.Equal(t, natsConn.Opts.ReconnectWait, config.ReconnectWait)
	assert.True(t, natsConn.Opts.RetryOnFailedConnect)
}

func Test_ConnectionBuilder_Build_ForErrConnect(t *testing.T) {
	// SuT: ConnectionBuilder
	// UoW: Build()
	// test kind: value
	t.Parallel()

	// given: a free port on localhost without a NATS server running.
	url := fixtureUnusedLocalhostURL(t)

	config := env.NATSConfig{URL: url}
	cb := jetstream.NewConnectionBuilder(config)

	// when: connection establishment should fail as
	// there is no NATS server to connect to on this URL.
	_, err := cb.Build()

	// then
	require.ErrorIs(t, err, jetstream.ErrConnect)
}

// Test_ConnectionBuilder_IsConnected ensures that the IsConnected method always returns the correct value.
func Test_ConnectionBuilder_IsConnected(t *testing.T) {
	// SuT: ConnectionBuilder
	// UoW: Build()
	// test kind: state verification

	// given
	natsServer := startManagedNATSServer(t)

	config := env.NATSConfig{URL: natsServer.ClientURL()}
	cb := jetstream.NewConnectionBuilder(config)

	// when: a NATS server is online
	connection, err := cb.Build()

	// then
	require.NoError(t, err)
	assert.True(t, connection.IsConnected())
	assert.NotNil(t, connection)

	// when: NATS server is offline
	natsServer.Shutdown()
	// then
	require.Eventually(t, func() bool {
		return !connection.IsConnected()
	}, 60*time.Second, 10*time.Millisecond)
}

// startManagedNATSServer starts a NATS server and shuts the server down as soon as the test is
// completed (also when it failed!).
func startManagedNATSServer(t *testing.T) *server.Server {
	t.Helper()
	natsServer, _, err := jetstream.StartNATSServer(eventingtesting.WithJetStreamEnabled())
	require.NoError(t, err)
	t.Cleanup(func() {
		natsServer.Shutdown()
	})
	return natsServer
}

// fixtureUnusedLocalhostUrl provides a localhost URL with an unused port.
func fixtureUnusedLocalhostURL(t *testing.T) string {
	t.Helper()
	port, err := eventingtesting.GetFreePort()
	require.NoError(t, err)
	url := fmt.Sprintf("http://localhost:%d", port)
	return url
}
