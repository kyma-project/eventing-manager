# Configuration

The CustomResourceDefinition (CRD) `eventings.operator.kyma-project.io` describes the Eventing custom resource (CR) in detail. To show the current CRD, run the following command:

   ```shell
   kubectl get crd eventings.operator.kyma-project.io -o yaml
   ```

View the complete [Eventing CRD](https://github.com/kyma-project/eventing-manager/blob/main/config/crd/bases/operator.kyma-project.io_eventings.yaml) including detailed descriptions for each field.

The CRD is equipped with validation rules and defaulting, so the CR is automatically filled with sensible defaults. You can override the defaults. The validation rules provide guidance when you edit the CR.

## Examples

Use the following sample CRs as guidance. Each can be applied immediately when you [install](../contributor/installation.md) Eventing Manager.

- [Default CR - NATS backend](https://github.com/kyma-project/eventing-manager/blob/main/config/samples/default.yaml)
- [Default CR - EventMesh backend](https://github.com/kyma-project/eventing-manager/blob/main/config/samples/default_eventmesh.yaml)

## Reference

<!-- The table below was generated automatically -->
<!-- Some special tags (html comments) are at the end of lines due to markdown requirements. -->
<!-- The content between "TABLE-START" and "TABLE-END" will be replaced -->

<!-- TABLE-START -->
### Eventing.operator.kyma-project.io/v1alpha1

**Spec:**

| Parameter | Type | Description |
| ---- | ----------- | ---- |
| **annotations**  | map\[string\]string | Annotations allows to add annotations to resources. |
| **backend** (required) | object | Backend defines the active backend used by Eventing. |
| **backend.&#x200b;config**  | object | Config defines configuration for eventing backend. |
| **backend.&#x200b;config.&#x200b;domain**  | string | Domain defines the cluster public domain used to configure the EventMesh Subscriptions and their corresponding ApiRules. |
| **backend.&#x200b;config.&#x200b;eventMeshSecret**  | string | EventMeshSecret defines the namespaced name of K8s Secret containing EventMesh credentials. The format of name is "namespace/name". |
| **backend.&#x200b;config.&#x200b;eventTypePrefix**  | string |  |
| **backend.&#x200b;config.&#x200b;natsMaxMsgsPerTopic**  | integer | NATSMaxMsgsPerTopic limits how many messages in the NATS stream to retain per subject. |
| **backend.&#x200b;config.&#x200b;natsStreamMaxSize**  | \{integer or string\} | NATSStreamMaxSize defines the maximum storage size for stream data. |
| **backend.&#x200b;config.&#x200b;natsStreamReplicas**  | integer | NATSStreamReplicas defines the number of replicas for stream. |
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
| **publisher.&#x200b;resources.&#x200b;claims**  | \[\]object | Claims lists the names of resources, defined in spec.resourceClaims, that are used by this container. 
 This is an alpha field and requires enabling the DynamicResourceAllocation feature gate. 
 This field is immutable. It can only be set for containers. |
| **publisher.&#x200b;resources.&#x200b;claims.&#x200b;name** (required) | string | Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container. |
| **publisher.&#x200b;resources.&#x200b;limits**  | map\[string\]\{integer or string\} | Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |
| **publisher.&#x200b;resources.&#x200b;requests**  | map\[string\]\{integer or string\} | Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. Requests cannot exceed Limits. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/ |

**Status:**

| Parameter | Type | Description |
| ---- | ----------- | ---- |
| **activeBackend** (required) | string |  |
| **conditions**  | \[\]object | Condition contains details for one aspect of the current state of this API Resource. --- This struct is intended for direct use as an array at the field path .status.conditions.  For example, 
 type FooStatus struct{ // Represents the observations of a foo's current state. // Known .status.conditions.type are: "Available", "Progressing", and "Degraded" // +patchMergeKey=type // +patchStrategy=merge // +listType=map // +listMapKey=type Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"` 
 // other fields } |
| **conditions.&#x200b;lastTransitionTime** (required) | string | lastTransitionTime is the last time the condition transitioned from one status to another. This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable. |
| **conditions.&#x200b;message** (required) | string | message is a human readable message indicating details about the transition. This may be an empty string. |
| **conditions.&#x200b;observedGeneration**  | integer | observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance. |
| **conditions.&#x200b;reason** (required) | string | reason contains a programmatic identifier indicating the reason for the condition's last transition. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API. The value should be a CamelCase string. This field may not be empty. |
| **conditions.&#x200b;status** (required) | string | status of the condition, one of True, False, Unknown. |
| **conditions.&#x200b;type** (required) | string | type of condition in CamelCase or in foo.example.com/CamelCase. --- Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be useful (see .node.status.conditions), the ability to deconflict is important. The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt) |
| **specHash** (required) | integer |  |
| **state** (required) | string |  |

<!-- TABLE-END -->