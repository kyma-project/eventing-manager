package nats

import (
	natsio "github.com/nats-io/nats.go"
)

//go:generate go run github.com/vektra/mockery/v2 --name=Interface --structname=Connection --filename=connection.go
type Interface interface {
	// Connect connects to NATS and returns an error if it cannot connect.
	// It also registers both the connect and disconnect handlers.
	Connect(handler natsio.ConnHandler, errorHandler natsio.ConnErrHandler) error

	// Disconnect disconnects the NATS connection.
	Disconnect()
}
