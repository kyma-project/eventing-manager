package client

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_disableCacheForObjects(t *testing.T) {
	// given
	type args struct {
		options client.Options
	}
	tests := []struct {
		name string
		args args
		want client.Options
	}{
		{
			name: "should disable cache for the correct objects",
			args: args{
				options: client.Options{},
			},
			want: client.Options{
				Cache: &client.CacheOptions{
					DisableFor: []client.Object{
						&kcorev1.Secret{},
						&kcorev1.Service{},
						&kcorev1.ConfigMap{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got := disableCacheForObjects(tt.args.options)

			// then
			require.True(t, deepEqualOptions(tt.want, got))
		})
	}
}

func deepEqualOptions(a, b client.Options) bool {
	// we only care about the Cache comparison
	return deepEqualCacheOptions(a.Cache, b.Cache)
}

func deepEqualCacheOptions(a, b *client.CacheOptions) bool {
	if a == b {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	// we only care about the DisableFor comparison
	if len(a.DisableFor) != len(b.DisableFor) {
		return false
	}

	aTypeMap := make(map[string]interface{}, len(a.DisableFor))
	bTypeMap := make(map[string]interface{}, len(a.DisableFor))
	computeDisableForMap(a, aTypeMap)
	computeDisableForMap(b, bTypeMap)
	return reflect.DeepEqual(aTypeMap, bTypeMap)
}

func computeDisableForMap(cacheOptions *client.CacheOptions, disableForMap map[string]interface{}) {
	keyOf := func(i interface{}) string { return fmt.Sprintf(">>> %T", i) }
	for _, obj := range cacheOptions.DisableFor {
		if obj, ok := obj.(*kcorev1.Secret); ok {
			key := keyOf(obj)
			disableForMap[key] = nil
		}
		if obj, ok := obj.(*kcorev1.Service); ok {
			key := keyOf(obj)
			disableForMap[key] = nil
		}
		if obj, ok := obj.(*kcorev1.ConfigMap); ok {
			key := keyOf(obj)
			disableForMap[key] = nil
		}
	}
}
