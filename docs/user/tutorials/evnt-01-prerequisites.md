# Eventing Tutorial Prerequisites

Browse the tutorials for Eventing to learn how to use it step-by-step in different scenarios.

To perform the steps described in the Eventing tutorials, you need the following: 

1. Kyma cluster provisioned with Eventing module added. See [Quick Install](https://kyma-project.io/#/02-get-started/01-quick-install).
2. (Optional) [Kyma dashboard](https://kyma-project.io/#/01-overview/ui/README?id=kyma-dashboard) deployed on the Kyma cluster. To access the Kyma dashboard, run:

   ```bash
   kyma dashboard
   ```

   Alternatively, you can just use the `kubectl` CLI instead.

3. (Optional) [CloudEvents Conformance Tool](https://github.com/cloudevents/conformance) for publishing events.

   ```bash
   go install github.com/cloudevents/conformance/cmd/cloudevents@latest
   ```

   Alternatively, you can just use `curl` to publish events.
