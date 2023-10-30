[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/eventing-manager)](https://api.reuse.software/info/github.com/kyma-project/eventing-manager)

# Eventing Manager

Manages the lifecycle of Eventing resources.

## Description

It is a standard Kubernetes operator which observes the state of Eventing resources and reconciles its state according to the desired state.

## Getting Started

You’ll need a Kubernetes cluster to run against. You can use [k3d](https://k3d.io/) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### How it works

This project follows the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/), which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

This project is scaffolded using [Kubebuilder](https://book.kubebuilder.io), and all the Kubebuilder `makefile` helpers mentioned [here](https://book.kubebuilder.io/reference/makefile-helpers.html) can be used.

## Development

### Prerequisites

- [Go](https://go.dev/)
- [Docker](https://www.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/)
- [kustomize](https://kustomize.io/)
- Access to Kubernetes cluster ([k3d](https://k3d.io/) / k8s)

### Running locally

1. Install the CRDs into the cluster:

   ```sh
   make install
   ```

1. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

   ```sh
   make run
   ```

**NOTE:** You can also run this in one step by running: `make install run`

### Running tests

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

### Modifying the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

For more information, see the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

### Build container images

Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<container-registry>/eventing-manager:<tag> # If using docker, <container-registry> is your username.
```

**NOTE**: For MacBook M1 devices, run:

```sh
make docker-buildx IMG=<container-registry>/eventing-manager:<tag>
```

## Deployment

You need a Kubernetes cluster to run against. You can use [k3d](https://k3d.io/) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller automatically uses the current context in your kubeconfig file (that is, whatever cluster `kubectl cluster-info` shows).

### Deploying on the cluster

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

5. [Optional] For EventMesh backend, set env var `DOMAIN` to the cluster domain in the deployment resource; for example:

   ```yaml
   - name: DOMAIN
     value: {CLUSTER_NAME}.kymatunas.shoot.canary.k8s-hana.ondemand.com
   ```

### Removing from the cluster

1. To undeploy the controller from the cluster, run:

   ```sh
   make undeploy
   ```

2. To delete the CRDs from the cluster, run:

   ```sh
   make uninstall
   ```

### Deploying with [Kyma Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager/tree/main)

1. Deploy the Lifecycle Manager and Module Manager to the Control Plane cluster:

   ```shell
   kyma alpha deploy
   ```

   **NOTE**: For single-cluster mode, edit the Lifecycle Manager role to give access to all resources. Run `kubectl edit clusterrole lifecycle-manager-manager-role` and have the following under `rules`:

   ```shell
   - apiGroups:
     - "*"
     resources:
     - "*"
     verbs:
     - "*"
   ```

2. Prepare OCI container registry:

   Supported registries are Github, DockerHub, GCP or local registry.
   If you do not have a registry available, hereread the following resources to guide you through the setup:

   - Lifecycle manager [provision-cluster-and-registry](https://github.com/kyma-project/lifecycle-manager/blob/main/docs/developer/provision-cluster-and-registry.md) documentation
   - [Github container registry documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry). Change the visibility of a GH package to public if you don't provide a registry secret.

3. Generate a module template and push the container image by running the following command in the project root director:

   ```sh
   kyma alpha create module -n kyma-project.io/module/eventing --version 0.0.1 --registry ghcr.io/{GH_USERNAME}/eventing-manager -c {REGISTRY_USER_NAME}:{REGISTRY_AUTH_TOKEN} -w
   ```

   In the command, the GH container registry sample is used. Replace GH_USERNAME=REGISTRY_USER_NAME and REGISTRY_AUTH_TOKEN with the GH username and token/password respectively.

   The command generates a ModuleTemplate `template.yaml` file in the project folder.

   **NOTE:** Change `template.yaml` content with `spec.target=remote` to `spec.target=control-plane` for **single-cluster** mode as follows:

   ```yaml
   spec:
     target: control-plane
     channel: regular
   ```

4. Apply the module template to the K8s cluster:

   ```sh
   kubectl apply -f template.yaml
   ```

5. Deploy the `eventing` module by adding it to the `kyma` custom resource `spec.modules`:

   ```sh
   kubectl edit -n kyma-system kyma default-kyma
   ```

   The spec part should have the following:

   ```yaml
   ...
   spec:
     modules:
     - name: eventing
   ...
   ```

6. Check whether your module is deployed properly:

   Check eventing resource if it has ready state:

   ```shell
   kubectl get -n kyma-system eventing
   ```

   Check if the Kyma resource has the ready state:

   ```shell
   kubectl get -n kyma-system kyma
   ```

   If it doesn't have the ready state, troubleshoot it by checking the Pods under `eventing-manager-system` Namespace where the module is installed:

   ```shell
   kubectl get pods -n eventing-manager-system
   ```

### Uninstalling controller with [Kyma Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager/tree/main)

1. Delete eventing from `kyma` resource `spec.modules` `kubectl edit -n kyma-system kyma default-kyma`:

2. Check whether the `eventing` resource and module namespace were deleted:

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
