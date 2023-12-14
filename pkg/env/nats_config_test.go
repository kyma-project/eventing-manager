package env

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
)

func Test_GetNewNATSConfig(t *testing.T) {
	// given
	givenConfig := NATSConfig{
		URL:                     "http://eventing-nats.svc.cluster.local",
		MaxReconnects:           10,
		ReconnectWait:           100,
		MaxIdleConns:            5,
		MaxConnsPerHost:         10,
		MaxIdleConnsPerHost:     10,
		IdleConnTimeout:         100,
		JSStreamName:            "kyma",
		JSSubjectPrefix:         "kyma",
		JSStreamRetentionPolicy: "Interest",
		JSStreamMaxMessages:     100000,
		JSStreamDiscardPolicy:   "DiscardNew",
		JSConsumerDeliverPolicy: "DeliverNew",
	}

	givenEventing := &v1alpha1.Eventing{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Eventing",
			APIVersion: "operator.kyma-project.io/v1alpha1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      "test",
			Namespace: "test-ns",
			UID:       "1234-5678-1234-5678",
		},
		Spec: v1alpha1.EventingSpec{
			Backend: &v1alpha1.Backend{
				Type: v1alpha1.NatsBackendType,
				Config: v1alpha1.BackendConfig{
					EventTypePrefix:       "sap.kyma.custom",
					NATSStreamStorageType: "Memory",
					NATSStreamMaxSize:     resource.MustParse("650M"),
					NATSStreamReplicas:    5,
					NATSMaxMsgsPerTopic:   5000,
				},
			},
		},
	}

	// when
	result := givenConfig.GetNewNATSConfig(*givenEventing)

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
	require.Equal(t, givenEventing.Spec.Backend.Config.EventTypePrefix, result.EventTypePrefix)
	require.Equal(t, strings.ToLower(givenEventing.Spec.Backend.Config.NATSStreamStorageType), result.JSStreamStorageType)
	require.Equal(t, givenEventing.Spec.Backend.Config.NATSStreamReplicas, result.JSStreamReplicas)
	require.Equal(t, givenEventing.Spec.Backend.Config.NATSStreamMaxSize.String(), result.JSStreamMaxBytes)
	require.Equal(t, int64(givenEventing.Spec.Backend.Config.NATSMaxMsgsPerTopic), result.JSStreamMaxMsgsPerTopic)
}

func Test_GetNATSConfig(t *testing.T) {
	type args struct {
		maxReconnects int
		reconnectWait time.Duration
		envs          map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    NATSConfig
		wantErr bool
	}{
		{
			name: "Required values only gives valid config",
			args: args{
				envs: map[string]string{
					"NATS_URL":                 "natsurl",
					"JS_STREAM_NAME":           "jsn",
					"JS_STREAM_SUBJECT_PREFIX": "kma",
				},
				maxReconnects: 1,
				reconnectWait: 1 * time.Second,
			},
			want: NATSConfig{
				MaxReconnects:           1,
				ReconnectWait:           1 * time.Second,
				MaxIdleConns:            50,
				MaxConnsPerHost:         50,
				MaxIdleConnsPerHost:     50,
				IdleConnTimeout:         10 * time.Second,
				JSStreamName:            "jsn",
				JSSubjectPrefix:         "kma",
				JSStreamRetentionPolicy: "interest",
				JSStreamMaxMessages:     -1,
				JSConsumerDeliverPolicy: "new",
				JSStreamDiscardPolicy:   "new",
			},
			wantErr: false,
		},
		{
			name: "Envs are mapped correctly",
			args: args{
				envs: map[string]string{
					"JS_STREAM_NAME":             "jsn",
					"MAX_IDLE_CONNS":             "1",
					"MAX_CONNS_PER_HOST":         "2",
					"MAX_IDLE_CONNS_PER_HOST":    "3",
					"IDLE_CONN_TIMEOUT":          "1s",
					"JS_STREAM_RETENTION_POLICY": "jsrp",
					"JS_STREAM_MAX_MSGS":         "5",
					"JS_CONSUMER_DELIVER_POLICY": "jcdp",
					"JS_STREAM_DISCARD_POLICY":   "jsdp",
				},
				maxReconnects: 1,
				reconnectWait: 1 * time.Second,
			},
			want: NATSConfig{
				MaxReconnects:           1,
				ReconnectWait:           1 * time.Second,
				MaxIdleConns:            1,
				MaxConnsPerHost:         2,
				MaxIdleConnsPerHost:     3,
				IdleConnTimeout:         1 * time.Second,
				JSStreamName:            "jsn",
				JSStreamRetentionPolicy: "jsrp",
				JSStreamMaxMessages:     5,
				JSConsumerDeliverPolicy: "jcdp",
				JSStreamDiscardPolicy:   "jsdp",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store the current environ and restore it after this test.
			// A wrongly set up environment would break this test otherwise.
			env := os.Environ()
			t.Cleanup(func() {
				for _, e := range env {
					s := strings.Split(e, "=")
					if err := os.Setenv(s[0], s[1]); err != nil {
						t.Log(err)
					}
				}
			})

			// Clean the environment to make this test reliable.
			os.Clearenv()
			for k, v := range tt.args.envs {
				t.Setenv(k, v)
			}

			got, err := GetNATSConfig(tt.args.maxReconnects, tt.args.reconnectWait)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNATSConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNATSConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
