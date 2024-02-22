package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_initNATSConnectionBuilder(t *testing.T) {
	natsConnectionBuilder, err := initNATSConnectionBuilder()
	require.NoError(t, err)
	require.NotNil(t, natsConnectionBuilder)
}
