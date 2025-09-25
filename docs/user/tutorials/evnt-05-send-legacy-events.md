# Publish Legacy Events Using Kyma Eventing

<!-- I SUGGEST DELETING THIS TUTORIAL, IT ADDS LITTLE VALUE BEYOND STATING THAT LEGACY EVENTS ARE STILL SUPPORTED -->

While the Eventing module still supports sending and receiving of legacy events, it's recommended to use [CloudEvents](https://cloudevents.io/) instead.

## Prerequisites

- You have the Eventing module in your Kyma cluster.
- You have access to Kyma dashboard. Alternatively, if you prefer CLI, you need `kubectl` and `curl`.
- Optionally, you have the [CloudEvents Conformance Tool](https://github.com/cloudevents/conformance) for publishing events.
- You have an inline Function as event sink (see [Create and Modify an Inline Function](https://kyma-project.io/#/serverless-manager/user/tutorials/01-10-create-inline-function)).

> **TIP:** Use Istio sidecar proxies for reliability, observability, and security (see [Istio Service Mesh](https://kyma-project.io/#/istio/user/00-00-istio-sidecar-proxies)).

## Context


## Procedure

1. Create a Subscription that subscribe to events of the type `order.received.v1` and check that it is ready.
2. Port-forward the [Eventing Publisher Proxy](../evnt-architecture.md) Service to localhost, using port `3000`:
   ```bash
   kubectl -n kyma-system port-forward service/eventing-publisher-proxy 3000:80
   ```
3. Publish a legacy event to trigger your Function. In another terminal window, run:

   ```bash
   curl -v -X POST \
       --data @<(<<EOF
       {
           "event-type": "order.received",
           "event-type-version": "v1",
           "event-time": "2020-09-28T14:47:16.491Z",
           "data": {"orderCode":"3211213"}
       }
   EOF
       ) \
       -H "Content-Type: application/json" \
       http://localhost:3000/myapp/v1/events
   ```

> [!NOTE]
> If you want to use a Function to publish a CloudEvent, see the [Event object SDK specification](https://kyma-project.io/#/serverless-manager/user/technical-reference/07-70-function-specification?id=event-object-sdk).

4. To verify that the event was properly delivered, check the logs of the Function.

## Result

You see the received event in the logs:

```sh
Received event: { orderCode: '3211213' }
```
