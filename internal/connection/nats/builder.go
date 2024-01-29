package nats

import (
	"strings"

	natsio "github.com/nats-io/nats.go"

	"github.com/kyma-project/eventing-manager/internal/connection/nats/errors"
)

type Builder interface {
	Build() Interface
}

type ConnectionBuilder struct {
	url  string
	opts []natsio.Option
}

func NewBuilder(url, name string, opts ...natsio.Option) (*ConnectionBuilder, error) {
	if len(strings.TrimSpace(url)) == 0 {
		return nil, errors.ErrEmptyConnectionURL
	}

	if len(strings.TrimSpace(name)) == 0 {
		return nil, errors.ErrEmptyConnectionName
	}

	opts = append(opts, natsio.Name(name)) // enforce configuring the connection name
	return &ConnectionBuilder{url: url, opts: opts}, nil
}

func (b *ConnectionBuilder) Build() Interface {
	return &connection{
		url:                            b.url,
		conn:                           nil,
		opts:                           b.opts,
		reconnectHandlerRegistered:     false,
		disconnectErrHandlerRegistered: false,
	}
}
