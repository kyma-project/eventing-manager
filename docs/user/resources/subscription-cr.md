# Subscription CR

Use the Subscription custom resource (CR) to describe the kind of data and the format used to subscribe to events. You specify the event types and the target endpoint for event delivery.

When you define a Subscription, the Eventing Manager configures a dedicated consumer in the chosen backend for each event type you specify. These consumers are push-based, meaning the backend delivers events to your subscriber's sink as soon as they become available.

The following components use the Subscription CR:

- [Eventing Manager](../../user/README.md#eventing-manager): Reconciles on Subscriptions and creates a connection between subscribers and the Eventing backend.
- [Eventing Publisher Proxy](../../user/README.md#eventing-publisher-proxy): Reads the Subscriptions to find out how events are used for each Application.

You must delete all Subscription CRs before you can delete the Eventing module.

To see the current CRD in YAML format, run:

`kubectl get crd subscriptions.eventing.kyma-project.io -o yaml`

## Sample Custom Resource

This sample Subscription CR subscribes to an event called `order.created.v1`.

```yaml
apiVersion: eventing.kyma-project.io/v1alpha2
kind: Subscription
metadata:
  name: test
  namespace: test
spec:
  typeMatching: standard
  source: commerce
  types:
    - order.created.v1
  sink: http://test.test.svc.cluster.local
  config:
    maxInFlightMessages: "10"
```

## Subscription Status

The `status.ready` field shows the overall readiness of the Subscription. If `false`, check the `status.conditions` array for details.

## Custom Resource Parameters

> [!NOTE]
> If the Subscription CR and the target subscriber aren't in the same namespace, you must specify the **sink.ref.namespace**.
> 
> Eventing backends might not support certain characters in event names defined under **spec.type**. If you use unsupported characters, the Eventing module removes them. For details, see [Event Name Cleanup](../evnt-event-names.md#event-name-cleanup).

<!-- TABLE-START -->
### Subscription.eventing.kyma-project.io/v1alpha2

**Spec:**

| Parameter | Type | Description |
| ---- | ----------- | ---- |
| **config**  | map\[string\]string | Map of configuration options that will be applied on the backend. |
| **id**  | string | Unique identifier of the Subscription, read-only. |
| **sink** (required) | string | Kubernetes Service that should be used as a target for the events that match the Subscription. Must exist in the same Namespace as the Subscription. |
| **source** (required) | string | Defines the origin of the event. |
| **typeMatching**  | string | Defines how types should be handled.<br /> - `standard`: backend-specific logic will be applied to the configured source and types.<br /> - `exact`: no further processing will be applied to the configured source and types. |
| **types** (required) | \[\]string | List of event types that will be used for subscribing on the backend. |

**Status:**

| Parameter | Type | Description |
| ---- | ----------- | ---- |
| **backend**  | object | Backend-specific status which is applicable to the active backend only. |
| **backend.&#x200b;apiRuleName**  | string | Name of the APIRule which is used by the Subscription. |
| **backend.&#x200b;emsSubscriptionStatus**  | object | Status of the Subscription as reported by EventMesh. |
| **backend.&#x200b;emsSubscriptionStatus.&#x200b;lastFailedDelivery**  | string | Timestamp of the last failed delivery. |
| **backend.&#x200b;emsSubscriptionStatus.&#x200b;lastFailedDeliveryReason**  | string | Reason for the last failed delivery. |
| **backend.&#x200b;emsSubscriptionStatus.&#x200b;lastSuccessfulDelivery**  | string | Timestamp of the last successful delivery. |
| **backend.&#x200b;emsSubscriptionStatus.&#x200b;status**  | string | Status of the Subscription as reported by the backend. |
| **backend.&#x200b;emsSubscriptionStatus.&#x200b;statusReason**  | string | Reason for the current status. |
| **backend.&#x200b;emsTypes**  | \[\]object | List of mappings from event type to EventMesh compatible types. Used only with EventMesh as the backend. |
| **backend.&#x200b;emsTypes.&#x200b;eventMeshType** (required) | string | Event type that is used on the EventMesh backend. |
| **backend.&#x200b;emsTypes.&#x200b;originalType** (required) | string | Event type that was originally used to subscribe. |
| **backend.&#x200b;emshash**  | integer | Hash used to identify an EventMesh Subscription retrieved from the server without the WebhookAuth config. |
| **backend.&#x200b;ev2hash**  | integer | Checksum for the Subscription custom resource. |
| **backend.&#x200b;eventMeshLocalHash**  | integer | Hash used to identify an EventMesh Subscription posted to the server without the WebhookAuth config. |
| **backend.&#x200b;externalSink**  | string | Webhook URL used by EventMesh to trigger subscribers. |
| **backend.&#x200b;failedActivation**  | string | Provides the reason if a Subscription failed activation in EventMesh. |
| **backend.&#x200b;types**  | \[\]object | List of event type to consumer name mappings for the NATS backend. |
| **backend.&#x200b;types.&#x200b;consumerName**  | string | Name of the JetStream consumer created for the event type. |
| **backend.&#x200b;types.&#x200b;originalType** (required) | string | Event type that was originally used to subscribe. |
| **backend.&#x200b;webhookAuthHash**  | integer | Hash used to identify the WebhookAuth of an EventMesh Subscription existing on the server. |
| **conditions**  | \[\]object | Current state of the Subscription. |
| **conditions.&#x200b;lastTransitionTime**  | string | Defines the date of the last condition status change. |
| **conditions.&#x200b;message**  | string | Provides more details about the condition status change. |
| **conditions.&#x200b;reason**  | string | Defines the reason for the condition status change. |
| **conditions.&#x200b;status** (required) | string | Status of the condition. The value is either `True`, `False`, or `Unknown`. |
| **conditions.&#x200b;type**  | string | Short description of the condition. |
| **ready** (required) | boolean | Overall readiness of the Subscription. |
| **types** (required) | \[\]object | List of event types after cleanup for use with the configured backend. |
| **types.&#x200b;cleanType** (required) | string | Event type after it was cleaned up from backend compatible characters. |
| **types.&#x200b;originalType** (required) | string | Event type as specified in the Subscription spec. |

<!-- TABLE-END -->
