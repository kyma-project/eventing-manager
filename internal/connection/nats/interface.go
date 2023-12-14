package nats

import (
	natsio "github.com/nats-io/nats.go"
)

//go:generate go run github.com/vektra/mockery/v2 --name=Interface --structname=Connection --filename=connection.go
type Interface interface {
	// Connect connects to NATS and returns an error if it cannot connect.
	Connect() error

	// Disconnect disconnects the NATS connection.
	Disconnect()

	// RegisterReconnectHandlerIfNotRegistered registers a ReconnectHandler only if it was not registered before.
	RegisterReconnectHandlerIfNotRegistered(natsio.ConnHandler)

	// RegisterDisconnectErrHandlerIfNotRegistered registers a DisconnectErrHandler only if it was not registered before.
	RegisterDisconnectErrHandlerIfNotRegistered(natsio.ConnErrHandler)
}
