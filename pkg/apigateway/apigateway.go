package apigateway

import (
	"context"
	"fmt"
	"net/url"

	recerrors "github.com/kyma-project/eventing-manager/internal/controller/errors"
	"github.com/kyma-project/eventing-manager/pkg/apigateway/authorizationpolicy"
	"github.com/kyma-project/eventing-manager/pkg/apigateway/requestauthentication"
	"github.com/kyma-project/eventing-manager/pkg/apigateway/virtualservice"
	"github.com/kyma-project/eventing-manager/pkg/constants"
	"github.com/kyma-project/eventing-manager/pkg/k8s"
	"github.com/kyma-project/eventing-manager/pkg/utils"
	eventingv1alpha2 "github.com/kyma-project/kyma/components/eventing-controller/api/v1alpha2"
	"golang.org/x/xerrors"
)

const (
	suffixLength       = 10
	externalHostPrefix = "web"
	externalSinkScheme = "https"
	namePrefix         = "sub-sink-"
	kymaGateway        = "kyma-gateway.kyma-system.svc.cluster.local"
)

var _ APIGateway = Gateway{}

//go:generate mockery --name=APIGateway --outpkg=mocks --case=underscore
type APIGateway interface {
	ExposeSink(context.Context, eventingv1alpha2.Subscription, string, string, string) (string, error)
}

type Gateway struct {
	kubeClient k8s.Client
}

func NewGateway(kubeClient k8s.Client) *Gateway {
	return &Gateway{
		kubeClient: kubeClient,
	}
}

func (g Gateway) ExposeSink(ctx context.Context, subscription eventingv1alpha2.Subscription, domain, issuer, jwkURI string) (string, error) {
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
	gatewayName := fmt.Sprintf("%s-%s", namePrefix, svcName)
	hostName := GenerateWebhookHostName(gatewayName, domain)
	targetHost := GenerateServiceURI(sinkService.Name, sinkService.Namespace)

	if err = g.ReconcileVirtualService(ctx, gatewayName, hostName, targetHost, sinkPort,
		commonLabels, subscription); err != nil {
		return "", err
	}
	if err = g.ReconcileRequestAuthentication(ctx, gatewayName, issuer, jwkURI, commonLabels,
		sinkService.Spec.Selector, subscription); err != nil {
		return "", err
	}
	if err = g.ReconcileAuthorizationPolicy(ctx, gatewayName, commonLabels,
		sinkService.Spec.Selector, subscription); err != nil {
		return "", err
	}

	return hostName, nil
}

func (g Gateway) ReconcileVirtualService(
	ctx context.Context,
	name string,
	hostName string,
	targetHost string,
	targetPort uint32,
	labels map[string]string,
	subscription eventingv1alpha2.Subscription,
) error {
	wantVS := virtualservice.NewVirtualService(name, subscription.Namespace,
		virtualservice.WithLabels(labels),
		virtualservice.WithGateway(kymaGateway),
		virtualservice.WithOwnerReference(subscription),
		virtualservice.WithHost(hostName),
		virtualservice.WithDefaultHttpRoute(hostName, targetHost, targetPort),
	)

	//// check if VirtualService already exists.
	//existingVS, err := g.kubeClient.GetVirtualService(ctx, wantVS.Name, wantVS.Namespace)
	//if err != nil && !k8serrors.IsNotFound(err) {
	//	return err
	//}
	//
	//// check if Spec is same and OwnerReferences is set, then return.
	//if existingVS != nil &&
	//	reflect.DeepEqual(wantVS.Spec, existingVS.Spec) &&
	//	HasOwnerReference(existingVS.GetOwnerReferences(), subscription) {
	//	return nil
	//}
	//
	//// sync ownerReferences.
	//wantVS.OwnerReferences = AppendOwnerReference(existingVS.OwnerReferences, wantVS.OwnerReferences[0])

	// patch apply the VS.
	return g.kubeClient.PatchApply(ctx, wantVS)
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

	//// check if VirtualService already exists.
	//existingRA, err := g.kubeClient.GetRequestAuthentication(ctx, wantRA.Name, wantRA.Namespace)
	//if err != nil && !k8serrors.IsNotFound(err) {
	//	return err
	//}
	//
	//// check if Spec is same and OwnerReferences is set, then return.
	//if existingRA != nil &&
	//	reflect.DeepEqual(wantRA.Spec, existingRA.Spec) &&
	//	HasOwnerReference(existingRA.GetOwnerReferences(), subscription) {
	//	return nil
	//}
	//
	//// sync ownerReferences.
	//wantRA.OwnerReferences = AppendOwnerReference(existingRA.OwnerReferences, wantRA.OwnerReferences[0])

	// patch apply the object.
	return g.kubeClient.PatchApply(ctx, wantRA)
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

	//// check if VirtualService already exists.
	//existingAP, err := g.kubeClient.GetAuthorizationPolicy(ctx, wantAP.Name, wantAP.Namespace)
	//if err != nil && !k8serrors.IsNotFound(err) {
	//	return err
	//}
	//
	//// check if Spec is same and OwnerReferences is set, then return.
	//if existingAP != nil &&
	//	reflect.DeepEqual(wantAP.Spec, existingAP.Spec) &&
	//	HasOwnerReference(existingAP.GetOwnerReferences(), subscription) {
	//	return nil
	//}
	//
	//// sync ownerReferences.
	//wantAP.OwnerReferences = AppendOwnerReference(existingAP.OwnerReferences, wantAP.OwnerReferences[0])

	// patch apply the object.
	return g.kubeClient.PatchApply(ctx, wantAP)
}
