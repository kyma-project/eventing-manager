//go:build e2e
// +build e2e

// Package cleanup-test is part of the end-to-end-tests. This package contains tests that evaluate the deletion of
// Eventing CRs and the cascading deletion of all correlated Kubernetes resources.
// To run the tests a k8s cluster and an Eventing-CR need to be available and configured. For this reason, the tests are
// seperated via the 'e2e' buildtags. For more information please consult the readme.
package cleanup_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/hack/e2e/env"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.uber.org/zap"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/kyma-project/eventing-manager/hack/e2e/common"
	. "github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
)

// Constants for retries.
const (
	interval = 2 * time.Second
	attempts = 60
)

// clientSet is what is used to access K8s build-in resources like Pods, Namespaces and so on.
var clientSet *kubernetes.Clientset //nolint:gochecknoglobals // This will only be accessible in e2e tests.

// k8sClient is what is used to access the Eventing CR.
var k8sClient client.Client //nolint:gochecknoglobals // This will only be accessible in e2e tests.

var logger *zap.Logger

var testConfigs env.E2EConfig

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	var err error
	logger, err = SetupLogger()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	testConfigs, err = env.GetE2EConfig()
	if err != nil {
		logger.Error(err.Error())
		panic(err)

	}
	logger.Info(fmt.Sprintf("##### NOTE: Tests will run w.r.t. backend: %s", testConfigs.BackendType))

	clientSet, k8sClient, err = GetK8sClients()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	ctx := context.TODO()
	// Delete the Eventing CR used for testing.
	err = Retry(attempts, interval, func() error {
		return k8sClient.Delete(ctx, EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType)))
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

func Test_NoEventingCRExists(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	err := Retry(attempts, interval, func() error {
		_, crErr := getEventingCR(ctx, CRName, NamespaceName)
		// This is what we want here.
		if k8serrors.IsNotFound(crErr) {
			return nil
		}
		// All other errors are unexpected here.
		if crErr != nil {
			return crErr
		}
		// If we still find the CR we will return an error.
		return errors.New("found Eventing CR, but wanted the Eventing CR to be deleted")
	})
	require.NoError(t, err)
}

// Test_NoPublisherServiceAccountExists tests if the publisher-proxy ServiceAccount was deleted.
func Test_NoPublisherServiceAccountExists(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		_, getErr := clientSet.CoreV1().ServiceAccounts(NamespaceName).Get(ctx,
			eventing.GetPublisherServiceAccountName(*eventingCR), metav1.GetOptions{})
		if getErr == nil {
			return errors.New("PublisherServiceAccount should have been deleted")
		}
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Enable this test once https://github.com/kyma-project/eventing-manager/issues/34 is done!
//// Test_NoPublisherClusterRoleExists tests if the publisher-proxy ClusterRole was deleted.
//func Test_NoPublisherClusterRoleExists(t *testing.T) {
//	t.Parallel()
//	ctx := context.TODO()
//	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
//	err := Retry(attempts, interval, func() error {
//		_, getErr := clientSet.RbacV1().ClusterRoles().Get(ctx,
//			eventing.GetPublisherClusterRoleName(*eventingCR), metav1.GetOptions{})
//		if getErr == nil {
//			return errors.New("PublisherClusterRole should have been deleted")
//		}
//		if !k8serrors.IsNotFound(getErr) {
//			return getErr
//		}
//		return nil
//	})
//	require.NoError(t, err)
//}

// Enable this test once https://github.com/kyma-project/eventing-manager/issues/34 is done!
//// Test_NoPublisherClusterRoleBindingExists tests if the publisher-proxy ClusterRoleBinding was deleted.
//func Test_NoPublisherClusterRoleBindingExists(t *testing.T) {
//	t.Parallel()
//	ctx := context.TODO()
//	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
//	err := Retry(attempts, interval, func() error {
//		_, getErr := clientSet.RbacV1().ClusterRoleBindings().Get(ctx,
//			eventing.GetPublisherClusterRoleBindingName(*eventingCR), metav1.GetOptions{})
//		if getErr == nil {
//			return errors.New("PublisherClusterRoleBinding should have been deleted")
//		}
//		if !k8serrors.IsNotFound(getErr) {
//			return getErr
//		}
//		return nil
//	})
//	require.NoError(t, err)
//}

// Test_NoPublisherServicesExists tests if the publisher-proxy Services was deleted.
func Test_NoPublisherServicesExists(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		// check service to expose event publishing endpoint, was deleted.
		_, getErr := clientSet.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherPublishServiceName(*eventingCR), metav1.GetOptions{})
		if getErr == nil {
			return errors.New("Publisher PublishService should have been deleted")
		}
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}

		// check service to expose metrics endpoint, was deleted.
		_, getErr = clientSet.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherMetricsServiceName(*eventingCR), metav1.GetOptions{})
		if getErr == nil {
			return errors.New("Publisher MetricsService should have been deleted")
		}
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}

		// check service to expose health endpoint, was deleted.
		_, getErr = clientSet.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherHealthServiceName(*eventingCR), metav1.GetOptions{})
		if getErr == nil {
			return errors.New("Publisher HealthService should have been deleted")
		}
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherHPA tests the publisher-proxy HorizontalPodAutoscaler was deleted.
func Test_NoPublisherHPAExists(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		_, getErr := clientSet.AutoscalingV2().HorizontalPodAutoscalers(NamespaceName).Get(ctx,
			eventing.GetPublisherDeploymentName(*eventingCR), metav1.GetOptions{})
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherProxyDeployment checks the publisher-proxy deployment was deleted.
func Test_NoPublisherProxyDeploymentExists(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		_, getErr := clientSet.AppsV1().Deployments(NamespaceName).Get(ctx,
			eventing.GetPublisherDeploymentName(*eventingCR), metav1.GetOptions{})
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

func getEventingCR(ctx context.Context, name, namespace string) (*eventingv1alpha1.Eventing, error) {
	var eventingCR eventingv1alpha1.Eventing
	err := k8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &eventingCR)
	return &eventingCR, err
}
