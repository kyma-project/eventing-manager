//go:build e2e
// +build e2e

// Package setup_test is part of the end-to-end-tests. This package contains tests that evaluate the creation of a
// eventing CR and the creation of all correlated Kubernetes resources.
// To run the tests a Kubernetes cluster and Eventing-CR need to be available and configured. For this reason, the tests
// are seperated via the `e2e` buildtags. For more information please consult the `readme.md`.
package setup_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"istio.io/client-go/pkg/apis/security/v1beta1"
	istio "istio.io/client-go/pkg/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	. "github.com/kyma-project/eventing-manager/hack/e2e/common"
	. "github.com/kyma-project/eventing-manager/hack/e2e/common/fixtures"
	test "github.com/kyma-project/eventing-manager/hack/e2e/common/testenvironment"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/kyma-project/eventing-manager/pkg/utils/istio/peerauthentication"
)

var testPeerauthentication bool
var env *test.TestEnvironment
var istioClient *istio.Clientset

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	testPeerauthentication = *flag.Bool("peerauthentication", false, "Tests if PeerAuthentication was created")
	flag.Parse()

	env = test.NewTestEnvironment()

	// Create a test namespace.
	if err := env.CreateTestNamespace(); err != nil {
		env.Logger.Fatal(err.Error())
	}

	// Wait for eventing-manager deployment to get ready.
	if env.TestConfigs.ManagerImage == "" {
		env.Logger.Warn(
			"ENV `MANAGER_IMAGE` is not set. Test will not verify if the " +
				"manager deployment image is correct or not.",
		)
	}
	if err := env.WaitForDeploymentReady(ManagerDeploymentName, NamespaceName, env.TestConfigs.ManagerImage); err != nil {
		env.Logger.Fatal(err.Error())
	}

	// Create the Eventing CR used for testing.
	if err := env.SetupEventingCR(); err != nil {
		env.Logger.Fatal(err.Error())
	}

	// Wait for a test.Interval for reconciliation to update status.
	time.Sleep(test.Interval)

	// Wait for Eventing CR to get ready.
	if err := env.WaitForEventingCRReady(); err != nil {
		env.Logger.Fatal(err.Error())
	}

	if testPeerauthentication {
		config, err := GetRestConfig()
		if err != nil {
			env.Logger.Fatal(err.Error())
		}

		istioClient, err = istio.NewForConfig(config)
		if err != nil {
			env.Logger.Fatal(err.Error())
		}
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_EventPublisherProxyMetricsPeerAuthentication checks if the Istio PeerAuthentication for the metrics endpoint of
// Event-Publisher-Proxy was created.
func Test_EventPublisherProxyMetricsPeerAuthentication(t *testing.T) {
	t.Parallel()
	if !testPeerauthentication {
		return
	}

	deploy, err := env.GetDeploymentFromK8s(ManagerDeploymentName, NamespaceName)
	require.NoError(t, err)

	wantPeerAuth := peerauthentication.EventPublisherProxyMetrics(NamespaceName, deploy.OwnerReferences)
	var givenPeerAuth *v1beta1.PeerAuthentication
	eppErr := Retry(test.Attempts, test.Interval, func() error {
		givenPeerAuth, err = istioClient.SecurityV1beta1().PeerAuthentications(NamespaceName).Get(
			context.TODO(), wantPeerAuth.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if givenPeerAuth == nil {
			return fmt.Errorf("could not fetch PeerAuthentication %s", wantPeerAuth.Name)
		}
		return err
	})
	require.NoError(t, eppErr)

	require.Equal(t, wantPeerAuth.OwnerReferences, givenPeerAuth.OwnerReferences)
}

// Test_EventingManagerMetricsPeerAuthentication checks if the Istio PeerAuthentication for the metrics endpoint of
// Eventing-Manager was created.
func Test_EventingManagerMetricsPeerAuthentication(t *testing.T) {
	t.Parallel()
	if !testPeerauthentication {
		return
	}

	deploy, err := env.GetDeploymentFromK8s(ManagerDeploymentName, NamespaceName)
	require.NoError(t, err)

	wantPeerAuth := peerauthentication.EventingManagerMetrics(NamespaceName, deploy.OwnerReferences)
	var givenPeerAuth *v1beta1.PeerAuthentication
	emErr := Retry(test.Attempts, test.Interval, func() error {
		givenPeerAuth, err = istioClient.SecurityV1beta1().PeerAuthentications(NamespaceName).Get(
			context.TODO(), wantPeerAuth.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if givenPeerAuth == nil {
			return fmt.Errorf("could not fetch PeerAuthentication %s", wantPeerAuth.Name)
		}
		return err
	})
	require.NoError(t, emErr)

	require.Equal(t, wantPeerAuth.OwnerReferences, givenPeerAuth.OwnerReferences)
}

// Test_WebhookServerCertSecret tests if the Secret exists.
func Test_WebhookServerCertSecret(t *testing.T) {
	t.Parallel()
	err := Retry(test.Attempts, test.Interval, func() error {
		_, getErr := env.K8sClientset.CoreV1().Secrets(NamespaceName).Get(context.TODO(), WebhookServerCertSecretName, metav1.GetOptions{})
		return getErr
	})
	require.NoError(t, err)
}

// Test_WebhookServerCertJob tests if the Job exists.
func Test_WebhookServerCertJob(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(test.Attempts, test.Interval, func() error {
		job, getErr := env.K8sClientset.BatchV1().Jobs(NamespaceName).Get(ctx, WebhookServerCertJobName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}

		// Check if the PriorityClassName was set correctly.
		if job.Spec.Template.Spec.PriorityClassName != eventing.PriorityClassName {
			return fmt.Errorf("Job '%s' was expected to have PriorityClassName '%s' but has '%s'",
				job.GetName(),
				eventing.PriorityClassName,
				job.Spec.Template.Spec.PriorityClassName,
			)
		}

		return nil
	})
	require.NoError(t, err)
}

// Test_WebhookServerCertCronJob tests if the CronJob exists.
func Test_WebhookServerCertCronJob(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(test.Attempts, test.Interval, func() error {
		job, getErr := env.K8sClientset.BatchV1().CronJobs(NamespaceName).Get(
			ctx,
			WebhookServerCertJobName,
			metav1.GetOptions{},
		)
		if getErr != nil {
			return getErr
		}

		// Check if the PriorityClassName was set correctly.
		if job.Spec.JobTemplate.Spec.Template.Spec.PriorityClassName != eventing.PriorityClassName {
			return fmt.Errorf("ChronJob '%s' was expected to have PriorityClassName '%s' but has '%s'",
				job.GetName(),
				eventing.PriorityClassName,
				job.Spec.JobTemplate.Spec.Template.Spec.PriorityClassName,
			)
		}

		return nil
	})
	require.NoError(t, err)
}

// Test_PublisherServiceAccount tests if the publisher-proxy ServiceAccount exists.
func Test_PublisherServiceAccount(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	eventingCR := EventingCR(eventingv1alpha1.BackendType(env.TestConfigs.BackendType))
	err := Retry(test.Attempts, test.Interval, func() error {
		_, getErr := env.K8sClientset.CoreV1().ServiceAccounts(NamespaceName).Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(env.TestConfigs.BackendType))
	err := Retry(test.Attempts, test.Interval, func() error {
		_, getErr := env.K8sClientset.RbacV1().ClusterRoles().Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(env.TestConfigs.BackendType))
	err := Retry(test.Attempts, test.Interval, func() error {
		_, getErr := env.K8sClientset.RbacV1().ClusterRoleBindings().Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(env.TestConfigs.BackendType))
	err := Retry(test.Attempts, test.Interval, func() error {
		// check service to expose event publishing endpoint.
		_, getErr := env.K8sClientset.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherPublishServiceName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return errors.Wrap(getErr, "failed to ensure existence of publish service")
		}

		// check service to expose metrics endpoint.
		_, getErr = env.K8sClientset.CoreV1().Services(NamespaceName).Get(ctx,
			eventing.GetPublisherMetricsServiceName(*eventingCR), metav1.GetOptions{})
		if getErr != nil {
			return errors.Wrap(getErr, "failed to ensure existence of metrics service")
		}

		// check service to expose health endpoint.
		_, getErr = env.K8sClientset.CoreV1().Services(NamespaceName).Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(env.TestConfigs.BackendType))
	err := Retry(test.Attempts, test.Interval, func() error {
		gotHPA, getErr := env.K8sClientset.AutoscalingV2().HorizontalPodAutoscalers(NamespaceName).Get(ctx,
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(env.TestConfigs.BackendType))
	err := Retry(test.Attempts, test.Interval, func() error {
		gotDeployment, getErr := env.GetDeployment(eventing.GetPublisherDeploymentName(*eventingCR), NamespaceName)
		if getErr != nil {
			return getErr
		}

		// check if the deployment is ready.
		if gotDeployment.Status.Replicas != gotDeployment.Status.UpdatedReplicas ||
			gotDeployment.Status.Replicas != gotDeployment.Status.ReadyReplicas {
			err := fmt.Errorf("waiting for publisher-proxy deployment to get ready")
			env.Logger.Debug(err.Error())
			return err
		}

		if gotDeployment.Spec.Template.Spec.PriorityClassName != eventing.PriorityClassName {
			return fmt.Errorf(
				"error while checking deployment '%s'; PriorityClasssName was supposed to be %s, but was %s",
				gotDeployment.GetName(),
				eventing.PriorityClassName,
				gotDeployment.Spec.Template.Spec.PriorityClassName,
			)
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
	eventingCR := EventingCR(eventingv1alpha1.BackendType(env.TestConfigs.BackendType))
	// Retry to get the Pods and test them.
	err := Retry(test.Attempts, test.Interval, func() error {
		// get publisher deployment
		gotDeployment, getErr := env.GetDeployment(eventing.GetPublisherDeploymentName(*eventingCR), NamespaceName)
		if getErr != nil {
			return getErr
		}

		// RetryGet the Pods via labels.
		pods, err := env.K8sClientset.CoreV1().Pods(NamespaceName).List(ctx, metav1.ListOptions{
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

		// Go through all Pods, check its spec. It should be same as defined in Eventing CR.
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

			// Check if the PriorityClassName was set as expected.
			if pod.Spec.PriorityClassName != eventing.PriorityClassName {
				return fmt.Errorf("'PriorityClassName' of Pod %v should be %v but was %v",
					pod.GetName(),
					eventing.PriorityClassName,
					pod.Spec.PriorityClassName,
				)
			}

			// Check if the ENV `BACKEND` is defined correctly.
			wantBackendENVValue := "nats"
			if eventingv1alpha1.BackendType(env.TestConfigs.BackendType) == eventingv1alpha1.EventMeshBackendType {
				wantBackendENVValue = "beb"
			}

			// Get value of ENV `BACKEND`.
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
	err := Retry(test.Attempts, test.Interval, func() error {
		_, getErr := env.K8sClientset.SchedulingV1().PriorityClasses().Get(
			ctx, eventing.PriorityClassName, metav1.GetOptions{})
		return getErr
	})
	require.Nil(t, err, fmt.Errorf("error while fetching PriorityClass: %v", err))

	// Check if the Eventing-Manager Deployment has the right PriorityClassName. This implicits that the
	// corresponding Pod also has the right PriorityClassName.
	err = Retry(test.Attempts, test.Interval, func() error {
		deploy, getErr := env.GetDeployment(ManagerDeploymentName, NamespaceName)
		if getErr != nil {
			return getErr
		}

		if deploy.Spec.Template.Spec.PriorityClassName != eventing.PriorityClassName {
			return fmt.Errorf("deployment '%s' should have the PriorityClassName %s but was %s",
				deploy.GetName(),
				eventing.PriorityClassName,
				deploy.Spec.Template.Spec.PriorityClassName,
			)
		}

		return nil
	})
}
