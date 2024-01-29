package nats

import (
	"reflect"
	"testing"
	"time"

	natsio "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/eventing-manager/internal/connection/nats/errors"
)

func TestNewBuilder(t *testing.T) {
	t.Parallel()

	// given
	const (
		url                  = "url"
		name                 = "name"
		maxReconnects        = 10
		retryOnFailedConnect = true
		reconnectWait        = time.Minute
	)

	type args struct {
		url  string
		name string
		opts []natsio.Option
	}
	tests := []struct {
		name        string
		args        args
		wantBuilder *ConnectionBuilder
		wantErr     error
	}{
		{
			name: "should return an error if the URL is empty",
			args: args{
				url:  "",
				name: name,
				opts: nil,
			},
			wantBuilder: nil,
			wantErr:     errors.ErrEmptyConnectionURL,
		},
		{
			name: "should return an error if the name is empty",
			args: args{
				url:  url,
				name: "",
				opts: nil,
			},
			wantBuilder: nil,
			wantErr:     errors.ErrEmptyConnectionName,
		},
		{
			name: "should return a connection builder instance with no errors if there are no given options",
			args: args{
				url:  url,
				name: name,
				opts: nil, // no given options
			},
			wantBuilder: &ConnectionBuilder{
				url: url,
				opts: []natsio.Option{
					natsio.Name(name),
				},
			},
			wantErr: nil,
		},
		{
			name: "should return a connection builder instance with no errors if there are given options",
			args: args{
				url:  url,
				name: name,
				opts: []natsio.Option{
					natsio.MaxReconnects(maxReconnects),
					natsio.RetryOnFailedConnect(retryOnFailedConnect),
					natsio.ReconnectWait(reconnectWait),
				},
			},
			wantBuilder: &ConnectionBuilder{
				url: url,
				opts: []natsio.Option{
					natsio.Name(name),
					natsio.MaxReconnects(10),
					natsio.RetryOnFailedConnect(true),
					natsio.ReconnectWait(time.Minute),
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		ttc := tt

		t.Run(ttc.name, func(t *testing.T) {
			t.Parallel()

			// when
			gotBuilder, gotErr := NewBuilder(ttc.args.url, ttc.args.name, ttc.args.opts...)

			// then
			require.Equal(t, ttc.wantErr, gotErr)
			require.True(t, builderDeepEqual(t, ttc.wantBuilder, gotBuilder))
		})
	}
}

func TestConnectionBuilder_Build(t *testing.T) {
	b := ConnectionBuilder{
		url: "url",
		opts: []natsio.Option{
			natsio.Name("name"),
			natsio.MaxReconnects(10),
			natsio.RetryOnFailedConnect(true),
			natsio.ReconnectWait(time.Minute),
		},
	}
	require.NotNil(t, b.Build())
}

func builderDeepEqual(t *testing.T, a, b *ConnectionBuilder) bool {
	t.Helper()

	if a == b {
		return true
	}

	if a.url != b.url {
		return false
	}

	if len(a.opts) != len(b.opts) {
		return false
	}

	aOpts := &natsio.Options{}
	for _, opt := range a.opts {
		require.NoError(t, opt(aOpts))
	}

	bOpts := &natsio.Options{}
	for _, opt := range b.opts {
		require.NoError(t, opt(bOpts))
	}

	return reflect.DeepEqual(aOpts, bOpts)
}
