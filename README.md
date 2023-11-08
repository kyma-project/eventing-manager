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

## Install

1. To install the latest version of the Eventing manager on your cluster, run:

   ```bash
   kubectl apply -f https://github.com/kyma-project/eventing-manager/releases/latest/download/eventing-manager.yaml
   ```

2. To install the latest version of the default Eventing CR on your cluster, run:

   ```bash
   kubectl apply -f https://github.com/kyma-project/eventing-manager/releases/latest/download/eventing_default_cr.yaml
   ```

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
make generate-and-test
```

### Linting

1. Fix common lint issues:

   ```sh
   make imports
   make fmt
   make lint
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

### Deploy on the cluster

1. Download Go packages:

   ```sh
   go mod vendor && go mod tidy
   ```

2. Install the CRDs to the cluster:

   ```sh
   make install
   ```

3. Build and push your image to the location specified by `IMG`:

   ```sh
   make docker-build docker-push IMG=<container-registry>/eventing-manager:<tag>
   ```

4. Deploy the `eventing-manager` controller to the cluster:

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

### Undeploy Eventing Manager

Undeploy the Eventing Manager from the cluster:

   ```sh
   make undeploy
   ```

### Undeploy Eventing Manager

To delete the CRDs from the cluster:

   ```sh
   make uninstall
   ```

### Deploy Eventing Manager module with [Kyma Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager/tree/main)

1. Deploy the Lifecycle Manager to the Kubernetes cluster:

   ```shell
   kyma alpha deploy
   ```

2. Apply the Eventing module template to the Kubernetes cluster:

   > **NOTE:** You can get the latest released [module template](https://github.com/kyma-project/eventing-manager/releases/latest/download/module-template.yaml), or you can use the module template from the artifacts of `eventing-module-build` job either from the `main` branch or from your pull request.

   ```sh
   kubectl apply -f module-template.yaml
   ```

3. Enable the Eventing module:

   ```sh
   kyma alpha enable module eventing -c fast -n kyma-system
   ```

4. If you want to verify whether your Eventing module is deployed properly, perform the following checks:

   - Check if the Eventing resource has the ready state:

     ```shell
     kubectl get -n kyma-system eventing
     ```

   - Check if the Kyma resource has the ready state:

     ```shell
     kubectl get -n kyma-system kyma
     ```

### Uninstall Eventing Manager module with [Kyma Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager/tree/main)

1. Delete Eventing Custom Resource (CR) from the Kubernetes cluster (if exists):

    ```sh
    kubectl delete -n kyma-system eventing eventing
    ```

2. Disable the Eventing module:

   ```sh
   kyma alpha disable module eventing
   ```

3. Delete the Eventing module template:

   ```sh
   kubectl get moduletemplates -A
   kubectl delete moduletemplate -n <NAMESPACE> <NAME>
   ```

4. Check whether your Eventing module is uninstalled properly:

   - Make sure that the Eventing Custom Resource (CR) does not exist. If it exists, then check the status of Eventing CR:

     ```shell
     kubectl get -n kyma-system eventing eventing -o yaml
     ```

   - Check if the Kyma resource has the ready state:

     ```shell
     kubectl get -n kyma-system kyma
     ```

## End-to-End Tests

See [hack/e2e/README.md](hack/e2e/README.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)

## Code of Conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## Licensing

See the [License file](./LICENSE)
