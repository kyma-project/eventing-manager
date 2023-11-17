//go:build e2e
// +build e2e

// Package cleanup-test is part of the end-to-end-tests. This package contains tests that evaluate the deletion of
// Eventing CRs and the cascading deletion of all correlated Kubernetes resources.
// To run the tests a k8s cluster and an Eventing-CR need to be available and configured. For this reason, the tests are
// seperated via the 'e2e' buildtags. For more information please consult the readme.
package cleanup_test

import (
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/batch/v1alpha1"
	"github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"
	"github.com/kyma-project/eventing-manager/pkg/eventing"

	. "github.com/kyma-project/eventing-manager/hack/e2e/common"
	. "github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
)

var testEnvironment *testenvironment.TestEnvironment

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	testEnvironment = testenvironment.NewTestEnvironment()

	// Delete the Eventing CR used for testing.
	if err := testEnvironment.DeleteEventingCR(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

func Test_NoEventingCRExists(t *testing.T) {
	t.Parallel()

	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, crErr := testEnvironment.GetEventingCRFromK8s(CRName, NamespaceName)
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.CoreV1().ServiceAccounts(NamespaceName).Get(ctx,
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

// Test_NoPublisherClusterRoleExists tests if the publisher-proxy ClusterRole was deleted.
func Test_NoPublisherClusterRoleExists(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.RbacV1().ClusterRoles().Get(ctx,
			eventing.GetPublisherClusterRoleName(*eventingCR), metav1.GetOptions{})
		if getErr == nil {
			return errors.New("PublisherClusterRole should have been deleted")
		}
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_NoPublisherClusterRoleBindingExists tests if the publisher-proxy ClusterRoleBinding was deleted.
func Test_NoPublisherClusterRoleBindingExists(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.RbacV1().ClusterRoleBindings().Get(ctx,
			eventing.GetPublisherClusterRoleBindingName(*eventingCR), metav1.GetOptions{})
		if getErr == nil {
			return errors.New("PublisherClusterRoleBinding should have been deleted")
		}
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_NoPublisherServicesExists tests if the publisher-proxy Services was deleted.
func Test_NoPublisherServicesExists(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		// check service to expose event publishing endpoint, was deleted.
		_, getErr := testEnvironment.K8sClientset.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherPublishServiceName(*eventingCR), metav1.GetOptions{})
		if getErr == nil {
			return errors.New("Publisher PublishService should have been deleted")
		}
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}

		// check service to expose metrics endpoint, was deleted.
		_, getErr = testEnvironment.K8sClientset.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherMetricsServiceName(*eventingCR), metav1.GetOptions{})
		if getErr == nil {
			return errors.New("Publisher MetricsService should have been deleted")
		}
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}

		// check service to expose health endpoint, was deleted.
		_, getErr = testEnvironment.K8sClientset.CoreV1().Services(NamespaceName).Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.AutoscalingV2().HorizontalPodAutoscalers(NamespaceName).Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.AppsV1().Deployments(NamespaceName).Get(ctx,
			eventing.GetPublisherDeploymentName(*eventingCR), metav1.GetOptions{})
		if !k8serrors.IsNotFound(getErr) {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}
