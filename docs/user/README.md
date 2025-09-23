# Eventing Module

Use the Eventing module to set up event-driven communication between applications in your Kyma cluster using a publish-subscribe model.

## What is Eventing?

The Eventing module enables event-driven communication between applications in your Kyma cluster. One application publishes an event (the "publisher"), and other applications ("subscribers") subscribe to receive it.

This decouples your services, as publishers and subscribers do not need to know about each other. They can communicate asynchronously and evolve independently.

## Features

The Eventing module provides the following features:

- Publish-subscribe (pub/sub) messaging: Decouples applications so you can build resilient and scalable event-driven systems.
- Flexible backend support: Use the default in-cluster NATS backend (see [NATS module](https://kyma-project.io/#/nats-manager/user/README)) or configure SAP Event Mesh (see [SAP Event Mesh](https://help.sap.com/docs/event-mesh/event-mesh/what-is-sap-event-mesh?version=Cloud&locale=en-US)) for enterprise messaging.
- Standardized event format: All events follow the [CloudEvents](https://cloudevents.io/) specification, ensuring a consistent and portable format.
- Automatic legacy event conversion: Converts older, non-standard Kyma event formats into valid CloudEvents automatically.
- At-least-once delivery: Ensures that each event is delivered at least one time when you use the NATS backend, preventing message loss during temporary failures.
- Declarative subscriptions: Manage event subscriptions with a simple [Subscription](./resources/subscription-cr.md) custom resource (CR).
- Built-in observability: Exposes key health and performance metrics in Prometheus format. You use the [Telemetry module](https://kyma-project.io/#/telemetry-manager/user/README) to collect, process, and forward these metrics to your preferred observability backend.

## Scope

The Eventing module focuses on in-cluster, asynchronous communication using the CloudEvents standard. It does not provide exactly-once delivery or long-term event storage. For cross-cluster or hybrid scenarios, you can configure SAP Event Mesh as the backend.

## Architecture

The Eventing module uses an [operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)-based architecture to manage the components that process and deliver events within the Kyma cluster. It consists of a control plane and a data plane.

- Control Plane: The Eventing Manager watches for Subscription custom resources and configures the eventing infrastructure.
- Data Plane: The Eventing Publisher Proxy receives events, and the configured backend (NATS or SAP Event Mesh) delivers them.

**DIAGRAM TO BE UPDATED!**
![Eventing flow](../assets/evnt-architecture.svg)

<!-- 
1. The Eventing Manager watches the Subscription custom resource. It detects if there are any new incoming events.

2. The Eventing Manager creates an infrastructure for the NATS server.

3. An event source publishes events to the Eventing Publisher Proxy.

4. The Eventing Publisher Proxy sends events to the NATS server.

5. The NATS server dispatches events to the Eventing Manager.

6. The Eventing Manager dispatches events to subscribers (microservices or Functions).-->

The Eventing module processes events through a series of steps:

1. Application publishes event: Your application sends an event to the Eventing Publisher Proxy.
2. Proxy processes and forwards: The Eventing Publisher Proxy receives, validates, converts the event to CloudEvents format, and forwards it to the configured eventing backend.
3. Backend processes event: The selected backend (NATS or SAP Event Mesh) processes the event.
4. Backend dispatches to Eventing Manager: The backend dispatches the event to the Eventing Manager.
5. Eventing Manager delivers to subscriber: The Eventing Manager delivers the event to the appropriate subscriber (microservice or Function) based on your Subscription configuration.

### Eventing Manager

The Eventing Manager is the module's controller. It watches for Subscription custom resources and configures the underlying eventing infrastructure.

When you create or update a Subscription, the Eventing Manager performs the following tasks:

- Configures the selected eventing backend to manage event streams and consumers for subscriptions.
- Ensures events are routed from the correct publisher to the specified subscriber (the "sink").
- Creates and manages Kubernetes resources, such as ConfigMaps, Secrets, Services, StatefulSets, DestinationRules, and Pod Disruption Budgets, adapting them to the desired state.

### Eventing Publisher Proxy

The [Eventing Publisher Proxy](https://github.com/kyma-project/eventing-publisher-proxy) provides a single, stable endpoint where your applications publish events using a standard [HTTP POST](https://www.w3schools.com/tags/ref_httpmethods.asp) request (endpoint: `/publish` for CloudEvents and `<application_name>/v1/events` for legacy events). This simplifies integration, as you can use common tools like curl or any standard HTTP client. 

The proxy performs the following tasks:

- Receives inbound events from your applications.
- Converts events from legacy formats into the standard CloudEvents format.
- Forwards validated CloudEvents to the configured eventing backend for delivery.

## API/Custom Resource Definitions

You configure the Eventing module by creating and applying Kubernetes Custom Resource Definitions (CRD), which extend the Kubernetes API with custom additions:

- To understand and configure the module's global settings, see the [Eventing CRD](./resources/eventing-cr.md).
- To create a subscriber, define a [Subscription CRD](./resources/subscription-cr.md). You cannot delete the Eventing module as long as Subscription CRs exist.

## Resource Consumption

To learn more about the resources used by the Eventing module, see [Kyma Modules' Sizing](https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules-sizing?locale=en-US&version=Cloud).

<!-- 
## Glossary
- **Streams and Consumers**
  - `Streams`: A stream stores messages for the published events. Kyma uses only one stream, with _**file**_ storage, for all the events. You can configure the retention and delivery policies for the stream, depending on the use case.
    <= These terms describe the inner workings of the NATS backend. We should update the NATS Backend description in the Architecture topic to explain that a Kyma Subscription maps to a NATS Consumer, and that all events are stored in a single NATS Stream. This connects the user's declarative Subscription to the underlying system behavior.
  - `Consumers`: A consumer reads or consumes the messages from the stream. Kyma Subscription creates one consumer for each specified filter. Kyma uses push-based consumers.
      <= These terms describe the inner workings of the NATS backend. We should update the NATS Backend description in the Architecture topic to explain that a Kyma Subscription maps to a NATS Consumer, and that all events are stored in a single NATS Stream. This connects the user's declarative Subscription to the underlying system behavior.
- **Delivery Guarantees**
  - `at least once` delivery: With NATS JetStream, Kyma ensures that for each event published, all the subscribers subscribed to that event receive the event at least once.
    <= Mentioned in Features
  - `max bytes and discard policy`: NATS JetStream uses these configurations to ensure that no messages are lost when the storage is almost full. By default, Kyma ensures that no new messages are accepted when the storage reaches 90% capacity.
    <= should be mentioned in [Resource Consumption](https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules-sizing?locale=en-US&version=Cloud).
     -->
