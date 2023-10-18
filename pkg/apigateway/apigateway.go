package apigateway

import (
	"context"
	"fmt"
	recerrors "github.com/kyma-project/eventing-manager/internal/controller/errors"
	"github.com/kyma-project/eventing-manager/pkg/apigateway/authorizationpolicy"
	"github.com/kyma-project/eventing-manager/pkg/apigateway/requestauthentication"
	"github.com/kyma-project/eventing-manager/pkg/apigateway/virtualservice"
	"github.com/kyma-project/eventing-manager/pkg/backend/eventmesh"
	"github.com/kyma-project/eventing-manager/pkg/constants"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/eventing-manager/pkg/utils"
	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"net/url"
	"reflect"
)

const (
	externalHostPrefix = "web"
	externalSinkScheme = "https"
	namePrefix         = "eventing-"
	kymaGateway        = "kyma-system/kyma-gateway"
)

var _ APIGateway = Gateway{}

//go:generate mockery --name=APIGateway --outpkg=mocks --case=underscore
type APIGateway interface {
	ExposeSink(context.Context, eventingv1alpha2.Subscription, string, eventmesh.OAuth2ClientCredentials, *zap.SugaredLogger) (string, error)
}

type Gateway struct {
	kubeClient k8s.Client
}

func NewGateway(kubeClient k8s.Client) *Gateway {
	return &Gateway{
		kubeClient: kubeClient,
	}
}

func (g Gateway) ExposeSink(ctx context.Context, subscription eventingv1alpha2.Subscription,
	domain string, credentials eventmesh.OAuth2ClientCredentials, logger *zap.SugaredLogger) (string, error) {
	sinkURL, err := url.ParseRequestURI(subscription.Spec.Sink)
	if err != nil {
		return "", recerrors.NewSkippable(xerrors.Errorf("failed to parse URI: %v", err))
	}

	sinkPort, err := utils.GetPortNumberFromURL(*sinkURL)
	if err != nil {
		return "", xerrors.Errorf("failed to extract port from sink URI: %v", err)
	}

	// extract k8s service name and namespace from sink URL.
	svcName, svcNamespace, err := GetServiceNameFromSink(sinkURL.Host)
	if err != nil {
		return "", xerrors.Errorf("failed to parse svc name and ns: %v", err)
	}
	// fetch service from cluster.
	sinkService, err := g.kubeClient.GetService(ctx, svcName, svcNamespace)
	if err != nil {
		return "", err
	}

	commonLabels := map[string]string{
		constants.ControllerServiceLabelKey:  svcName,
		constants.ControllerIdentityLabelKey: constants.ControllerIdentityLabelValue,
	}

	// using the same name as of k8s Service, because if multiple subscriptions have the same sink,
	// then we do not need to create new gateway resources.
	gatewayName := fmt.Sprintf("%s%s", namePrefix, svcName)
	hostName := GenerateWebhookHostName(gatewayName, domain)
	targetHost := GenerateServiceURI(sinkService.Name, sinkService.Namespace)
	jwkURI := credentials.GetJwkURI()
	certIssuer, err := credentials.GetIssuer()
	if err != nil {
		return "", err
	}

	logger.Infof("using jwkURI: %s", jwkURI)
	logger.Infof("using certIssuer: %s", certIssuer)

	// reconcile Istio resources.
	if err = g.ReconcileRequestAuthentication(ctx, gatewayName, certIssuer, jwkURI, commonLabels,
		sinkService.Spec.Selector, subscription); err != nil {
		return "", err
	}
	if err = g.ReconcileAuthorizationPolicy(ctx, gatewayName, commonLabels,
		sinkService.Spec.Selector, subscription); err != nil {
		return "", err
	}
	if err = g.ReconcileVirtualService(ctx, gatewayName, hostName, targetHost, sinkPort,
		commonLabels, subscription, logger); err != nil {
		return "", err
	}

	return GenerateWebhookFQDN(hostName, sinkURL.Path), nil
}

