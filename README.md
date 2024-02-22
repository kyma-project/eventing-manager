# Eventing Manager

## Status

[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/eventing-manager)](https://api.reuse.software/info/github.com/kyma-project/eventing-manager)

![GitHub tag checks state](https://img.shields.io/github/checks-status/kyma-project/eventing-manager/main?label=eventing-operator&link=https%3A%2F%2Fgithub.com%2Fkyma-project%2Feventing-manager%2Fcommits%2Fmain)

## Overview

Eventing Manager is a standard Kubernetes [operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) that observes the state of Eventing resources and reconciles their state according to the desired state. It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/), which provide a reconcile function responsible for synchronizing resources until the desired state is reached in the cluster.

This project is scaffolded using [Kubebuilder](https://book.kubebuilder.io), and all the Kubebuilder `makefile` helpers mentioned [here](https://book.kubebuilder.io/reference/makefile-helpers.html) can be used.

## Get Started

You need a Kubernetes cluster to run against. You can use [k3d](https://k3d.io/) to get a local cluster for testing, or run against a remote cluster.
> [!NOTE]
> Your controller automatically uses the current context in your kubeconfig file, that is, whatever cluster `kubectl cluster-info` shows.

## Install

1. To install the latest version of the Eventing Manager in your cluster, run:

   ```bash
   kubectl apply -f https://github.com/kyma-project/eventing-manager/releases/latest/download/eventing-manager.yaml
   ```

2. To install the latest version of the default Eventing custom resource (CR) in your cluster, run:

   ```bash
   kubectl apply -f https://github.com/kyma-project/eventing-manager/releases/latest/download/eventing-default-cr.yaml
   ```

## Development


### Prerequisites

- [Go](https://go.dev/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)
- [kustomize](https://kustomize.io/)
- Access to Kubernetes cluster ([k3d](https://k3d.io/) / k8s)

### Run Eventing Manager Locally

1. Install the CRDs into the cluster:

   ```sh
   make install
   ```

2. Run Eventing Manager. It runs in the foreground, so if you want to leave it running, switch to a new terminal.

   ```sh
   make run
   ```

> [!NOTE]
> You can also run this in one step with the command: `make install run`.

### Run Tests

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

### Modify the API Definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs:

```sh
make manifests
```

> [!NOTE]
> Run `make --help` for more information on all potential `make` targets.

For more information, see the [Kubebuilder documentation](https://book.kubebuilder.io/introduction.html).

### Build Container Images

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
> [!NOTE]
> Your controller automatically uses the current context in your kubeconfig file, that is, whatever cluster `kubectl cluster-info` shows.

### Deploy in the Cluster

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

5. [Optional] Install `Eventing` Custom Resource:

   ```sh
   kubectl apply -f config/samples/default.yaml
   ```

6. For EventMesh backend, if the Kyma Kubernetes cluster is managed by Gardener, then the Eventing Manager reads the cluster public domain from the ConfigMap **kube-system/shoot-info**.
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

Undeploy Eventing Manager from the cluster:

   ```sh
   make undeploy
   ```

### Uninstall CRDs

To delete the CRDs from the cluster:

   ```sh
   make uninstall
   ```

## End-to-End Tests

See [hack/e2e/README.md](hack/e2e/README.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)

## Code of Conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## Licensing

See the [License file](./LICENSE)
