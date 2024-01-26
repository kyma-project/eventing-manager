package nats

import (
	natsio "github.com/nats-io/nats.go"

	natsconnectionerrors "github.com/kyma-project/eventing-manager/internal/connection/nats/errors"
)

// compile-time check.
var _ Interface = &connection{}

// connection represents a NATS connection.
type connection struct {
	url                            string
	conn                           *natsio.Conn
	opts                           []natsio.Option
	reconnectHandlerRegistered     bool
	disconnectErrHandlerRegistered bool
}

func (c *connection) Connect(connHandler natsio.ConnHandler, connErrHandler natsio.ConnErrHandler) error {
	if c.isConnected() {
		return nil
	}

	var err error
	if c.conn, err = natsio.Connect(c.url, c.opts...); err != nil {
		return err
	}

	// register handlers
	c.registerReconnectHandler(connHandler)
	c.registerDisconnectErrHandler(connErrHandler)

	if c.isConnected() {
		return nil
	}

	return natsconnectionerrors.ErrCannotConnect
}

func (c *connection) Disconnect() {
	if c.conn == nil || c.conn.IsClosed() {
		return
	}
	c.conn.Close()
}

func (c *connection) isConnected() bool {
	if c.conn == nil {
		return false
	}
	return c.conn.IsConnected()
}

func (c *connection) registerReconnectHandler(handler natsio.ConnHandler) {
	if c.conn == nil || c.reconnectHandlerRegistered {
		return
	}
	c.conn.SetReconnectHandler(handler)
	c.reconnectHandlerRegistered = true
}

func (c *connection) registerDisconnectErrHandler(handler natsio.ConnErrHandler) {
	if c.conn == nil || c.disconnectErrHandlerRegistered {
		return
	}
	c.conn.SetDisconnectErrHandler(handler)
	c.disconnectErrHandlerRegistered = true
}