func (g Gateway) ReconcileVirtualService(
	ctx context.Context,
	name string,
	hostName string,
	targetHost string,
	targetPort uint32,
	labels map[string]string,
	subscription eventingv1alpha2.Subscription,
	logger *zap.SugaredLogger,
) error {
	wantVS := virtualservice.NewVirtualService(name, subscription.Namespace,
		virtualservice.WithLabels(labels),
		virtualservice.WithGateway(kymaGateway),
		virtualservice.WithOwnerReference(subscription),
		virtualservice.WithHost(hostName),
		virtualservice.WithDefaultHttpRoute(hostName, targetHost, targetPort),
	)

	// check if VirtualService already exists.
	existingVS, err := g.kubeClient.GetVirtualService(ctx, wantVS.Name, wantVS.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	// check if Spec is same and OwnerReferences is set, then return.
	if existingVS != nil {
		if //reflect.DeepEqual(*wantVS.Spec, *existingVS.Spec) &&
		HasOwnerReference(existingVS.GetOwnerReferences(), subscription) {
			logger.Debugf("sink is already exposed!")
			return nil
		}

		// sync ownerReferences.
		wantVS.OwnerReferences = AppendOwnerReference(existingVS.OwnerReferences, wantVS.OwnerReferences[0])
	}

	// patch apply the VS as unstructured because the GVK is not registered in Scheme.
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(wantVS)
	if err != nil {
		return err
	}

	return g.kubeClient.PatchApplyUnstructured(ctx, &unstructured.Unstructured{Object: obj})
}

func (g Gateway) ReconcileRequestAuthentication(
	ctx context.Context,
	name string,
	issuer string,
	jwkURI string,
	labels map[string]string,
	selectorLabels map[string]string,
	subscription eventingv1alpha2.Subscription,
) error {
	wantRA := requestauthentication.NewRequestAuthentication(name, subscription.Namespace,
		requestauthentication.WithLabels(labels),
		requestauthentication.WithSelectorLabels(selectorLabels),
		requestauthentication.WithOwnerReference(subscription),
		requestauthentication.WithDefaultRules(issuer, jwkURI),
	)

	// check if RequestAuthentication already exists.
	existingRA, err := g.kubeClient.GetRequestAuthentication(ctx, wantRA.Name, wantRA.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	// check if Spec is same and OwnerReferences is set, then return.
	if existingRA != nil {
		if reflect.DeepEqual(wantRA.Spec, existingRA.Spec) &&
			HasOwnerReference(existingRA.GetOwnerReferences(), subscription) {
			return nil
		}
		// else sync ownerReferences.
		wantRA.OwnerReferences = AppendOwnerReference(existingRA.OwnerReferences, wantRA.OwnerReferences[0])
	}

	// patch apply the object as unstructured because the GVK is not registered in Scheme.
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(wantRA)
	if err != nil {
		return err
	}
	return g.kubeClient.PatchApplyUnstructured(ctx, &unstructured.Unstructured{Object: obj})
}

func (g Gateway) ReconcileAuthorizationPolicy(
	ctx context.Context,
	name string,
	labels map[string]string,
	selectorLabels map[string]string,
	subscription eventingv1alpha2.Subscription,
) error {
	wantAP := authorizationpolicy.NewAuthorizationPolicy(name, subscription.Namespace,
		authorizationpolicy.WithLabels(labels),
		authorizationpolicy.WithSelectorLabels(selectorLabels),
		authorizationpolicy.WithOwnerReference(subscription),
		authorizationpolicy.WithDefaultRules(),
	)

	// check if VirtualService already exists.
	existingAP, err := g.kubeClient.GetAuthorizationPolicy(ctx, wantAP.Name, wantAP.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	// check if Spec is same and OwnerReferences is set, then return.
	if existingAP != nil {
		if reflect.DeepEqual(wantAP.Spec, existingAP.Spec) &&
			HasOwnerReference(existingAP.GetOwnerReferences(), subscription) {
			return nil
		}
		// else sync ownerReferences.
		wantAP.OwnerReferences = AppendOwnerReference(existingAP.OwnerReferences, wantAP.OwnerReferences[0])
	}

	// patch apply the object as unstructured because the GVK is not registered in Scheme.
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(wantAP)
	if err != nil {
		return err
	}
	return g.kubeClient.PatchApplyUnstructured(ctx, &unstructured.Unstructured{Object: obj})
}
