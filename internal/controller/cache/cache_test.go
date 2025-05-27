package cache

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	kappsv1 "k8s.io/api/apps/v1"
	kautoscalingv1 "k8s.io/api/autoscaling/v1"
	kcorev1 "k8s.io/api/core/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_applySelectors(t *testing.T) {
	// given
	selector := cache.ByObject{
		Label: labels.SelectorFromSet(
			map[string]string{
				"app.kubernetes.io/created-by": "eventing-manager",
			},
		),
	}
	type args struct {
		options cache.Options
	}
	testCases := []struct {
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
					&kappsv1.Deployment{}:                     selector,
					&kcorev1.ServiceAccount{}:                 selector,
					&krbacv1.ClusterRole{}:                    selector,
					&krbacv1.ClusterRoleBinding{}:             selector,
					&kautoscalingv1.HorizontalPodAutoscaler{}: selector,
				},
			},
		},
	}
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			got := applySelectors(testcase.args.options)

			// then
			require.True(t, deepEqualOptions(testcase.want, got))
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
	keyOf := func(i any) string { return fmt.Sprintf(">>> %T", i) }
	for key, value := range byObjectMap {
		if obj, ok := key.(*kappsv1.Deployment); ok {
			k := keyOf(obj)
			typeMap[k] = value
		}
		if obj, ok := key.(*kcorev1.ServiceAccount); ok {
			k := keyOf(obj)
			typeMap[k] = value
		}
		if obj, ok := key.(*krbacv1.ClusterRole); ok {
			k := keyOf(obj)
			typeMap[k] = value
		}
		if obj, ok := key.(*krbacv1.ClusterRoleBinding); ok {
			k := keyOf(obj)
			typeMap[k] = value
		}
		if obj, ok := key.(*kautoscalingv1.HorizontalPodAutoscaler); ok {
			k := keyOf(obj)
			typeMap[k] = value
		}
	}
}

func Test_fromLabelSelector(t *testing.T) {
	// given
	type args struct {
		label labels.Selector
	}
	testCases := []struct {
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
	for _, tc := range testCases {
		testcase := tc
		t.Run(testcase.name, func(t *testing.T) {
			// when
			got := fromLabelSelector(testcase.args.label)

			// then
			require.Equal(t, testcase.want, got)
		})
	}
}
