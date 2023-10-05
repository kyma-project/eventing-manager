/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/zapr"

	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager"
	subscriptionv1alpha1 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha1"
	subscriptionv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"

	"github.com/kyma-project/kyma/components/eventing-controller/pkg/subscriptionmanager/jetstream"

	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/kyma-project/eventing-manager/pkg/k8s"

	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/eventing"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/options"
	backendmetrics "github.com/kyma-project/kyma/components/eventing-controller/pkg/backend/metrics"
	apiclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	apigatewayv1beta1 "github.com/kyma-incubator/api-gateway/api/v1beta1"
	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/v1alpha1"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(eventingv1alpha1.AddToScheme(scheme))
	utilruntime.Must(natsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apigatewayv1beta1.AddToScheme(scheme))

	utilruntime.Must(jetstream.AddToScheme(scheme))
	utilruntime.Must(jetstream.AddV1Alpha2ToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

const defaultMetricsPort = 9443

func main() { //nolint:funlen // main function needs to initialize many object
	var enableLeaderElection bool
	var leaderElectionID string
	var metricsPort int
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionID, "leaderElectionID", "26479083.kyma-project.io",
		"ID for the controller leader election.")
	flag.IntVar(&metricsPort, "metricsPort", defaultMetricsPort, "Port number for metrics endpoint.")

	opts := options.New()
	if err := opts.Parse(); err != nil {
		log.Fatalf("Failed to parse options, error: %v", err)
	}

	ctrLogger, err := logger.New(opts.LogFormat, opts.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger, error: %v", err)
	}
	defer func() {
		if err = ctrLogger.WithContext().Sync(); err != nil {
			log.Printf("Failed to flush logger, error: %v", err)
		}
	}()

	// Set controller core logger.
	ctrl.SetLogger(zapr.NewLogger(ctrLogger.WithContext().Desugar()))

	// setup ctrl manager
	k8sRestCfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(k8sRestCfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     opts.MetricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: opts.ProbeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
		SyncPeriod:             &opts.ReconcilePeriod,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// init custom kube client wrapper
	k8sClient := mgr.GetClient()

	// init custom kube client wrapper
	apiClientSet, err := apiclientset.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "failed to create new k8s clientset")
		os.Exit(1)
	}

	kubeClient := k8s.NewKubeClient(k8sClient, apiClientSet, "eventing-manager")
	recorder := mgr.GetEventRecorderFor("eventing-manager")
	ctx := context.Background()

	// get backend configs.
	backendConfig := env.GetBackendConfig()

	// create eventing manager instance.
	eventingManager := eventing.NewEventingManager(ctx, k8sClient, kubeClient, backendConfig, ctrLogger, recorder)

	// init the metrics collector.
	metricsCollector := backendmetrics.NewCollector()
	metricsCollector.RegisterMetrics()

	// init subscription manager factory.
	subManagerFactory := subscriptionmanager.NewFactory(
		k8sRestCfg,
		":8080",
		metricsCollector,
		opts.ReconcilePeriod,
		ctrLogger,
	)

	// create Eventing reconciler instance
	eventingReconciler := eventingcontroller.NewReconciler(
		k8sClient,
		kubeClient,
		mgr.GetScheme(),
		ctrLogger,
		recorder,
		eventingManager,
		backendConfig,
		subManagerFactory,
		opts,
		&eventingv1alpha1.Eventing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backendConfig.EventingCRName,
				Namespace: backendConfig.EventingCRNamespace,
			},
		},
	)

	if err = (eventingReconciler).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Eventing")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	// setup webhooks.
	if err = (&subscriptionv1alpha1.Subscription{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create webhook")
		os.Exit(1)
	}

	if err = (&subscriptionv1alpha2.Subscription{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create webhook")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
