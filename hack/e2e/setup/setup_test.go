//go:build e2e
// +build e2e

// Package setup_test is part of the end-to-end-tests. This package contains tests that evaluate the creation of a
// NATS-server CR and the creation of all correlated Kubernetes resources.
// To run the tests a Kubernetes cluster and a NATS-CR need to be available and configured. For this reason, the tests
// are seperated via the `e2e` buildtags. For more information please consult the `readme.md`.
package setup_test

import (
	"context"
	"fmt"
	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/hack/e2e/env"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/kyma-project/eventing-manager/hack/e2e/common"
	. "github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
)

// Constants for retries.
const (
	interval = 2 * time.Second
	attempts = 60
)

// clientSet is what is used to access K8s build-in resources like Pods, Namespaces and so on.
var clientSet *kubernetes.Clientset //nolint:gochecknoglobals // This will only be accessible in e2e tests.

// k8sClient is what is used to access the NATS CR.
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
	// Create the Namespace used for testing.
	err = Retry(attempts, interval, func() error {
		// It's fine if the Namespace already exists.
		return client.IgnoreAlreadyExists(k8sClient.Create(ctx, Namespace()))
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Wait for eventing-manager deployment to get ready.
	if testConfigs.ManagerImage == "" {
		logger.Warn(
			"ENV `MANAGER_IMAGE` is not set. Test will not verify if the " +
				"manager deployment image is correct or not.",
		)
	}
	if err := waitForEventingManagerDeploymentReady(testConfigs.ManagerImage); err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Create the Eventing CR used for testing.
	err = Retry(attempts, interval, func() error {
		errEvnt := k8sClient.Create(ctx, EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType)))
		if k8serrors.IsAlreadyExists(errEvnt) {
			logger.Warn(
				"error while creating Eventing CR, resource already exist; test will continue with existing CR",
			)
			return nil
		}
		return errEvnt
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// wait for an interval for reconciliation to update status.
	time.Sleep(interval)

	// Wait for Eventing CR to get ready.
	if err := waitForEventingCRReady(); err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_CR checks if the CR in the cluster is equal to what we defined.
func Test_CR(t *testing.T) {
	want := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))

	ctx := context.TODO()
	// Get the Eventing CR from the cluster.
	actual, err := RetryGet(attempts, interval, func() (*eventingv1alpha1.Eventing, error) {
		return getEventingCR(ctx, want.Name, want.Namespace)
	})
	require.NoError(t, err)

	require.True(t,
		reflect.DeepEqual(want.Spec, actual.Spec),
		fmt.Sprintf("wanted spec.cluster to be \n\t%v\n but got \n\t%v", want.Spec, actual.Spec),
	)
}

/// Checklist of resources to check:
// Webhook secret
// Webhook job and cronjob
// Manager deployment
// CR: Publisher deployment
// CR: Publisher pods
// - check if the ENVs are correctly defined for active backend
// - check if number of pods are between spec min and max
// - check if resources are as defined in eventing CR spec
// CR: ServiceAccount // exists
// CR: ClusterRole // exists
// CR: ClusterRoleBinding // exists
// CR: Service to expose event publishing endpoint of EPP.
// CR: Service to expose metrics endpoint of EPP.
// CR: Service to expose health endpoint of EPP.
// CR: HPA to auto-scale publisher proxy. // spec min and max

