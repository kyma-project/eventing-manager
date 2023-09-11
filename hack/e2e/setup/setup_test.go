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

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	"github.com/kyma-project/eventing-manager/hack/e2e/env"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/pkg/errors"

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

var testConfigs *env.E2EConfig

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	var err error
	logger, err = SetupLogger()
	if err != nil {
		logger.Fatal(err.Error())
	}

	testConfigs, err = env.GetE2EConfig()
	if err != nil {
		logger.Fatal(err.Error())
	}
	logger.Info(fmt.Sprintf("##### NOTE: Tests will run w.r.t. backend: %s", testConfigs.BackendType))

	clientSet, k8sClient, err = GetK8sClients()
	if err != nil {
		logger.Fatal(err.Error())
	}

	ctx := context.TODO()
	// Create the Namespace used for testing.
	err = Retry(attempts, interval, func() error {
		// It's fine if the Namespace already exists.
		return client.IgnoreAlreadyExists(k8sClient.Create(ctx, Namespace()))
	})
	if err != nil {
		logger.Fatal(err.Error())
	}

	// Wait for eventing-manager deployment to get ready.
	if testConfigs.ManagerImage == "" {
		logger.Warn(
			"ENV `MANAGER_IMAGE` is not set. Test will not verify if the " +
				"manager deployment image is correct or not.",
		)
	}
	if err := waitForEventingManagerDeploymentReady(testConfigs.ManagerImage); err != nil {
		logger.Fatal(err.Error())
	}

	// Create the Eventing CR used for testing.
	err = Retry(attempts, interval, func() error {
		eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
		errEvnt := k8sClient.Create(ctx, eventingCR)
		if k8serrors.IsAlreadyExists(errEvnt) {
			gotEventingCR, getErr := getEventingCR(ctx, eventingCR.Name, eventingCR.Namespace)
			if getErr != nil {
				return err
			}

			// If Backend type is changed then update the CR.
			if gotEventingCR.Spec.Backend.Type != eventingCR.Spec.Backend.Type {
				eventingCR.ObjectMeta = gotEventingCR.ObjectMeta
				if errEvnt = k8sClient.Update(ctx, eventingCR); getErr != nil {
					return err
				}
			} else {
				logger.Warn(
					"error while creating Eventing CR, resource already exist; test will continue with existing CR",
				)
			}
			return nil
		}
		return errEvnt
	})
	if err != nil {
		logger.Fatal(err.Error())
	}

	// wait for an interval for reconciliation to update status.
	time.Sleep(interval)

	// Wait for Eventing CR to get ready.
	if err := waitForEventingCRReady(); err != nil {
		logger.Fatal(err.Error())
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_WebhookServerCertSecret tests if the Secret exists.
func Test_WebhookServerCertSecret(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(attempts, interval, func() error {
		_, getErr := clientSet.CoreV1().Secrets(NamespaceName).Get(ctx, WebhookServerCertSecretName, metav1.GetOptions{})
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
	err := Retry(attempts, interval, func() error {
		_, getErr := clientSet.BatchV1().Jobs(NamespaceName).Get(ctx, WebhookServerCertJobName, metav1.GetOptions{})
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
	err := Retry(attempts, interval, func() error {
		_, getErr := clientSet.BatchV1().CronJobs(NamespaceName).Get(ctx, WebhookServerCertJobName, metav1.GetOptions{})
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		_, getErr := clientSet.CoreV1().ServiceAccounts(NamespaceName).Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		_, getErr := clientSet.RbacV1().ClusterRoles().Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		_, getErr := clientSet.RbacV1().ClusterRoleBindings().Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		// check service to expose event publishing endpoint.
		_, getErr := clientSet.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherPublishServiceName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return errors.Wrap(getErr, "failed to ensure existence of publish service")
		}

		// check service to expose metrics endpoint.
		_, getErr = clientSet.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherMetricsServiceName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return errors.Wrap(getErr, "failed to ensure existence of metrics service")
		}

		// check service to expose health endpoint.
		_, getErr = clientSet.CoreV1().Services(NamespaceName).Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	err := Retry(attempts, interval, func() error {
		gotHPA, getErr := clientSet.AutoscalingV2().HorizontalPodAutoscalers(NamespaceName).Get(ctx,
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

// Test_PublisherProxyPods checks the publisher-proxy pods.
// - checks if number of pods are between spec min and max.
// - checks if the ENV `BACKEND` is correctly defined for active backend.
// - checks if container resources are as defined in eventing CR spec.
func Test_PublisherProxyPods(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(testConfigs.BackendType))
	// RetryGet the Pods and test them.
	err := Retry(attempts, interval, func() error {
		// get publisher deployment
		gotDeployment, getErr := getDeployment(ctx, eventing.GetPublisherDeploymentName(*eventingCR), NamespaceName)
		if getErr != nil {
			return getErr
		}

		// RetryGet the Pods via labels.
		pods, err := clientSet.CoreV1().Pods(NamespaceName).List(ctx, metav1.ListOptions{
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
			if eventingv1alpha1.BackendType(testConfigs.BackendType) == eventingv1alpha1.EventMeshBackendType {
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

// Wait for Eventing CR to get ready.
func waitForEventingCRReady() error {
	// RetryGet the Eventing CR and test status.
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

		if gotEventingCR.Spec.Backend.Type != gotEventingCR.Status.ActiveBackend {
			err := fmt.Errorf("waiting for Eventing CR to switch backend")
			logger.Debug(err.Error())
			return err
		}

		if gotEventingCR.Status.State != eventingv1alpha1.StateReady {
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
