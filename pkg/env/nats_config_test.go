package env

import (
	"strings"
	"testing"

	"github.com/kyma-project/eventing-manager/test/utils"
	"github.com/stretchr/testify/require"
)

func Test_ToECENVNATSConfig(t *testing.T) {
	// given
	givenConfig := NATSConfig{
		URL:                     "http://eventing-nats.svc.cluster.local",
		MaxReconnects:           10,
		ReconnectWait:           100,
		EventTypePrefix:         "sap.kyma.custom",
		MaxIdleConns:            5,
		MaxConnsPerHost:         10,
		MaxIdleConnsPerHost:     10,
		IdleConnTimeout:         100,
		JSStreamName:            "kyma",
		JSSubjectPrefix:         "kyma",
		JSStreamStorageType:     "File",
		JSStreamReplicas:        3,
		JSStreamRetentionPolicy: "Interest",
		JSStreamMaxMessages:     100000,
		JSStreamMaxBytes:        "700Mi",
		JSStreamMaxMsgsPerTopic: 10000,
		JSStreamDiscardPolicy:   "DiscardNew",
		JSConsumerDeliverPolicy: "DeliverNew",
	}

	givenEventing := utils.NewEventingCR(
		utils.WithEventingStreamData("Memory", "650M", 5, 5000),
		utils.WithEventingEventTypePrefix("sap.kyma.custom"),
	)

	// when
	result := givenConfig.ToECENVNATSConfig(*givenEventing)

	// then
	// check values from local NATSConfig.
	require.Equal(t, givenConfig.URL, result.URL)
	require.Equal(t, givenConfig.MaxReconnects, result.MaxReconnects)
	require.Equal(t, givenConfig.ReconnectWait, result.ReconnectWait)
	require.Equal(t, givenConfig.MaxIdleConns, result.MaxIdleConns)
	require.Equal(t, givenConfig.MaxConnsPerHost, result.MaxConnsPerHost)
	require.Equal(t, givenConfig.MaxIdleConnsPerHost, result.MaxIdleConnsPerHost)
	require.Equal(t, givenConfig.IdleConnTimeout, result.IdleConnTimeout)
	require.Equal(t, givenConfig.JSStreamName, result.JSStreamName)
	require.Equal(t, givenConfig.JSSubjectPrefix, result.JSSubjectPrefix)
	require.Equal(t, givenConfig.JSStreamRetentionPolicy, result.JSStreamRetentionPolicy)
	require.Equal(t, givenConfig.JSStreamMaxMessages, result.JSStreamMaxMessages)
	require.Equal(t, givenConfig.JSStreamDiscardPolicy, result.JSStreamDiscardPolicy)
	require.Equal(t, givenConfig.JSConsumerDeliverPolicy, result.JSConsumerDeliverPolicy)

	// check values from eventing CR.
	require.Equal(t, givenEventing.Spec.Backends[0].Config.EventTypePrefix, result.EventTypePrefix)
	require.Equal(t, strings.ToLower(givenEventing.Spec.Backends[0].Config.NATSStreamStorageType), result.JSStreamStorageType)
	require.Equal(t, givenEventing.Spec.Backends[0].Config.NATSStreamReplicas, result.JSStreamReplicas)
	require.Equal(t, givenEventing.Spec.Backends[0].Config.NATSStreamMaxSize.String(), result.JSStreamMaxBytes)
	require.Equal(t, int64(givenEventing.Spec.Backends[0].Config.NATSMaxMsgsPerTopic), result.JSStreamMaxMsgsPerTopic)
}
