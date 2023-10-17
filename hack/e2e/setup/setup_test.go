//go:build e2e
// +build e2e

// Package setup_test is part of the end-to-end-tests. This package contains tests that evaluate the creation of a
// eventing CR and the creation of all correlated Kubernetes resources.
// To run the tests a Kubernetes cluster and Eventing-CR need to be available and configured. For this reason, the tests
// are seperated via the `e2e` buildtags. For more information please consult the `readme.md`.
package setup_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"

	"github.com/pkg/errors"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/pkg/eventing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/kyma-project/eventing-manager/hack/e2e/common"
	. "github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
)

var testEnvironment *testenvironment.TestEnvironment

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	testEnvironment = testenvironment.NewTestEnvironment()

	// create test namespace,
	if err := testEnvironment.CreateTestNamespace(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// Wait for eventing-manager deployment to get ready.
	if testEnvironment.TestConfigs.ManagerImage == "" {
		testEnvironment.Logger.Warn(
			"ENV `MANAGER_IMAGE` is not set. Test will not verify if the " +
				"manager deployment image is correct or not.",
		)
	}
	if err := testEnvironment.WaitForDeploymentReady(ManagerDeploymentName, NamespaceName, testEnvironment.TestConfigs.ManagerImage); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// Create the Eventing CR used for testing.
	if err := testEnvironment.SetupEventingCR(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// wait for a testenvironment.Interval for reconciliation to update status.
	time.Sleep(testenvironment.Interval)

	// Wait for Eventing CR to get ready.
	if err := testEnvironment.WaitForEventingCRReady(); err != nil {
		testEnvironment.Logger.Fatal(err.Error())
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_WebhookServerCertSecret tests if the Secret exists.
func Test_WebhookServerCertSecret(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.CoreV1().Secrets(NamespaceName).Get(ctx, WebhookServerCertSecretName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_WebhookServerCertJob tests if the Job exists.
func Test_WebhookServerCertJob(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.BatchV1().Jobs(NamespaceName).Get(ctx, WebhookServerCertJobName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_WebhookServerCertCronJob tests if the CronJob exists.
func Test_WebhookServerCertCronJob(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.BatchV1().CronJobs(NamespaceName).Get(ctx, WebhookServerCertJobName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherServiceAccount tests if the publisher-proxy ServiceAccount exists.
func Test_PublisherServiceAccount(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.CoreV1().ServiceAccounts(NamespaceName).Get(ctx,
			eventing.GetPublisherServiceAccountName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherClusterRole tests if the publisher-proxy ClusterRole exists.
func Test_PublisherClusterRole(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.RbacV1().ClusterRoles().Get(ctx,
			eventing.GetPublisherClusterRoleName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherClusterRoleBinding tests if the publisher-proxy ClusterRoleBinding exists.
func Test_PublisherClusterRoleBinding(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.RbacV1().ClusterRoleBindings().Get(ctx,
			eventing.GetPublisherClusterRoleBindingName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherServices tests if the publisher-proxy Services exists.
func Test_PublisherServices(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		// check service to expose event publishing endpoint.
		_, getErr := testEnvironment.K8sClientset.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherPublishServiceName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return errors.Wrap(getErr, "failed to ensure existence of publish service")
		}

		// check service to expose metrics endpoint.
		_, getErr = testEnvironment.K8sClientset.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherMetricsServiceName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return errors.Wrap(getErr, "failed to ensure existence of metrics service")
		}

		// check service to expose health endpoint.
		_, getErr = testEnvironment.K8sClientset.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherHealthServiceName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return errors.Wrap(getErr, "failed to ensure existence of health service")
		}

		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherHPA tests the publisher-proxy HorizontalPodAutoscaler.
func Test_PublisherHPA(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		gotHPA, getErr := testEnvironment.K8sClientset.AutoscalingV2().HorizontalPodAutoscalers(NamespaceName).Get(ctx,
			eventing.GetPublisherDeploymentName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}

		// check if the MinReplicas is correct.
		if int(*gotHPA.Spec.MinReplicas) != eventingCR.Spec.Min {
			return fmt.Errorf("HPA MinReplicas do not match. Want: %v, Got: %v",
				eventingCR.Spec.Min, int(*gotHPA.Spec.MinReplicas))
		}

		// check if the MaxReplicas is correct.
		if int(gotHPA.Spec.MaxReplicas) != eventingCR.Spec.Max {
			return fmt.Errorf("HPA MinReplicas do not match. Want: %v, Got: %v",
				eventingCR.Spec.Max, gotHPA.Spec.MaxReplicas)
		}

		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherProxyDeployment checks the publisher-proxy deployment.
func Test_PublisherProxyDeployment(t *testing.T) {
	t.Parallel()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		gotDeployment, getErr := testEnvironment.GetDeployment(eventing.GetPublisherDeploymentName(*eventingCR), NamespaceName)
		if getErr != nil {
			return getErr
		}

		// check if the deployment is ready.
		if gotDeployment.Status.Replicas != gotDeployment.Status.UpdatedReplicas ||
			gotDeployment.Status.Replicas != gotDeployment.Status.ReadyReplicas {
			err := fmt.Errorf("waiting for publisher-proxy deployment to get ready")
			testEnvironment.Logger.Debug(err.Error())
			return err
		}

		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherProxyPods checks the publisher-proxy pods.
// - checks if number of pods are between spec min and max.
// - checks if the ENV `BACKEND` is correctly defined for active backend.
// - checks if container resources are as defined in eventing CR spec.
func Test_PublisherProxyPods(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType))
	// RetryGet the Pods and test them.
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		// get publisher deployment
		gotDeployment, getErr := testEnvironment.GetDeployment(eventing.GetPublisherDeploymentName(*eventingCR), NamespaceName)
		if getErr != nil {
			return getErr
		}

		// RetryGet the Pods via labels.
		pods, err := testEnvironment.K8sClientset.CoreV1().Pods(NamespaceName).List(ctx, metav1.ListOptions{
			LabelSelector: ConvertSelectorLabelsToString(gotDeployment.Spec.Selector.MatchLabels)})
		if err != nil {
			return err
		}

		// check number of replicas of Publisher pods.
		if len(pods.Items) < eventingCR.Spec.Publisher.Min || len(pods.Items) > eventingCR.Spec.Publisher.Max {
			return fmt.Errorf(
				"the number of replicas for Publisher pods do not match with Eventing CR spec. "+
					"Wanted replicas to be between: %v and %v, but got %v",
				eventingCR.Spec.Publisher.Min,
				eventingCR.Spec.Publisher.Max,
				len(pods.Items),
			)
		}

		// Go through all Pods, check its spec. It should be same as defined in Eventing CR
		for _, pod := range pods.Items {
			// find the container.
			container := FindContainerInPod(pod, PublisherContainerName)
			if container == nil {
				return fmt.Errorf("Container: %v not found in publisher pod.", PublisherContainerName)
			}
			// compare the Resources with what is defined in the Eventing CR.
			if !reflect.DeepEqual(eventingCR.Spec.Publisher.Resources, container.Resources) {
				return fmt.Errorf(
					"error when checking pod %s resources:\n\twanted: %s\n\tgot: %s",
					pod.GetName(),
					eventingCR.Spec.Publisher.Resources.String(),
					container.Resources.String(),
				)
			}

			// check if the ENV `BACKEND` is defined correctly.
			wantBackendENVValue := "nats"
			if eventingv1alpha1.BackendType(testEnvironment.TestConfigs.BackendType) == eventingv1alpha1.EventMeshBackendType {
				wantBackendENVValue = "beb"
			}

			// get value of ENV `BACKEND`
			gotBackendENVValue := ""
			for _, envVar := range container.Env {
				if envVar.Name == "BACKEND" {
					gotBackendENVValue = envVar.Value
				}
			}
			// compare value
			if wantBackendENVValue != gotBackendENVValue {
				return fmt.Errorf(
					`error when checking ENV [BACKEND], wanted: %s, got: %s`,
					wantBackendENVValue,
					gotBackendENVValue,
				)
			}
		}

		// Everything is fine.
		return nil
	})
	require.NoError(t, err)
}

func Test_PriorityClass(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()

	// Check if the PriorityClass exists in the cluster.
	err := Retry(testenvironment.Attempts, testenvironment.Interval, func() error {
		_, getErr := testEnvironment.K8sClientset.SchedulingV1().PriorityClasses().Get(
			ctx, eventing.PriorityClassName, metav1.GetOptions{})
		return getErr
	})
	require.Nil(t, err, fmt.Errorf("error while fetching PriorityClass: %v", err))
}
