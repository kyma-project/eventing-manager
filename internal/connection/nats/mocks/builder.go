package mocks

import (
	natsconnection "github.com/kyma-project/eventing-manager/internal/connection/nats"
)

type Builder struct {
	conn *Connection
}

func NewBuilder(conn *Connection) *Builder {
	return &Builder{conn: conn}
}

func (b *Builder) Build() natsconnection.Interface {
	return b.conn
}

func (b *Builder) SetConnection(conn *Connection) {
	b.conn = conn
}
