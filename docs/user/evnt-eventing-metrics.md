# Eventing Metrics

The Eventing module provides metrics that offer insights into its operational health, performance, and the flow of events within your Kyma cluster. The metrics follow [Prometheus naming convention](https://prometheus.io/docs/practices/naming/), and you can collect them with the Telemetry module's MetricPipeline.

> [!TIP]
> If you're using NATS as eventing backend, you might also want to monitor the eventing system's health. For details on the metrics emitted by the [Prometheus NATS Exporter](https://github.com/nats-io/prometheus-nats-exporter), see [NATS Monitoring](https://docs.nats.io/running-a-nats-service/configuration/monitoring#jetstream-information).

## Metrics Emitted by Eventing Publisher Proxy

The Eventing Publisher Proxy handles incoming events from your applications. Its metrics provide insights into event reception and forwarding.

| Metric                                         | Description                                                                      |
| ---------------------------------------------- | :------------------------------------------------------------------------------- |
| **eventing_epp_backend_duration_milliseconds** | The duration of sending events to the messaging server in milliseconds           |
| **eventing_epp_event_type_published_total**    | The total number of events published for a given eventTypeLabel                  |
| **eventing_epp_health**                        | The current health of the system. `1` indicates a healthy system                 |
| **eventing_epp_requests_duration_seconds**     | The duration of processing an incoming request (includes sending to the backend) |
| **eventing_epp_requests_total**                | The total number of requests                                                     |

## Metrics Emitted by Eventing Manager

The Eventing Manager configures event routing and delivers events to subscribers. Its metrics provide insights into subscription status and event dispatch.

| Metric                                                    | Description                                                                                                                 |
| --------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------- |
| **eventing_ec_event_type_subscribed_total**               | The total number of eventTypes subscribed using the Subscription CRD                                                        |
| **eventing_ec_health**                                    | The current health of the system. `1` indicates a healthy system                                                            |
| **eventing_ec_nats_delivery_per_subscription_total**      | The total number of dispatched events per subscription                                                                      |
| **eventing_ec_nats_subscriber_dispatch_duration_seconds** | The duration of sending an incoming NATS message to the subscriber (not including processing the message in the dispatcher) |
| **eventing_ec_subscription_status**                       | The status of a subscription. `1` indicates the subscription is marked as ready                                             |
