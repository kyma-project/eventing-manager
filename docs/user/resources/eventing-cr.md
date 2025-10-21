# Eventing CR

Use the Eventing custom resource (CR) to configure global settings for the Eventing module, such as the backend type, logging level, and resource allocations for the Eventing Publisher Proxy.

You can have only one Eventing CR in your cluster, which must be in the `kyma-system` namespace.

Many fields have default values, so you only have to specify the parameters you want to override. Built-in validation rules guide you when you edit the CR and prevent invalid configurations. For example, you're not allowed to delete an existing backend.

To see the current CRD in YAML format, run:

`kubectl get crd eventings.operator.kyma-project.io -o yaml`

## Sample Custom Resource

Use the following sample CRs as guidance. Each can be applied immediately when you [install](../../contributor/installation.md) Eventing Manager.

- [Default CR - NATS backend](https://github.com/kyma-project/eventing-manager/blob/main/config/samples/default_nats.yaml)
- [Default CR - EventMesh backend](https://github.com/kyma-project/eventing-manager/blob/main/config/samples/default_eventmesh.yaml)

## Eventing Module Status and Event Flow

The Eventing module can have one of the following status values (based on the `status.state` field):

- **Ready**: All resources managed by the Eventing manager are deployed successfully and the Eventing backend is connected.
- **Processing**: The resources managed by the Eventing manager are being created or updated.
- **Warning**: There is a user input misconfiguration. For example:
    - No backend is configured.
    - The backend is NATS, but the NATS module is not installed.
    - The backend is EventMesh, but the EventMesh Secret is missing or invalid.
- **Error**: An error occurred while reconciling the Eventing CR.

This status, combined with your backend configuration, determines how the module processes events. The following table describes how the event flow is affected by the backend configuration and infrastructure health.

| **Specified Backend** | **Backend Infrastructure Status**                                                                         | **Eventing Module Status** | **Event Flow** | **Recommended User Action**                                                                    |
|-----------------------|-----------------------------------------------------------------------------------------------------------|----------------------------|----------------|------------------------------------------------------------------------------------------------|
| None                  | Not applicable                                                                                            | Warning                    | Blocked        | Specify a backend (NATS or EventMesh) in the Eventing CR.                                      |
| NATS/EventMesh        | Processing (while initializing or switching backends)                                                     | Processing                 | Blocked        | Wait for the Eventing module to complete initialization or backend switch.                     |
| NATS                  | Warning (NATS module deletion is blocked)                                                                 | Ready                      | Active         | Investigate and resolve why NATS module can't be deleted.                                      |
| NATS                  | Error (NATS is unavailable or Eventing cannot connect to it)                                              | Warning                    | Blocked        | Ensure the NATS module is added and operational. Check connectivity between Eventing and NATS. |
| NATS                  | Missing (NATS module is not installed)                                                                    | Warning                    | Blocked        | Add the NATS module to your cluster.                                                           |
| EventMesh             | Error (EventMesh Secret is missing or invalid)                                                            | Warning                    | Blocked        | Create or update the EventMesh Secret.                                                         |
| NATS/EventMesh        | Error (backend available but error with, for example, Eventing Publisher Proxy deployment or subscription manager) | Error                      | Blocked        | Review Eventing logs and contact support if the issue persists.                                |

## Custom Resource Parameters

For details, see [operator.kyma-project.io_eventings.yaml](https://github.com/kyma-project/eventing-manager/blob/main/config/crd/bases/operator.kyma-project.io_eventings.yaml).

<!-- The table below was generated automatically -->
<!-- Some special tags (html comments) are at the end of lines due to markdown requirements. -->
<!-- The content between "TABLE-START" and "TABLE-END" will be replaced -->

<!-- TABLE-START -->
### Eventing.operator.kyma-project.io/v1alpha1

**Spec:**

| Parameter | Type | Description |
| ---- | ----------- | ---- |
| **annotations**  | map\[string\]string | Annotations allows to add annotations to resources. |
| **backend**  | object | Backend defines the active backend used by Eventing. |
| **backend.&#x200b;config**  | object | Config defines configuration for the Eventing backend. |
| **backend.&#x200b;config.&#x200b;domain**  | string | Domain defines the cluster public domain used to configure the EventMesh Subscriptions and their corresponding ApiRules. |
| **backend.&#x200b;config.&#x200b;eventMeshSecret**  | string | EventMeshSecret defines the namespaced name of the Kubernetes Secret containing EventMesh credentials. The format of name is "namespace/name". |
| **backend.&#x200b;config.&#x200b;eventTypePrefix**  | string | EventTypePrefix defines the prefix for all event types. |
| **backend.&#x200b;config.&#x200b;natsMaxMsgsPerTopic**  | integer | NATSMaxMsgsPerTopic limits how many messages in the NATS stream to retain per subject. |
| **backend.&#x200b;config.&#x200b;natsStreamMaxSize**  | \{integer or string\} | NATSStreamMaxSize defines the maximum storage size for stream data. |
| **backend.&#x200b;config.&#x200b;natsStreamReplicas**  | integer | NATSStreamReplicas defines the number of replicas for the stream. |
| **backend.&#x200b;config.&#x200b;natsStreamStorageType**  | string | NATSStreamStorageType defines the storage type for stream data. |
| **backend.&#x200b;type** (required) | string | Type defines which backend to use. The value is either `EventMesh`, or `NATS`. |
| **labels**  | map\[string\]string | Labels allows to add Labels to resources. |
| **logging**  | object | Logging defines the log level for eventing-manager. |
| **logging.&#x200b;logLevel**  | string | LogLevel defines the log level. |
| **publisher**  | object | Publisher defines the configurations for eventing-publisher-proxy. |
| **publisher.&#x200b;replicas**  | object | Replicas defines the scaling min/max for eventing-publisher-proxy. |
| **publisher.&#x200b;replicas.&#x200b;max**  | integer | Max defines maximum number of replicas. |
| **publisher.&#x200b;replicas.&#x200b;min**  | integer | Min defines minimum number of replicas. |
| **publisher.&#x200b;resources**  | object | Resources defines resources for eventing-publisher-proxy. |
| **publisher.&#x200b;resources.&#x200b;claims**  | \[\]object | Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container.  This is an alpha field and requires enabling the DynamicResourceAllocation feature gate.  This field is immutable. It can only be set for containers. |
| **publisher.&#x200b;resources.&#x200b;claims.&#x200b;name** (required) | string | Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container. |
| **publisher.&#x200b;resources.&#x200b;claims.&#x200b;request**  | string | Request is the name chosen for a request in the referenced claim. If empty, everything from the claim is made available, otherwise only the result of this request. |
| **publisher.&#x200b;resources.&#x200b;limits**  | map\[string\]\{integer or string\} | Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |
| **publisher.&#x200b;resources.&#x200b;requests**  | map\[string\]\{integer or string\} | Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |

**Status:**

| Parameter | Type | Description |
| ---- | ----------- | ---- |
| **activeBackend** (required) | string | ActiveBackend shows the backend currently used by the Eventing module. |
| **conditions**  | \[\]object | Conditions contains the list of status conditions for this resource. |
| **conditions.&#x200b;lastTransitionTime** (required) | string | lastTransitionTime is the last time the condition transitioned from one status to another. This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable. |
| **conditions.&#x200b;message** (required) | string | message is a human readable message indicating details about the transition. This may be an empty string. |
| **conditions.&#x200b;observedGeneration**  | integer | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. |
| **conditions.&#x200b;reason** (required) | string | reason contains a programmatic identifier indicating the reason for the condition's last transition. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API. The value should be a CamelCase string. This field may not be empty. |
| **conditions.&#x200b;status** (required) | string | status of the condition, one of True, False, Unknown. |
| **conditions.&#x200b;type** (required) | string | type of condition in CamelCase or in foo.example.com/CamelCase. |
| **publisherService**  | string | PublisherService is the Kubernetes Service for the Eventing Publisher Proxy. |
| **specHash** (required) | integer | BackendConfigHash is a hash of the spec.backend configuration, used internally to detect changes. |
| **state** (required) | string | State defines the overall status of the Eventing custom resource. It can be `Ready`, `Processing`, `Error`, or `Warning`. |

<!-- TABLE-END -->
