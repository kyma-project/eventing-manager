# Configuration

The CustomResourceDefinition (CRD) `eventings.operator.kyma-project.io` describes the Eventing custom resource (CR) in detail. To show the current CRD, run the following command:

   ```shell
   kubectl get crd nats.operator.kyma-project.io -o yaml
   ```

View the complete [Eventing CRD](https://github.com/kyma-project/eventing-manager/blob/main/config/crd/bases/operator.kyma-project.io_eventings.yaml) including detailed descriptions for each field.

The CRD is equipped with validation rules and defaulting, so the CR is automatically filled with sensible defaults. You can override the defaults. The validation rules provide guidance when you edit the CR.

## Examples

Use the following sample CRs as guidance. Each can be applied immediately when you [install](../contributor/installation.md) the Eventing Manager.

- [Default CR - NATS backend](https://github.com/kyma-project/eventing-manager/blob/main/config/samples/default.yaml)
- [Default CR - EventMesh backend](https://github.com/kyma-project/eventing-manager/blob/main/config/samples/default_eventmesh.yaml)

## Reference

