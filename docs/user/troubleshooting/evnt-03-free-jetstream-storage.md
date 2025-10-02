# NATS Backend Storage Is Full

## Symptom

- When you publish an event, the Eventing Publisher Proxy returns status `507 Insufficient Storage`.
- The Eventing Publisher Proxy logs show the error: `cannot send to stream: nats: maximum bytes exceeded`.

## Cause

In Kyma, the default retention policy for NATS JetStream is [Interest](https://docs.nats.io/using-nats/developer/develop_jetstream/model_deep_dive).
This retention policy keeps messages in the stream if they can't be delivered to the sink, as long as there are consumers in the stream that match the published event's subject.

If there are too many undelivered events, the message backlog grows until it fills the available storage. To prevent event loss, the backend stops receiving events, and no further events can be persisted to the stream.

This often happens if you delete a subscriber's deployment without first deleting the corresponding Subscription. 

## Solution

To free up storage and resume event flow, choose one of the following solutions:

- Temporarily reduce the events' publish rate until the events are delivered.

- If the subscriber is no longer needed, delete its Subscription resource. This action removes the NATS consumer and discards all pending messages for that subscription, freeing up space. The `Interest` retention policy specifies that events published to the subject are not kept in the stream if they don't match any consumer filter. 

- If a subscriber is unhealthy or slow, fix the underlying issue. When the subscriber comes back online and starts acknowledging messages, the stream will begin to clear. See Check the [NATS JetStream backend status](evnt-01-eventing-troubleshooting.md#step-6-check-nats-jetstream-status) and if [the sink is reachable and can accept the events](evnt-01-eventing-troubleshooting.md#step-5-check-if-the-subscription-sink-is-healthy).

- If the events' publish rate is very high (more than 1.5k events per second), the subscriber may not be able to keep up. Speed up the event dispatching by increasing the `maxInFlightMessages` configuration of the Subscription (default is set to 10).

- As a long-term solution, scale the NATS backend with additional replicas.
