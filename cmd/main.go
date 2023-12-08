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

	"github.com/go-logr/zapr"
	apigatewayv1beta1 "github.com/kyma-project/api-gateway/apis/gateway/v1beta1"
	kapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kapixclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kutilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	kkubernetesscheme "k8s.io/client-go/kubernetes/scheme"
	kctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	eventingv1alpha1 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha1"
	eventingv1alpha2 "github.com/kyma-project/eventing-manager/api/eventing/v1alpha2"
	operatorv1alpha1 "github.com/kyma-project/eventing-manager/api/operator/v1alpha1"
	controllercache "github.com/kyma-project/eventing-manager/internal/controller/cache"
	controllerclient "github.com/kyma-project/eventing-manager/internal/controller/client"
	eventingcontroller "github.com/kyma-project/eventing-manager/internal/controller/operator/eventing"
	"github.com/kyma-project/eventing-manager/options"
	backendmetrics "github.com/kyma-project/eventing-manager/pkg/backend/metrics"
	"github.com/kyma-project/eventing-manager/pkg/env"
	"github.com/kyma-project/eventing-manager/pkg/eventing"
	"github.com/kyma-project/eventing-manager/pkg/istio/peerauthentication"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/eventing-manager/pkg/logger"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager"
	"github.com/kyma-project/eventing-manager/pkg/subscriptionmanager/jetstream"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = kctrl.Log.WithName("setup")
)

// init
func init() {
	kutilruntime.Must(kkubernetesscheme.AddToScheme(scheme))

	kutilruntime.Must(operatorv1alpha1.AddToScheme(scheme))

	kutilruntime.Must(apigatewayv1beta1.AddToScheme(scheme))

	kutilruntime.Must(kapiextensionsv1.AddToScheme(scheme))

	kutilruntime.Must(jetstream.AddToScheme(scheme))
	kutilruntime.Must(jetstream.AddV1Alpha2ToScheme(scheme))
	kutilruntime.Must(eventingv1alpha1.AddToScheme(scheme))
	kutilruntime.Must(eventingv1alpha2.AddToScheme(scheme))
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
	kctrl.SetLogger(zapr.NewLogger(ctrLogger.WithContext().Desugar()))

	// setup ctrl manager
	k8sRestCfg := kctrl.GetConfigOrDie()

	mgr, err := kctrl.NewManager(k8sRestCfg, kctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: opts.ProbeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
		WebhookServer:          webhook.NewServer(webhook.Options{Port: 9443}),
		Cache:                  cache.Options{SyncPeriod: &opts.ReconcilePeriod},
		Metrics:                server.Options{BindAddress: opts.MetricsAddr},
		NewCache:               controllercache.New,
		NewClient:              controllerclient.New,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// init custom kube client wrapper
	k8sClient := mgr.GetClient()
	dynamicClient, err := dynamic.NewForConfig(k8sRestCfg)
	if err != nil {
		panic(err.Error())
	}

	// init custom kube client wrapper
	apiClientSet, err := kapixclientset.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "failed to create new k8s clientset")
		os.Exit(1)
	}

	kubeClient := k8s.NewKubeClient(k8sClient, apiClientSet, "eventing-manager", dynamicClient)
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
		dynamicClient,
		mgr.GetScheme(),
		ctrLogger,
		recorder,
		eventingManager,
		backendConfig,
		subManagerFactory,
		opts,
		&operatorv1alpha1.Eventing{
			ObjectMeta: kmetav1.ObjectMeta{
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

	// Setup webhooks.
	if err = (&eventingv1alpha1.Subscription{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create webhook")
		os.Exit(1)
	}

	if err = (&eventingv1alpha2.Subscription{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create webhook")
		os.Exit(1)
	}

	// sync PeerAuthentications
	err = peerauthentication.SyncPeerAuthentications(ctx, kubeClient, ctrLogger.WithContext().Named("main"))
	if err != nil {
		setupLog.Error(err, "unable to sync PeerAuthentication")
		os.Exit(1)
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err = mgr.Start(kctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
