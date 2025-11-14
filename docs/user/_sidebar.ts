export default [
  { text: 'Event Naming and Cleanup', link: './evnt-event-names' },
  { text: 'Eventing Metrics', link: './evnt-eventing-metrics' },
  { text: 'Choose an Eventing Backend', link: './evnt-01-choose-backend' },
  { text: 'Configure SAP Event Mesh for Kyma Eventing', link: './tutorials/evnt-01-configure-event-mesh' },
  { text: 'Subscribe to Multiple Event Types', link: './tutorials/evnt-02-subs-with-multiple-filters' },
  { text: 'Manage Subscriber Workload with Max-In-Flight', link: './tutorials/evnt-04-change-max-in-flight-in-sub' },
  { text: 'Troubleshooting for the Eventing Module', link: './troubleshooting/README', collapsed: true, items: [
    { text: 'General Diagnostics: Event Not Delivered', link: './troubleshooting/evnt-01-eventing-troubleshooting' },
    { text: 'Subscriber Receives Irrelevant Events', link: './troubleshooting/evnt-02-subscriber-irrelevant-events' },
    { text: 'NATS Backend Storage Is Full', link: './troubleshooting/evnt-03-free-jetstream-storage' }
    ] },
  { text: 'Resources', link: './resources/README', collapsed: true, items: [
    { text: 'Eventing CR', link: './resources/eventing-cr' },
    { text: 'Subscription CR', link: './resources/subscription-cr' }
    ] }
];