// Test_WebhookServerCertSecret tests if the Secret exists.
func Test_WebhookServerCertSecret(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(attempts, interval, func() error {
		_, secErr := clientSet.CoreV1().Secrets(NamespaceName).Get(ctx, WebhookServerCertSecretName, metav1.GetOptions{})
		if secErr != nil {
			return secErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_WebhookServerCertJob tests if the Job exists.
func Test_WebhookServerCertJob(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(attempts, interval, func() error {
		_, secErr := clientSet.BatchV1().Jobs(NamespaceName).Get(ctx, WebhookServerCertJobName, metav1.GetOptions{})
		if secErr != nil {
			return secErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_WebhookServerCertCronJob tests if the CronJob exists.
func Test_WebhookServerCertCronJob(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(attempts, interval, func() error {
		_, secErr := clientSet.BatchV1().CronJobs(NamespaceName).Get(ctx, WebhookServerCertJobName, metav1.GetOptions{})
		if secErr != nil {
			return secErr
		}
		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherProxyDeployment checks the publisher-proxy deployment.
func Test_PublisherProxyDeployment(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		gotDeployment, getErr := getDeployment(ctx, eventing.GetPublisherDeploymentName(*eventingCR), NamespaceName)
		if getErr != nil {
			return getErr
		}

		// check if the deployment is ready.
		if gotDeployment.Status.Replicas != gotDeployment.Status.UpdatedReplicas ||
			gotDeployment.Status.Replicas != gotDeployment.Status.ReadyReplicas {
			err := fmt.Errorf("waiting for publisher-proxy deployment to get ready")
			logger.Debug(err.Error())
			return err
		}

		return nil
	})
	require.NoError(t, err)
}

//// Test_PodsResources checks if the number of Pods is the same as defined in the NATS CR and that all Pods have the resources,
//// that are defined in the CRD.
//func Test_PodsResources(t *testing.T) {
//	t.Parallel()
//
//	ctx := context.TODO()
//	// RetryGet the NATS Pods and test them.
//	err := Retry(attempts, interval, func() error {
//		// RetryGet the NATS Pods via labels.
//		pods, err := clientSet.CoreV1().Pods(NamespaceName).List(ctx, PodListOpts())
//		if err != nil {
//			return err
//		}
//
//		// The number of Pods must be equal NATS.spec.cluster.size. We check this in the retry, because it may take
//		// some time for all Pods to be there.
//		if len(pods.Items) != NATSCR().Spec.Cluster.Size {
//			return fmt.Errorf(
//				"error while fetching Pods; wanted %v Pods but got %v",
//				NATSCR().Spec.Cluster.Size,
//				pods.Items,
//			)
//		}
//
//		// Go through all Pods, find the natsCR container in each and compare its Resources with what is defined in
//		// the NATS CR.
//		foundContainers := 0
//		for _, pod := range pods.Items {
//			for _, container := range pod.Spec.Containers {
//				if !(container.Name == ContainerName) {
//					continue
//				}
//				foundContainers += 1
//				if !reflect.DeepEqual(NATSCR().Spec.Resources, container.Resources) {
//					return fmt.Errorf(
//						"error when checking pod %s resources:\n\twanted: %s\n\tgot: %s",
//						pod.GetName(),
//						NATSCR().Spec.Resources.String(),
//						container.Resources.String(),
//					)
//				}
//			}
//		}
//		if foundContainers != NATSCR().Spec.Cluster.Size {
//			return fmt.Errorf(
//				"error while fethching 'natsCR' Containers: expected %v but found %v",
//				NATSCR().Spec.Cluster.Size,
//				foundContainers,
//			)
//		}
//
//		// Everything is fine.
//		return nil
//	})
//	require.NoError(t, err)
//}
//
//// Test_PodsReady checks if the number of Pods is the same as defined in the NATS CR and that all Pods are ready.
//func Test_PodsReady(t *testing.T) {
//	t.Parallel()
//
//	ctx := context.TODO()
//	// RetryGet the NATS CR. It will tell us how many Pods we should expect.
//	natsCR, err := RetryGet(attempts, interval, func() (*natsv1alpha1.NATS, error) {
//		return getNATSCR(ctx, CRName, NamespaceName)
//	})
//	require.NoError(t, err)
//
//	// RetryGet the NATS Pods and test them.
//	err = Retry(attempts, interval, func() error {
//		var pods *v1.PodList
//		// RetryGet the NATS Pods via labels.
//		pods, err = clientSet.CoreV1().Pods(NamespaceName).List(ctx, PodListOpts())
//		if err != nil {
//			return err
//		}
//
//		// The number of Pods must be equal NATS.spec.cluster.size. We check this in the retry, because it may take
//		// some time for all Pods to be there.
//		if len(pods.Items) != natsCR.Spec.Cluster.Size {
//			return fmt.Errorf(
//				"Error while fetching pods; wanted %v Pods but got %v", natsCR.Spec.Cluster.Size, pods.Items,
//			)
//		}
//
//		// Check if all Pods are ready (the status.conditions array has an entry with .type="Ready" and .status="True").
//		for _, pod := range pods.Items {
//			foundReadyCondition := false
//			for _, cond := range pod.Status.Conditions {
//				if cond.Type != "Ready" {
//					continue
//				}
//				foundReadyCondition = true
//				if cond.Status != "True" {
//					return fmt.Errorf(
//						"Pod %s has 'Ready' conditon '%s' but wanted 'True'", pod.GetName(), cond.Status,
//					)
//				}
//			}
//			if !foundReadyCondition {
//				return fmt.Errorf("Could not find 'Ready' condition for Pod %s", pod.GetName())
//			}
//		}
//
//		// Everything is fine.
//		return nil
//	})
//	require.NoError(t, err)
//}
//
//// Test_Secret tests if the Secret was created.
//func Test_Secret(t *testing.T) {
//	t.Parallel()
//	ctx := context.TODO()
//	err := Retry(attempts, interval, func() error {
//		_, secErr := clientSet.CoreV1().Secrets(NamespaceName).Get(ctx, SecretName, metav1.GetOptions{})
//		if secErr != nil {
//			return secErr
//		}
//		return nil
//	})
//	require.NoError(t, err)
//}

func getEventingCR(ctx context.Context, name, namespace string) (*eventingv1alpha1.Eventing, error) {
	var eventingCR eventingv1alpha1.Eventing
	err := k8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &eventingCR)
	return &eventingCR, err
}

func getDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	return clientSet.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

func cmToMap(cm string) map[string]string {
	lines := strings.Split(cm, "\n")

	cmMap := make(map[string]string)
	for _, line := range lines {
		l := strings.Split(line, ": ")
		if len(l) < 2 {
			continue
		}
		key := strings.TrimSpace(l[0])
		val := strings.TrimSpace(l[1])
		cmMap[key] = val
	}

	return cmMap
}

func checkValueInCMMap(cmm map[string]string, key, expectedValue string) error {
	val, ok := cmm[key]
	if !ok {
		return fmt.Errorf("could net get '%s' from Configmap", key)
	}

	if val != expectedValue {
		return fmt.Errorf("expected value for '%s' to be '%s' but was '%s'", key, expectedValue, val)
	}

	return nil
}

// Wait for Eventing CR to get ready.
func waitForEventingCRReady() error {
	// RetryGet the NATS CR and test status.
	return Retry(attempts, interval, func() error {
		logger.Debug(fmt.Sprintf("waiting for Eventing CR to get ready. "+
			"CR name: %s, namespace: %s", CRName, NamespaceName))

		ctx := context.TODO()
		// Get the Eventing CR from the cluster.
		gotEventingCR, err := RetryGet(attempts, interval, func() (*eventingv1alpha1.Eventing, error) {
			return getEventingCR(ctx, CRName, NamespaceName)
		})
		if err != nil {
			return err
		}

		if gotEventingCR.Status.State != natsv1alpha1.StateReady {
			err := fmt.Errorf("waiting for Eventing CR to get ready state")
			logger.Debug(err.Error())
			return err
		}

		// Everything is fine.
		logger.Debug(fmt.Sprintf("Eventing CR is ready. "+
			"CR name: %s, namespace: %s", CRName, NamespaceName))
		return nil
	})
}

// Wait for eventing-manager deployment to get ready with correct image.
func waitForEventingManagerDeploymentReady(image string) error {
	// RetryGet the Eventing Manager and test status.
	return Retry(attempts, interval, func() error {
		logger.Debug(fmt.Sprintf("waiting for eventing-manager deployment to get ready with image: %s", image))
		ctx := context.TODO()
		// Get the eventing-manager deployment from the cluster.
		gotDeployment, err := RetryGet(attempts, interval, func() (*appsv1.Deployment, error) {
			return getDeployment(ctx, ManagerDeploymentName, NamespaceName)
		})
		if err != nil {
			return err
		}

		// if image is provided, then check if the deployment has correct image.
		if image != "" && gotDeployment.Spec.Template.Spec.Containers[0].Image != image {
			err := fmt.Errorf("expected eventing-manager image to be: %s, but found: %s", image,
				gotDeployment.Spec.Template.Spec.Containers[0].Image,
			)
			logger.Debug(err.Error())
			return err
		}

		// check if the deployment is ready.
		if *gotDeployment.Spec.Replicas != gotDeployment.Status.UpdatedReplicas ||
			*gotDeployment.Spec.Replicas != gotDeployment.Status.ReadyReplicas ||
			*gotDeployment.Spec.Replicas != gotDeployment.Status.AvailableReplicas {
			err := fmt.Errorf("waiting for eventing-manager deployment to get ready")
			logger.Debug(err.Error())
			return err
		}

		// Everything is fine.
		logger.Debug(fmt.Sprintf("eventing-manager deployment is ready with image: %s", image))
		return nil
	})
}
