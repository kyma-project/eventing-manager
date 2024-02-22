# Installation and Uninstallation

There are several ways to install Eventing Manager.
For development, you must run some make targets beforehand.
For information about the prerequisites, refer to [Development](./development.md) and for a detailed guide to the development flow, visit [Governance](./governance.md).

## Run the Manager on a (k3d) Cluster Using a Docker Image

### Installation

1. Ensure you have a k3d cluster ready.

   ```sh
   k3d create cluster <clusterName>
   ```

    > **NOTE:** Alternatively to a k3d cluster, the Kubecontext can also point to any existing Kubernetes cluster.

2. Install the CRD of Eventing Manager.

   ```sh
   make install
   ```

3. Export the target registry.

   If you are using Docker, `<container-registry>` is your username.

      ```sh
      export IMG=<container-registry>/<image>:<tag>
      ```

4. Build and push your image to the registry.

   ```sh
   make docker-build docker-push IMG=$IMG
   ```

   > **NOTE:** For MacBook M1 devices, run:
   >
   >   ```sh
   >   make docker-buildx IMG=$IMG
   >   ```

5. Deploy the controller to the k3d cluster.

   ```sh
   make deploy IMG=$IMG
   ```

6. To start the reconciliation process, apply the Eventing custom resource (CR).
This step depends on your desired backend: NATS or EventMesh.

    - **Backend: NATS**

      For NATS Backend you can apply the default CR using the following command:  

      ```sh
      kubectl apply -f config/samples/default.yaml
      ```

      The `spec.backend.type` needs to be set to `NATS`. You can configure the backend using the `NATS` related field in `spec.backend.config`.

      ```sh
      spec:
        backend:
          type: NATS
          config:
            natsStreamStorageType: File
            natsStreamReplicas: 3
            natsStreamMaxSize: 700M
            natsMaxMsgsPerTopic: 1000000
      ```

    - **Backend: EventMesh**

      For EventMesh Backend you can apply the default CR using the following command:

      ```sh
      kubectl apply -f config/samples/default_eventmesh.yaml
      ```

      - `spec.backend.type`: set to `EventMesh`
      - `spec.backend.config.eventMeshSecret`: set it to the `<namespace>/<name>` where you applied the secret
      - `spec.backend.config.eventTypePrefix`: change to your desired value or leave as is
      - `spec.backend.config.domain`: set to the cluster public domain

      If the Kyma Kubernetes cluster is managed by Gardener, Eventing Manager reads the cluster public domain automatically from the ConfigMap `kube-system/shoot-info`.
      Otherwise, you need to additionally set `spec.backend.config.domain` in the configuration. 

      ```sh
      spec:
        backend:
          type: "EventMesh"
          config:
            eventMeshSecret: "<namespace>/<name>"
            eventTypePrefix: "<prefix>"
            domain: "<example.domain.com>"
      ```

7. Check the `status` section to see if deployment was successful.

   ```shell
   kubectl get <resourceName> -n <namespace> -o yaml
   ```

   >**Note:** Usually, the default values are as follows:
   >
   >  ```shell
   >  kubectl get eventings.operator.kyma-project.io -n kyma-system -o yaml
   >  ```

### Uninstallation

1. Remove the controller.

   ```sh
   make undeploy
   ```

2. Remove the resources.

   ```sh
   make uninstall
   ```

## Run Eventing Manager in a Cluster Using the Go Runtime Environment

### Installation

1. Ensure you have the Kubecontext pointing to an existing Kubernetes cluster.

2. Clone Eventing Manager project.

3. Download Go packages.

   ```sh
   go mod vendor && go mod tidy
   ```

4. Install the CRD of Eventing Manager.

   ```sh
   make install
   ```

5. Run Eventing Manager locally.

   ```sh
   make run
   ```

### Uninstallation

Remove the resources.

   ```sh
   make uninstall
   ```

## Run Eventing Manager Using Kyma’s Lifecycle Manager

[Kyma's Lifecycle Manager](https://github.com/kyma-project/lifecycle-manager) helps manage the lifecycle of each module in the cluster and can be used to install Eventing Manager.

To run Eventing Manager, follow the steps detailed in the [Lifecycle Manager documentation](https://github.com/kyma-project/lifecycle-manager/tree/main/docs).
