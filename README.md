# Eventing Manager

## Status

[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/eventing-manager)](https://api.reuse.software/info/github.com/kyma-project/eventing-manager)

![GitHub tag checks state](https://img.shields.io/github/checks-status/kyma-project/eventing-manager/main?label=eventing-operator&link=https%3A%2F%2Fgithub.com%2Fkyma-project%2Feventing-manager%2Fcommits%2Fmain)

## Overview

Eventing Manager is a standard Kubernetes [operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) that observes the state of Eventing resources and reconciles their state according to the desired state. It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/), which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

This project is scaffolded using [Kubebuilder](https://book.kubebuilder.io), and all the Kubebuilder `makefile` helpers mentioned [here](https://book.kubebuilder.io/reference/makefile-helpers.html) can be used.

## Get started

You need a Kubernetes cluster to run against. You can use [k3d](https://k3d.io/) to get a local cluster for testing, or run against a remote cluster.
> **Note:** Your controller automatically uses the current context in your kubeconfig file, that is, whatever cluster `kubectl cluster-info` shows.

## Development

### Prerequisites

- [Go](https://go.dev/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)
- [kustomize](https://kustomize.io/)
- Access to Kubernetes cluster ([k3d](https://k3d.io/) / k8s)

### Run Eventing Manager locally

1. Install the CRDs into the cluster:

   ```sh
   make install
   ```

2. Run Eventing Manager. It runs in the foreground, so if you want to leave it running, switch to a new terminal.

   ```sh
   make run
   ```

> **NOTE:** You can also run this in one step with the command: `make install run`.

### Run tests

Run the unit and integration tests:

```sh
make test-only
```

### Linting

1. Fix common lint issues:

   ```sh
   make imports-local
   make fmt-local
   ```

2. Run lint check:

   ```sh
   make lint-thoroughly
   ```

### Modify the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs:

```sh
make manifests
```

> **NOTE:** Run `make --help` for more information on all potential `make` targets.

For more information, see the [Kubebuilder documentation](https://book.kubebuilder.io/introduction.html).

### Build container images

Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<container-registry>/eventing-manager:<tag> # If using docker, <container-registry> is your username.
```

> **NOTE**: For MacBook M1 devices, run:

```sh
make docker-buildx IMG=<container-registry>/eventing-manager:<tag>
```

## Deployment

You need a Kubernetes cluster to run against. You can use [k3d](https://k3d.io/) to get a local cluster for testing, or run against a remote cluster.
> **Note:** Your controller automatically uses the current context in your kubeconfig file, that is, whatever cluster `kubectl cluster-info` shows.

### Deploy Eventing Manager on the cluster

1. Install the CRDs to the cluster:

   ```sh
   make install
   ```

2. Build and push your image to the location specified by `IMG`:

   ```sh
   make docker-build docker-push IMG=<container-registry>/eventing-manager:<tag>
   ```

3. Deploy the `eventing-manager` controller to the cluster:

   ```sh
   make deploy IMG=<container-registry>/eventing-manager:<tag>
   ```

4. [Optional] Install `Eventing` Custom Resource:

   ```sh
   kubectl apply -f config/samples/default.yaml
   ```

5. For EventMesh backend, if the Kyma Kubernetes cluster is managed by Gardener, then the Eventing Manager reads the cluster public domain from the ConfigMap **kube-system/shoot-info**.
   Otherwise, set the **spec.backend.config.domain** to the cluster public domain in the `eventing` custom resource; for example:
   ```yaml
   spec:
     backend:
       type: "EventMesh"
       config:
         domain: "example.domain.com"
         eventMeshSecret: "kyma-system/eventing-backend"
         eventTypePrefix: "sap.kyma.custom"
   ```

### Remove Eventing Manager from the cluster

1. To undeploy the Eventing Manager from the cluster, run:

   ```sh
   make undeploy
   ```

2. To delete the CRDs from the cluster, run:

   ```sh
   make uninstall
   ```

### Deploy with [Kyma Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager/tree/main)

1. Deploy the Lifecycle Manager and Module Manager to the Control Plane cluster:

   ```shell
   kyma alpha deploy
   ```

  > **NOTE**: For single-cluster mode, edit the Lifecycle Manager role to give it access to all resources. Run `kubectl edit clusterrole lifecycle-manager-manager-role` and have the following under `rules`:

   ```shell
   - apiGroups:
     - "*"
     resources:
     - "*"
     verbs:
     - "*"
   ```

2. Prepare OCI container registry:

   Supported registries are GitHub, DockerHub, GCP, or local registry.
   If you do not have a registry available, read the following resources to guide you through the setup:

   - Lifecycle manager [provision-cluster-and-registry](https://github.com/kyma-project/lifecycle-manager/blob/main/docs/developer-tutorials/provision-cluster-and-registry.md) documentation
   - [GitHub container registry documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry). Change the visibility of a GH package to public if you don't provide a registry secret.

3. Generate a module template and push the container image by running the following command in the project root directory:

   ```sh
   kyma alpha create module -n kyma-project.io/module/eventing --version 0.0.1 --registry ghcr.io/{GH_USERNAME}/eventing-manager -c {REGISTRY_USER_NAME}:{REGISTRY_AUTH_TOKEN} -w
   ```

   In the command, the GH container registry sample is used. Replace GH_USERNAME=REGISTRY_USER_NAME and REGISTRY_AUTH_TOKEN with the GH username and token/password respectively.

   The command generates a ModuleTemplate `template.yaml` file in the project folder.

  > **NOTE:** Change `template.yaml` content with `spec.target=remote` to `spec.target=control-plane` for **single-cluster** mode as follows:

   ```yaml
   spec:
     target: control-plane
     channel: regular
   ```

4. Apply the module template to the Kubernetes cluster:

   ```sh
   kubectl apply -f template.yaml
   ```

5. Deploy the `eventing` module by adding it to the `kyma` custom resource `spec.modules`:

   ```sh
   kubectl edit -n kyma-system kyma default-kyma
   ```

   The spec part should contain the following:

   ```yaml
   ...
   spec:
     modules:
     - name: eventing
   ...
   ```

6. Check whether your module is deployed properly:

   Check if the `eventing` resource has the ready state:

   ```shell
   kubectl get -n kyma-system eventing
   ```

   Check if the Kyma resource has the ready state:

   ```shell
   kubectl get -n kyma-system kyma
   ```

   If it doesn't have the ready state, troubleshoot it by checking the Pods in the `eventing-manager-system` Namespace, where the module is installed:

   ```shell
   kubectl get pods -n eventing-manager-system
   ```

### Uninstall Eventing Manager with [Kyma Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager/tree/main)

1. Delete Eventing from `kyma` resource `spec.modules` `kubectl edit -n kyma-system kyma default-kyma`:

2. Check whether the `eventing` resource and module Namespace were deleted:

   ```shell
   kubectl get -n kyma-system eventing
   ```

## End-to-End Tests

See [hack/e2e/README.md](hack/e2e/README.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)

## Code of Conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## Licensing

See the [License file](./LICENSE)
