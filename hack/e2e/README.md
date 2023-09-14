[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/eventing-manager)](https://api.reuse.software/info/github.com/kyma-project/eventing-manager)

# Eventing End-to-End Tests
Tests the end-to-end flow for eventing-manager.

## Overview

This test covers the end-to-end flow for Eventing. It is divided in three parts:
1. `setup` - prepares an Eventing CR and verifies that all the required resources are being provisioned by the eventing-manager.
2. `eventing` - prepares Subscription CR(s) and tests the end-to-end delivery of events for different event types.
3. `cleanup` - removes the test resources and namespaces from the cluster.

## Usage
You’ll need a Kubernetes cluster to run against. You can use [k3d](https://k3d.io/) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Pre-requisites

- [Go](https://go.dev/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- Access to Kubernetes cluster ([k3d](https://k3d.io/) / k8s) with eventing-manager deployed.
- For EventMesh backend, the following secrets should exist in the cluster:
  - Event Mesh credentials secret by name: `eventing-backend` in namespace: `kyma-system`.
  - OAuth client secret by name: `eventing-webhook-auth` in namespace: `kyma-system`.

### How to run end-to-end tests

1. Prepare the `.env` file based on the `.env.template`.

```
KUBECONFIG=                  # Kyma cluster kubeconfig file path
BACKEND_TYPE=NATS            # NATS or EventMesh
MANAGER_IMAGE=               # [Optional] Container image of eventing-manager
EVENTMESH_NAMESPACE=         # [Optional] Default is: "/default/sap.kyma/tunas-develop"
```

2. Run the following command to set up the environment variables in your system:
```bash
export $(xargs < .env)
```

3. Run the following make target to run the whole ent-to-end test suite.
```bash
make e2e
```

### Make targets

- `make e2e-setup` - creates an Eventing CR and verifies that all the required resources are being provisioned by the eventing-manager. If an Eventing CR already exists in the cluster, then it will only update the CR if the `spec.backend.type` is different from the backend configured for tests.
- `make e2e-eventing-setup` - prepares Subscription CR(s) and deploys a subscriber to be used in subscriptions as a sink. It will not update the Subscription CR(s) if they already exists.
- `make e2e-eventing` - tests the end-to-end delivery of events for different event types.
- `make e2e-eventing-cleanup` - removes the subscriber and the Subscription CR(s) from the cluster.
- `make e2e-cleanup` - removes all the test resources and namespaces from the cluster.
