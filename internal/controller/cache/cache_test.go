package cache

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	kapps "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	kcore "k8s.io/api/core/v1"
	krbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_applySelectors(t *testing.T) {
	// given
	selector := cache.ByObject{
		Label: labels.SelectorFromSet(
			map[string]string{
				"app.kubernetes.io/instance": "eventing",
			},
		),
	}
	type args struct {
		options cache.Options
	}
	tests := []struct {
		name string
		args args
		want cache.Options
	}{
		{
			name: "should apply the correct selectors",
			args: args{
				options: cache.Options{},
			},
			want: cache.Options{
				ByObject: map[client.Object]cache.ByObject{
					&kapps.Deployment{}:                      selector,
					&kcore.ServiceAccount{}:                  selector,
					&krbac.ClusterRole{}:                     selector,
					&krbac.ClusterRoleBinding{}:              selector,
					&autoscalingv1.HorizontalPodAutoscaler{}: selector,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got := applySelectors(tt.args.options)

			// then
			require.True(t, deepEqualOptions(tt.want, got))
		})
	}
}

func deepEqualOptions(a, b cache.Options) bool {
	// we only care about the ByObject comparison
	return deepEqualByObject(a.ByObject, b.ByObject)
}

func deepEqualByObject(a, b map[client.Object]cache.ByObject) bool {
	if len(a) != len(b) {
		return false
	}

	aTypeMap := make(map[string]cache.ByObject, len(a))
	bTypeMap := make(map[string]cache.ByObject, len(a))
	computeTypeMap(a, aTypeMap)
	computeTypeMap(b, bTypeMap)
	return reflect.DeepEqual(aTypeMap, bTypeMap)
}

func computeTypeMap(byObjectMap map[client.Object]cache.ByObject, typeMap map[string]cache.ByObject) {
	keyOf := func(i interface{}) string { return fmt.Sprintf(">>> %T", i) }
	for k, v := range byObjectMap {
		if obj, ok := k.(*kapps.Deployment); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*kcore.ServiceAccount); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*krbac.ClusterRole); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*krbac.ClusterRoleBinding); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*autoscalingv1.HorizontalPodAutoscaler); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
	}
}

func Test_fromLabelSelector(t *testing.T) {
	// given
	type args struct {
		label labels.Selector
	}
	tests := []struct {
		name string
		args args
		want cache.ByObject
	}{
		{
			name: "should return the correct selector",
			args: args{
				label: labels.SelectorFromSet(map[string]string{"key": "value"}),
			},
			want: cache.ByObject{
				Label: labels.SelectorFromSet(map[string]string{"key": "value"}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got := fromLabelSelector(tt.args.label)

			// then
			require.Equal(t, tt.want, got)
		})
	}
}
