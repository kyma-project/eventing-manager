# Eventing End-to-End Tests
Tests the end-to-end flow for Eventing manager.

## Overview

This test covers the end-to-end flow for Eventing. It tests the creation of an Eventing CR and verifies that all the required resources are provisioned by the Eventing manager. It also tests the end-to-end delivery of events for different event types.

## Usage

### Prerequisites

- [Go](https://go.dev/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- Access to Kubernetes cluster (k8s)
- Eventing manager deployed to the cluster without any Eventing CR.
- For EventMesh backend, the following secrets must exist in the cluster:
  - Event Mesh credentials secret by name: `eventing-backend` in namespace: `kyma-system`.
  - OAuth client secret by name: `eventing-webhook-auth` in namespace: `kyma-system`.

### How to run end-to-end tests

1. Prepare the `.env` file based on the `.env.template`.

   ```
   KUBECONFIG=                  # Kubernetes cluster kubeconfig file path
   BACKEND_TYPE=NATS            # NATS or EventMesh
   MANAGER_IMAGE=               # [Optional] Container image of eventing-manager
   EVENTMESH_NAMESPACE=         # [Optional] Default is: "/default/sap.kyma/tunas-develop"
   ```

2. To set up the environment variables in your system, run:

   ```bash
   export $(xargs < .env)
   ```

3. To run the whole end-to-end test suite, run:

   ```bash
   make e2e
   ```

### Run a single test phase

If you want to run only a single test phase, use the following make targets:

- `make e2e-setup` - creates an Eventing CR and verifies that all the required resources are provisioned by the eventing-manager. If an Eventing CR already exists in the cluster, then it will only update the CR if the `spec.backend.type` is different from the backend configured for tests.
- `make e2e-eventing-setup` - prepares Subscription CR(s) and deploys a subscriber to be used in subscriptions as a sink. It does not update the Subscription CR(s) if they already exist.
- `make e2e-eventing` - tests the end-to-end delivery of events for different event types.
- `make e2e-eventing-cleanup` - removes the subscriber and the Subscription CR(s) from the cluster.
- `make e2e-cleanup` - removes all the test resources and namespaces from the cluster.
