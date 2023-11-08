# Testing

This document provides an overview of the testing activities used in this project.

## Testing Levels

| Test suite | Testing level | Purpose                                                                                                                                                                                               |
|------------|---------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Unit       | Unit          | This test suite tests the units in isolation. It assesses the implementation correctness of the unit's business logic.                                                                                |
| Env-tests  | Integration   | This test suite tests the behaviour of the Eventing Manager in integration with a Kubernetes API server replaced with a test double. It assesses the integration correctness of the Eventing Manager. |
| E2E        | Acceptance    | This test suite tests the usability scenarios of the Eventing Manager in a cluster. It assesses the functional correctness of the Eventing Manager.                                                   |

> **NOTE:** The validation and defaulting rules are tested within the integration tests.

### Unit tests and Env-tests

To run the unit and integration tests, the following command needs to be executed. If necessary, the needed binaries for the integration tests are downloaded to `./bin`.
Further information about integration tests can be found in the [Kubebuilder book](https://book.kubebuilder.io/reference/envtest.html).

   ```sh
   make test-only
   ```

If changes to the source code were made, or if this is your first time to execute the tests, the following command ensures that all necessary tooling is executed before running the unit and integration tests:

   ```sh
   make test
   ``` 

### E2E tests

Because E2E tests need a Kubernetes cluster to run on, they are separate from the remaining tests.

1. Ensure you have the Kubecontext pointing to an existing Kubernetes cluster and the Eventing Manager has been deployed.

   > Note: Creating an Eventing CR is optional. If an Eventing CR already exists, the test will update the CR to meet the requirements of the test.

2. Export the following ENV variables.

   ```sh
   export BACKEND_TYPE="NATS"         # if using NATS Backend
   export BACKEND_Type="EventMesh"    # if using EventMesh Backend
   ```

2. Execute the whole E2E test suite.

   ```sh
   make e2e
   ```

The E2E test consists of four consecutive steps. If desired, you can run them individually. Further information can be found in the [E2E documentation](https://github.com/kyma-project/eventing-manager/blob/main/hack/e2e/README.md).



## CI/CD

This project uses [Prow](https://docs.prow.k8s.io/docs/) and [GitHub Actions](https://docs.github.com/en/actions) as part of the development cycle.
The aim is to verify the functional correctness of the Eventing Manager.

### Prow jobs

The Prow jobs that cover code of this repository reside in [their own repository](https://github.com/kyma-project/test-infra/tree/main/prow/jobs/kyma-project/eventing-manager).
Presubmit jobs run on PRs and are marked with the prefix `pull`. Postsubmit jobs run on main after a PR was merged and carry the prefix `post`.

For more information on execution details of each job, refer to their `description` field and the `command` and `args` fields.
Alternatively, you can access this information from your PR by inspecting the details to the job and viewing the Prow job `.yaml` file.

### GitHub Actions

GitHub Actions reside [within this module repository](https://github.com/kyma-project/eventing-manager/tree/main/.github/workflows).
Pre- and postsubmit actions follow the same naming conventions as Prow jobs.

The [Actions overview](https://github.com/kyma-project/eventing-manager/actions/), shows all the existing workflows and their execution details. Here, you can also trigger a re-run of an action.
