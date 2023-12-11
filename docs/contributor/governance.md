# Governance

Some quality aspects are covered by automated verification, so you must execute tooling locally before a commit.
This document guides you through the development flow.

## Modifying the API Definitions

This project uses the [controller-gen](https://book.kubebuilder.io/reference/controller-gen.html) tool provided by [Kubebuilder](https://book.kubebuilder.io/).
To modify the API definitions, you must adapt the ["marker comments"](https://book.kubebuilder.io/reference/markers.html) in the Go code.

Additionally, this project uses [validation markers](https://book.kubebuilder.io/reference/markers/crd-validation.html) to provide CRD validation and defaulting with [OpenAPI v3 schemas](https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.0.md#schemaObject).
The rules are written using the [Common Expression Language](https://github.com/google/cel-spec).
For further information and examples, look to the [Kubernetes documentation](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#validation) of validation rules and the CEL [language definition](https://github.com/google/cel-spec/blob/v0.10.0/doc/langdef.md).

After the modifications, run the following command to generate the new manifests, such as custom resources (CRs) and CustomResourceDefinitions (CRDs):

```sh
make manifests
```

If changes to the `runtime.Object` interface were made, the `DeepCopy` functions must be updated as well.
The generation is controlled by [Kubebuilder markers](https://book.kubebuilder.io/reference/markers/object.html?highlight=deep#objectdeepcopy).

```sh
make generate
```

## Source Code Linting

The quality of this project is ensured by source code linting using [golangci-lint](https://golangci-lint.run/).

To fix common lint issues, run the following command:

```sh
make imports-local
make fmt
```

To run thorough lint checking, execute the following command:

```sh
make lint-thoroughly
```

If necessary, lint warnings can be ignored. However, if this is desired you must provide a comment explaining the reason. Use the following format:

`//no-lint:<LINTER> // <REASON>`
