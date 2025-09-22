# Eventing Module

Use the Eventing module to set up event-driven communication between applications in your Kyma cluster using a publish-subscribe model.

## What is Eventing?

The Eventing module enables event-driven communication between applications in your Kyma cluster. One application publishes an event (the "publisher"), and other applications ("subscribers") subscribe to receive it.

This decouples your services, as publishers and subscribers do not need to know about each other. They can communicate asynchronously and evolve independently.

## Features

The Eventing module provides the following features:

- Publish-subscribe (pub/sub) messaging: Decouples applications so you can build resilient and scalable event-driven systems.
- Flexible backend support: Use the default in-cluster NATS backend or configure SAP Event Mesh for enterprise messaging.
- Standardized event format: All events follow the [CloudEvents](https://cloudevents.io/) specification, ensuring a consistent and portable format.
- Automatic legacy event conversion: Converts older, non-standard Kyma event formats into valid CloudEvents automatically.
- At-least-once delivery: Ensures that each event is delivered at least one time when you use the NATS backend, preventing message loss during temporary failures.
- Declarative subscriptions: Manage event subscriptions with a simple [Subscription](./resources/subscription-cr.md) custom resource (CR).
- Built-in observability: Exposes key health and performance metrics in Prometheus format. You use the [Telemetry module](https://kyma-project.io/#/telemetry-manager/user/README) to collect, process, and forward these metrics to your preferred observability backend.

## Scope

The Eventing module focuses on in-cluster, asynchronous communication using the CloudEvents standard. It does not provide exactly-once delivery or long-term event storage. For cross-cluster or hybrid scenarios, you can configure SAP Event Mesh as the backend.

## Architecture

The Eventing module uses an [operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)-based architecture to manage the components that process and deliver events within the Kyma cluster.

[Diagram: Eventing flow]

The architecture consists of a control plane (Eventing Manager) that configures a data plane (Eventing Publisher Proxy and NATS backend) based on the Subscription custom resources you create.

### Eventing Manager

The Eventing Manager is the module's controller. It watches for Subscription custom resources and configures the underlying eventing infrastructure. 

When you create or update a Subscription, the Eventing Manager takes over the following tasks:

- Configures the NATS backend with the necessary streams and consumers.
- Ensures events are routed from the correct publisher to the specified subscriber (the "sink").
- Creates and manages Kubernetes resources, such as ConfigMaps, Secrets, Services, StatefulSets, DestinationRules, and Pod Disruption Budgets, adapting them to the desired state.

### Eventing Publisher Proxy

The Eventing Publisher Proxy provides a single, stable endpoint where your applications publish events using a standard [HTTP POST](https://www.w3schools.com/tags/ref_httpmethods.asp) request. This simplifies integration, as you can use common tools like curl or any standard HTTP client. 

The proxy is responsible for:

- Receiving inbound events from your applications.
- Converting events from legacy formats into the standard CloudEvents format.
- Forwarding the validated CloudEvents to the configured eventing backend for delivery.

### NATS Backend

By default, the Eventing module uses NATS as its in-cluster eventing backend. It uses the [NATS JetStream](https://docs.nats.io/) feature to provide persistence and guarantee at-least-once delivery. The NATS backend receives events from the Eventing Publisher Proxy and delivers them directly to the target subscribers, such as your microservices or Functions.

## API/Custom Resource Definitions

You configure the Eventing module by creating and applying Kubernetes Custom Resource Definitions (CRD), which extend the Kubernetes API with custom additions:

- To understand and configure the module's global settings, see the [Eventing CRD](./resources/eventing-cr.md).
- To create a subscriber, define a [Subscription CRD](./resources/subscription-cr.md).


## Resource Consumption

To learn more about the resources used by the Eventing module, see [Kyma Modules' Sizing](https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules-sizing?locale=en-US&version=Cloud).

<!-- ## Kyma Eventing Flow

Kyma Eventing follows the PubSub messaging pattern: Kyma publishes messages to a messaging backend, which filters these messages and sends them to interested subscribers. Kyma does not send messages directly to the subscribers as shown below:

![PubSub](../assets/evnt-pubsub.svg)

Eventing in Kyma from a user’s perspective works as follows:

- Offer an HTTP end point, for example a Function to receive the events.
- Specify the events the user is interested in using the Kyma [Subscription CR](./resources/evnt-cr-subscription.md).
- Send [CloudEvents](https://cloudevents.io/) or legacy events (deprecated) to the following HTTP end points on our [Eventing Publisher Proxy](https://github.com/kyma-project/eventing-publisher-proxy/blob/main/README.md) service.
  - `/publish` for CloudEvents.
  - `<application_name>/v1/events` for legacy events.

For more information, read [Eventing architecture](evnt-architecture.md).

## Glossary

- **Event Types**
  - `CloudEvents`: Events that conform to the [CloudEvents specification](https://cloudevents.io/) - a common specification for describing event data. The specification is currently under [CNCF](https://www.cncf.io/).
    <= explain this in **Event Naming and Formats**
  - `Legacy events` (deprecated): Events or messages published to Kyma that do not conform to the CloudEvents specification. All legacy events published to Kyma are converted to CloudEvents.
    <= explain this in **Event Naming and Formats**
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
